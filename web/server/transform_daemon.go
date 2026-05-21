package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syredb/database"
	"syredb/service"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const pollInteraval = 1500 * time.Millisecond

func NewTransformDaemon(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DBConnection,
	app_service *service.AppService,
	data_service *service.DataService,
) *TransformDaemon {
	return &TransformDaemon{
		ctx:          ctx,
		logger:       logger,
		db:           db,
		app_service:  app_service,
		data_service: data_service,
	}
}

type TransformDaemon struct {
	ctx          context.Context
	logger       *slog.Logger
	db           *database.DBConnection
	app_service  *service.AppService
	data_service *service.DataService
}

func (d *TransformDaemon) Start(ctx context.Context) error {
	last_poll := time.Now()
	for {
		sleep := pollInteraval - time.Since(last_poll)
		time.Sleep(sleep)
		last_poll = time.Now()

		jobs, err := d.pollPending()
		if err != nil {
			d.logger.With(
				"error", err,
			).Error("could not poll pending data type transform jobs")
			continue
		}

		var transform_ids []uuid.UUID
		for _, job := range jobs {
			if !slices.Contains(transform_ids, job.Transform) {
				transform_ids = append(transform_ids, job.Transform)
			}
		}
		transforms_info, err := d.transformsById(transform_ids)
		if err != nil {
			continue
		}

		for _, job := range jobs {
			transform_idx := slices.IndexFunc(transforms_info, func(info transformInfo) bool {
				return info.Id == job.Transform
			})
			if transform_idx < 0 {
				d.logger.With("transform", job.Transform).Error("invalid transform")
				panic("invalid transform")
			}
			transform := transforms_info[transform_idx]

			go d.runTransform(
				job.Id,
				transform,
				job.Payload,
			)
		}
	}
}

type transformJobStatus string

const (
	transformJobStatusPending   transformJobStatus = "pending"
	transformJobStatusRunning   transformJobStatus = "running"
	transformJobStatusCompleted transformJobStatus = "completed"
	transformJobStatusFailed    transformJobStatus = "failed"
)

type transformJobRx struct {
	Id        uuid.UUID `db:"_id"`
	Transform uuid.UUID `db:"_transform"`
	Payload   uuid.UUID `db:"_payload"`
}

func (d *TransformDaemon) pollPending() ([]transformJobRx, error) {
	query := fmt.Sprintf(
		"SELECT _id, _transform, _payload FROM _data_type_transform_queue_ WHERE status='%s'",
		transformJobStatusPending,
	)
	rows, _ := d.db.Conn.Query(d.ctx, query)
	jobs, err := pgx.CollectRows(rows, pgx.RowToStructByName[transformJobRx])
	if err != nil {
		d.logger.With("error", err).Error("could not poll data type transform queue")
		return nil, err
	}

	return jobs, nil
}

type dataTypeTransformCmdRx struct {
	Id      uuid.UUID `db:"_id"`
	Creator uuid.UUID `db:"_creator"`
	Path    string    `db:"_path"`
	Cmd     string    `db:"_cmd"`
	Args    []string  `db:"_args"`
}

type transformInfoRx struct {
	Id          uuid.UUID `db:"_id"`
	Source      uuid.UUID `db:"_source"`
	Destination uuid.UUID `db:"_destination"`
	Cmd         uuid.UUID `db:"cmd"`
}

// transformInfo
// `.Source` and `.Destination` are `dataTypeInfoInternal` or `dataTypeInfoExternal`.
type transformInfo struct {
	Id          uuid.UUID
	Cmd         dataTypeTransformCmdRx
	Source      service.DataType
	Destination service.DataType
}

func (d *TransformDaemon) transformsById(transforms []uuid.UUID) ([]transformInfo, error) {
	// TODO
	transform_query :=
		`SELECT _id, _source, _destination, cmd FROM data_type_transform_
		WHERE _id=ANY($1)`
	rows, _ := d.db.Conn.Query(d.ctx, transform_query, transforms)
	transform_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[transformInfoRx])
	if err != nil {
		d.logger.With(
			"error", err,
			"transforms", transforms,
		).Error("could not get data type transforms")
		return nil, err
	}

	data_type_ids := make([]uuid.UUID, 0, len(transform_rxs)*2)
	cmd_ids := make([]uuid.UUID, len(transform_rxs))
	for idx, rx := range transform_rxs {
		if !slices.Contains(data_type_ids, rx.Source) {
			data_type_ids = append(data_type_ids, rx.Source)
		}
		if !slices.Contains(data_type_ids, rx.Destination) {
			data_type_ids = append(data_type_ids, rx.Destination)
		}

		cmd_ids[idx] = rx.Cmd
	}

	types, err := d.dataTypesById(data_type_ids)
	if err != nil {
		d.logger.With(
			"error", err,
			"data types", data_type_ids,
		).Error("could not get data types")
		return nil, err
	}

	cmd_query :=
		`SELECT _id, _creator, _path, _cmd, _args FROM data_type_transform_cmd_
		WHERE _id=ANY($1)`
	rows, _ = d.db.Conn.Query(d.ctx, cmd_query, cmd_ids)
	cmds, err := pgx.CollectRows(rows, pgx.RowToStructByName[dataTypeTransformCmdRx])
	if err != nil {
		d.logger.With(
			"error", err,
			"commands", cmd_ids,
		).Error("could not get data type transforms commands")
		return nil, err
	}

	transforms_info := make([]transformInfo, len(transforms))
	for idx, transform := range transform_rxs {
		cmd_idx := slices.IndexFunc(cmds, func(cmd dataTypeTransformCmdRx) bool {
			return cmd.Id == transform.Cmd
		})
		if cmd_idx < 0 {
			panic("invalid data type transform command")
		}

		src_idx := slices.IndexFunc(types, func(info service.DataType) bool {
			switch info.DataStorage() {
			case service.DataStorageInternal:
				return info.(*dataTypeInfoInternal).Id == transform.Source
			case service.DataStorageExternal:
				return info.(*dataTypeInfoExternal).Id == transform.Source
			default:
				panic("unexpected service.DataStorage")
			}
		})
		if src_idx < 0 {
			panic("invalid data type transform source")
		}

		dst_idx := slices.IndexFunc(types, func(info service.DataType) bool {
			switch info.DataStorage() {
			case service.DataStorageInternal:
				return info.(*dataTypeInfoInternal).Id == transform.Destination
			case service.DataStorageExternal:
				return info.(*dataTypeInfoExternal).Id == transform.Destination
			default:
				panic("unexpected service.DataStorage")
			}
		})
		if dst_idx < 0 {
			panic("invalid data type transform destination")
		}

		transforms_info[idx] = transformInfo{
			Id:          transform.Id,
			Cmd:         cmds[cmd_idx],
			Source:      types[src_idx],
			Destination: types[dst_idx],
		}
	}

	return transforms_info, nil
}

type dataTypeInfoBasic struct {
	Id      uuid.UUID           `db:"_id"`
	Storage service.DataStorage `db:"_storage"`
}

type dataTypeInfoInternal struct {
	Id          uuid.UUID
	Cardinality service.DataSchemaCardinality
	SchemaId    uuid.UUID
	Schema      []service.DataSchemaField
}

func (d *dataTypeInfoInternal) DataStorage() service.DataStorage {
	return service.DataStorageInternal
}

type dataTypeInfoExternal struct {
	Id      uuid.UUID
	Sources []dataSourceInfo
}

func (d *dataTypeInfoExternal) DataStorage() service.DataStorage {
	return service.DataStorageExternal
}

// dataTypesById returns `dataTypeInfoInternal` and `dataTypeInfoExternal`.
func (d *TransformDaemon) dataTypesById(type_ids []uuid.UUID) ([]service.DataType, error) {
	query := "SELECT _id, _storage FROM data_type_ WHERE _id=ANY($1)"
	rows, _ := d.db.Conn.Query(d.ctx, query, type_ids)
	types_info, err := pgx.CollectRows(rows, pgx.RowToStructByName[dataTypeInfoBasic])
	if err != nil {
		d.logger.With(
			"error", err,
			"data types", type_ids,
		).Error("could not get data types")
		return nil, err
	}

	internal_storage_ids := make([]uuid.UUID, 0, len(type_ids))
	external_storage_ids := make([]uuid.UUID, 0, len(type_ids))
	for _, data := range types_info {
		switch data.Storage {
		case service.DataStorageInternal:
			internal_storage_ids = append(internal_storage_ids, data.Id)
		case service.DataStorageExternal:
			external_storage_ids = append(external_storage_ids, data.Id)
		default:
			panic(fmt.Sprintf("unexpected service.DataStorage: %#v", data.Storage))
		}
	}

	schemas, err := d.dataSchemasByTypeId(internal_storage_ids)
	if err != nil {
		d.logger.With(
			"error", err,
			"data types", internal_storage_ids,
		).Error("could not get data type schemas")
		return nil, err
	}

	sources, err := d.dataSourcesByTypeId(external_storage_ids)
	if err != nil {
		d.logger.With(
			"error", err,
			"data types", external_storage_ids,
		).Error("could not get data type sources")
		return nil, err
	}

	types := make([]service.DataType, len(type_ids))
	for idx, id := range type_ids {
		if slices.Contains(internal_storage_ids, id) {
			sidx := slices.IndexFunc(schemas, func(schema schemaInfo) bool {
				return schema.DataType == id
			})
			if sidx < 0 {
				d.logger.With(
					"data type", id,
					"schemas", schemas,
				).Error("data type schema not found")
				panic("invalid data type")
			}

			types[idx] = &dataTypeInfoInternal{
				Id:          id,
				Cardinality: schemas[sidx].Cardinality,
				SchemaId:    schemas[sidx].Id,
				Schema:      schemas[sidx].Schema,
			}
		} else if slices.Contains(external_storage_ids, id) {
			sidx := slices.IndexFunc(sources, func(source dataSourceInfo) bool {
				return source.DataType == id
			})
			if sidx < 0 {
				panic("invalid data type")
			}

			var type_sources []dataSourceInfo
			for _, source := range sources {
				if source.DataType == id {
					type_sources = append(type_sources, source)
				}
			}
			types[idx] = &dataTypeInfoExternal{
				Id:      id,
				Sources: type_sources,
			}
		}
	}

	return types, nil
}

type dataTypeSchemaInfo struct {
	DataType    uuid.UUID                     `db:"_data_type"`
	Cardinality service.DataSchemaCardinality `db:"_cardinality"`
	Schema      uuid.UUID                     `db:"_schema"`
}

type dataSchemaFieldRx struct {
	Id          uuid.UUID         `db:"_id"`
	Label       string            `db:"_label"`
	Dtype       service.ValueType `db:"_dtype"`
	Index       uint              `db:"index"`
	Description string            `db:"description"`
}

type schemaInfo struct {
	DataType    uuid.UUID
	Id          uuid.UUID
	Cardinality service.DataSchemaCardinality
	Schema      []service.DataSchemaField
}

func (d *TransformDaemon) dataSchemasByTypeId(types []uuid.UUID) ([]schemaInfo, error) {
	data_type_query :=
		`SELECT d._data_type, d._schema, s._cardinality
		FROM data_type_schema_ d JOIN data_schema_ s ON d._schema=s._id
		WHERE d._data_type=ANY($1)`
	rows, _ := d.db.Conn.Query(d.ctx, data_type_query, types)
	dt_schemas, err := pgx.CollectRows(rows, pgx.RowToStructByName[dataTypeSchemaInfo])
	if err != nil {
		d.logger.With(
			"error", err,
			"data types", types,
		).Error("could not get data types schema")
		return nil, err
	}

	schema_ids := make([]uuid.UUID, 0, len(dt_schemas))
	for _, schema := range dt_schemas {
		if !slices.Contains(schema_ids, schema.Schema) {
			schema_ids = append(schema_ids, schema.Schema)
		}
	}

	schema_query :=
		`SELECT _id, _label, _dtype, index, description FROM data_schema_field_
		WHERE _id=ANY($1)`
	rows, _ = d.db.Conn.Query(d.ctx, schema_query, schema_ids)
	fields, err := pgx.CollectRows(rows, pgx.RowToStructByName[dataSchemaFieldRx])
	if err != nil {
		d.logger.With(
			"error", err,
			"schemas", schema_ids,
		).Error("could not get data types schema")
		return nil, err
	}

	schema_fields := make(map[uuid.UUID][]service.DataSchemaField, len(schema_ids))
	for _, field := range fields {
		info := service.DataSchemaField{
			Label:       field.Label,
			DType:       field.Dtype,
			Index:       field.Index,
			Description: field.Description,
		}

		_, exists := schema_fields[field.Id]
		if exists {
			schema_fields[field.Id] = append(schema_fields[field.Id], info)
		} else {
			schema_fields[field.Id] = []service.DataSchemaField{info}
		}
	}

	schemas := make([]schemaInfo, len(dt_schemas))
	for idx, dt_schema := range dt_schemas {
		schemas[idx].DataType = dt_schema.DataType
		schemas[idx].Id = dt_schema.Schema
		schemas[idx].Cardinality = dt_schema.Cardinality
		schemas[idx].Schema = schema_fields[dt_schema.Schema]
	}

	return schemas, nil
}

type dataSourceInfo struct {
	Id          uuid.UUID                     `db:"_id"`
	DataType    uuid.UUID                     `db:"_data_type"`
	Label       string                        `db:"_label"`
	Required    bool                          `db:"_required"`
	Cardinality service.DataSourceCardinality `db:"_cardinality"`
	ExtFilter   []string                      `db:"ext_filter"`
}

func (d *TransformDaemon) dataSourcesByTypeId(types []uuid.UUID) ([]dataSourceInfo, error) {
	query :=
		`SELECT _id, _data_type, _label, _required, _cardinality, ext_filter 
		FROM data_type_source_ WHERE _data_type=ANY($1)`
	rows, _ := d.db.Conn.Query(d.ctx, query, types)
	sources, err := pgx.CollectRows(rows, pgx.RowToStructByName[dataSourceInfo])
	if err != nil {
		d.logger.With(
			"error", err,
			"data types", types,
		).Error("could not get data types sources")
		return nil, err
	}

	return sources, nil
}

type DataSchemaInfo struct {
	Id     uuid.UUID
	Schema []service.DataSchemaField
}

func (d *TransformDaemon) getDataSchemasById(data_schemas []uuid.UUID) ([]DataSchemaInfo, error) {
	// TODO
	return nil, nil
	// query := "SELECT _id, _schema FROM data_schema_ WHERE _id=ANY($1)"
	// rows, _ := d.db.Conn.Query(d.ctx, query, data_schemas)
	// schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchemaInfo, error) {
	// 	var schema DataSchemaInfo
	// 	err := row.Scan(&schema.Id, &schema.Schema)
	// 	return schema, err
	// })
	// if err != nil {
	// 	d.logger.With(
	// 		"error", err,
	// 		"schemas", data_schemas,
	// 	).Error("could not get data schemas")
	// 	return nil, err
	// }

	// return schemas, nil
}

func (d *TransformDaemon) runTransform(
	job uuid.UUID,
	transform transformInfo,
	payload uuid.UUID,
) {
	start_query := fmt.Sprintf(
		"UPDATE _data_type_transform_queue_ SET status='%s', started=$1 WHERE _id=$2",
		transformJobStatusRunning,
	)
	_, err := d.db.Conn.Exec(
		d.ctx,
		start_query,
		time.Now().Format(time.RFC3339),
		job,
	)
	if err != nil {
		d.logger.With(
			"error", err,
			"job", job,
		).Error("could not update job on start")
		return
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		d.logger.With(
			"error", err,
			"job", job,
		).Error("could not open listening port")
		return
	}

	data, err := d.createTransformData(payload, transform)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not create transform data")

		finish_time := time.Now().Format(time.RFC3339)
		err_query := fmt.Sprintf(
			`UPDATE _data_type_transform_queue_
			SET status='%s', finished=$2, error=$3
			WHERE _id=$1`,
			transformJobStatusFailed,
		)
		_, err = d.db.Conn.Exec(
			d.ctx,
			err_query,
			job,
			finish_time,
			fmt.Sprintf("could not create data transform file: %s", err),
		)
		if err != nil {
			d.logger.With(
				"error", err,
				"job", job,
			).Error("could not update transform job status")
		}
		return
	}

	port := listener.Addr().(*net.TCPAddr).Port
	go d.transformListener(listener, job, payload, transform, data)

	cmd := exec.Command(
		transform.Cmd.Cmd,
		transform.Cmd.Path,
		job.String(),
		strconv.Itoa(port),
	)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stderr = &stdout
	cmd.Stderr = &stderr

	d.logger.With("job", job).Debug("running transform command")
	cmd_err := cmd.Run()
	d.logger.With(
		"job", job,
		"stdout", stdout.String(),
		"stderr", stderr.String(),
	).Debug("transform command complete")

	finish_time := time.Now().Format(time.RFC3339)
	err = listener.Close()
	if err != nil {
		d.logger.With(
			"error", err,
			"job", job,
		).Error("could not close transform listener")
	}
	for _, file := range data.Files {
		err = os.Remove(file.Name())
		if err != nil {
			d.logger.With(
				"error", err,
				"file", file.Name(),
			).Error("could not remove file")
		}
	}

	if cmd_err != nil {
		d.logger.With(
			"error", cmd_err,
			"job", job,
			"cmd", cmd.String(),
			"stderr", stderr.String(),
		).Error("error running command")
		// SAFETY: Logging occurs in `setJobError` function
		// and nothing to do here if the status couldn't be updated
		_ = d.setJobError(job, finish_time, cmd_err)
	} else {
		ok_query := fmt.Sprintf(
			`UPDATE _data_type_transform_queue_ 
			SET status='%s', finished=$2, error=''
			WHERE _id=$1`,
			transformJobStatusCompleted,
		)
		_, err = d.db.Conn.Exec(d.ctx, ok_query, job, finish_time)
		if err != nil {
			d.logger.With(
				"error", err,
				"job", job,
			).Error("could not update transform job status")
		}
	}
}

// readMessage reads a framed JSON message from `reader`.
func readMessage(reader io.Reader) (transformMessage, error) {
	var length uint64
	err := binary.Read(reader, binary.LittleEndian, &length)
	if err != nil {
		return transformMessage{}, err
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return transformMessage{}, err
	}

	var msg transformMessage
	err = json.Unmarshal(buf, &msg)
	if err != nil {
		return transformMessage{}, err
	}

	return msg, nil
}

type outputData struct {
	Token      uuid.UUID       `json:"token"`
	Properties []PropertyValue `json:"properties"`
	Tags       []string        `json:"tags"`
	Values     map[string]any  `json:"values"`
}

// readData reads output data as a framed JSON message from `reader`.
func readData(reader io.Reader) (outputData, error) {
	var length uint64
	err := binary.Read(reader, binary.LittleEndian, &length)
	if err != nil {
		return outputData{}, err
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return outputData{}, err
	}

	var data outputData
	err = json.Unmarshal(buf, &data)
	if err != nil {
		return outputData{}, err
	}

	return data, nil
}

// writeMessage writes a framed JSON message of `value`.
func writeMessage(writer io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	length := uint64(len(data))
	err = binary.Write(writer, binary.LittleEndian, length)
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}

type transformFn string

const (
	fnGetData        transformFn = "get_data"
	fnValuesCsv      transformFn = "values_as_csv"     // only valid for input data with internal storage
	fnValuesFeather  transformFn = "values_as_feather" // only valid for input data with internal storage
	fnValuesMap      transformFn = "values_as_map"     // only valid for input data with internal storage
	fnOutputDataInfo transformFn = "output_data_info"
	fnSaveData       transformFn = "save_data"
)

type transformMessage struct {
	Token uuid.UUID   `json:"token"`
	Fn    transformFn `json:"fn"`
}

// `token` is the job id.
// `payload` is the id of the input data.
func (d *TransformDaemon) transformListener(
	listener net.Listener,
	token uuid.UUID,
	payload uuid.UUID,
	info transformInfo,
	data transformData,
) error {
	d.logger.With("job", token).Debug("transform listener launched")
	conn, err := listener.Accept()
	if err != nil {
		d.logger.With(
			"error", err,
			"job", token,
		).Error("could not connect listener")
		return err
	}
	defer conn.Close()

	var files []*os.File
	for {
		msg, err := readMessage(conn)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				d.logger.With(
					"error", err,
					"job", token,
				).Debug("job complete")
				break
			} else {
				d.logger.With(
					"error", err,
					"job", token,
				).Error("could not read message")
				continue
			}
		}
		d.logger.With(
			"job", token,
			"message", msg,
		).Debug("message received")

		if msg.Token != token {
			d.logger.With(
				"token", token,
				"message", msg,
			).Warn("invalid token, message ignored")
			continue
		}

		switch msg.Fn {
		case fnGetData:
			err = d.transformListenerFnGetData(conn, data)
			if err != nil {
				d.logger.With(
					"error", err,
					"job", token,
				).Error("could not get data")
				return err
			}
		case fnValuesCsv:
			if data.InputStorage != service.DataStorageInternal {
				d.logger.With(
					"job", token,
				).Warn("input data with non-internally data storage requested values")
				writeMessage(conn, "")
				continue
			}

			file, err := d.transformListenerFnGetValues(conn, payload, ValuesFileFormatCsv)
			if err != nil {
				d.logger.With(
					"error", err,
					"job", token,
				).Error("could not get values")
				return err
			}
			files = append(files, file)
			err = writeMessage(conn, file.Name())
			if err != nil {
				d.logger.With(
					"error", err,
					"data", data,
				).Error("could not write response")
				return err
			}
		case fnValuesFeather:
			if data.InputStorage != service.DataStorageInternal {
				d.logger.With(
					"job", token,
				).Warn("input data with non-internally data storage requested values")
				writeMessage(conn, "")
				continue
			}

			file, err := d.transformListenerFnGetValues(conn, payload, ValuesFileFormatFeather)
			if err != nil {
				d.logger.With(
					"error", err,
					"job", token,
				).Error("could not get values")
				return err
			}
			files = append(files, file)
			err = writeMessage(conn, file.Name())
			if err != nil {
				d.logger.With(
					"error", err,
					"data", data,
				).Error("could not write response")
				return err
			}
		case fnValuesMap:
			if data.InputStorage != service.DataStorageInternal {
				d.logger.With(
					"job", token,
				).Warn("input data with non-internally data storage requested values")
				writeMessage(conn, "")
				continue
			}

			file, err := d.transformListenerFnGetValues(conn, payload, ValuesFileFormatJson)
			if err != nil {
				d.logger.With(
					"error", err,
					"job", token,
				).Error("could not get values")
				return err
			}
			files = append(files, file)
			err = writeMessage(conn, file.Name())
			if err != nil {
				d.logger.With(
					"error", err,
					"data", data,
				).Error("could not write response")
				return err
			}
		case fnOutputDataInfo:
			err = d.transformListenerFnGetOutputDataInfo(conn, data)
			if err != nil {
				d.logger.With(
					"error", err,
					"job", token,
				).Error("could not get data")
				return err
			}
		case fnSaveData:
			err = d.transformListenerFnSaveData(conn, token, payload, info)
			if err != nil {
				d.logger.With(
					"error", err,
					"job", token,
				).Error("could not save data")
				return err
			}
		default:
			panic("invalid function")
		}
	}

	for _, file := range files {
		err = file.Close()
		if err != nil {
			d.logger.With(
				"error", err,
				"job", token,
				"file", file.Name(),
			).Error("could not close file")
		}
	}

	return nil
}

func (d *TransformDaemon) transformListenerFnGetData(
	conn net.Conn,
	data transformData,
) error {
	switch data.InputStorage {
	case service.DataStorageInternal:
		type inputDataInternal struct {
			Storage    service.DataStorage `json:"storage"`
			Tags       []string            `json:"tags"`
			Properties []PropertyValue     `json:"properties"`
		}

		dataInt := data.Input.(transformScriptInputDataInternal)
		inputData := inputDataInternal{
			Storage:    data.InputStorage,
			Tags:       dataInt.Tags,
			Properties: dataInt.Properties,
		}

		err := writeMessage(conn, inputData)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", data,
			).Error("could not write response")
			return err
		}
	case service.DataStorageExternal:
		type inputDataExternal struct {
			Storage    service.DataStorage           `json:"storage"`
			Tags       []string                      `json:"tags"`
			Properties []PropertyValue               `json:"properties"`
			DataPaths  map[string]dataSourceFilePath `json:"data_paths"`
		}

		dataExt := data.Input.(transformScriptInputDataExternal)
		inputData := inputDataExternal{
			Storage:    data.InputStorage,
			Tags:       dataExt.Tags,
			Properties: dataExt.Properties,
			DataPaths:  dataExt.DataPaths,
		}

		err := writeMessage(conn, inputData)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", data,
			).Error("could not write response")
			return err
		}
	default:
		panic("invalid storage")
	}

	return nil
}

type ValuesFileFormat string

const (
	ValuesFileFormatCsv     ValuesFileFormat = "csv"
	ValuesFileFormatJson    ValuesFileFormat = "json"
	ValuesFileFormatFeather ValuesFileFormat = "feather"
)

func (d *TransformDaemon) transformListenerFnGetValues(
	conn net.Conn,
	payload uuid.UUID,
	format ValuesFileFormat,
) (*os.File, error) {
	switch format {
	case ValuesFileFormatCsv:
		file, err := d.createTransformDataValues(payload, ValuesFileFormatCsv)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", payload,
				"format", format,
			).Error("could not create data file")
			return nil, err
		}

		return file, nil
	case ValuesFileFormatJson:
		file, err := d.createTransformDataValues(payload, ValuesFileFormatJson)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", payload,
				"format", format,
			).Error("could not create data file")
			return nil, err
		}

		return file, nil
	case ValuesFileFormatFeather:
		file, err := d.createTransformDataValues(payload, ValuesFileFormatFeather)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", payload,
				"format", format,
			).Error("could not create data file")
			return nil, err
		}

		return file, nil
	default:
		panic(fmt.Sprintf("unexpected main.ValuesFileFormat: %#v", format))
	}
}

type PropertyValue struct {
	Key   string               `json:"key"`
	Type  service.PropertyType `json:"type"`
	Value any                  `json:"value"`
}

func (d *TransformDaemon) setJobError(job uuid.UUID, finish_time string, err error) error {
	err_query := fmt.Sprintf(
		`UPDATE _data_type_transform_queue_ 
			SET status='%s', finished=$2, error=$3
			WHERE _id=$1`,
		transformJobStatusFailed,
	)
	_, err = d.db.Conn.Exec(d.ctx, err_query, job, finish_time, err.Error())
	if err != nil {
		d.logger.With(
			"error", err,
			"job", job,
		).Error("could not update transform job status")
		return err
	}

	return nil
}

type transformData struct {
	Input         transformScriptInputData
	Output        any
	InputStorage  service.DataStorage
	OutputStorage service.DataStorage
	Files         []*os.File
}

type transformScriptInputData interface {
	TranformScriptInputData()
}

type transformScriptInputDataInternal struct {
	Tags       []string        `json:"tags"`
	Properties []PropertyValue `json:"properties"`
}

func (d transformScriptInputDataInternal) TranformScriptInputData() {}

type transformScriptInputDataExternal struct {
	Tags       []string                      `json:"tags"`
	Properties []PropertyValue               `json:"properties"`
	DataPaths  map[string]dataSourceFilePath `json:"sources"`
}

func (d transformScriptInputDataExternal) TranformScriptInputData() {}

type outputDataSchemaField struct {
	Label    string            `json:"label"`
	DType    service.ValueType `json:"dtype"`
	Required bool              `db:"_required"`
	Nullable bool              `db:"_nullable"`
}

type transformScriptOutputDataInternal struct {
	Cardinality service.DataSchemaCardinality `json:"cardinality"`
	Fields      []outputDataSchemaField       `json:"fields"`
}

type outputDataSource struct {
	Label       string                        `json:"label"`
	Required    bool                          `json:"required"`
	Cardinality service.DataSourceCardinality `json:"cardinality"`
	ExtFilter   []string                      `json:"ext_filter"`
}

type transformScriptOutputDataExternal struct {
	Sources []outputDataSource `json:"sources"`
}

type dataSourceFilePath struct {
	Single   string
	Multiple []string
}

// TODO: Include project specific properties?
func (d *TransformDaemon) createTransformData(
	payload uuid.UUID,
	transform transformInfo,
) (transformData, error) {
	properties_query :=
		`SELECT _key, _type, value FROM data_property_
		WHERE _data=$1`
	rows, _ := d.db.Conn.Query(d.ctx, properties_query, payload)
	data_properties, err := pgx.CollectRows(rows, pgx.RowToStructByName[service.Property])
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not get data properties")
		return transformData{}, err
	}

	properties := make([]PropertyValue, len(data_properties))
	for idx, prop := range data_properties {
		properties[idx] = PropertyValue{
			Key:   prop.Key,
			Type:  prop.Type,
			Value: prop.Value,
		}
	}

	transform_data := transformData{
		InputStorage:  transform.Source.DataStorage(),
		OutputStorage: transform.Destination.DataStorage(),
	}

	switch transform.Source.DataStorage() {
	case service.DataStorageExternal:
		sources, err := d.createTransformDataSources(payload)
		if err != nil {
			d.logger.With(
				"err", err,
				"sample_data", payload,
			).Error("could not create data file")
			return transformData{}, err
		}
		source_paths := make(map[string]dataSourceFilePath, len(sources))
		for key, source := range sources {
			if source.Single != nil {
				source_paths[key] = dataSourceFilePath{Single: source.Single.Name()}
				transform_data.Files = append(transform_data.Files, source.Single)
			} else if source.Multiple != nil {
				paths := make([]string, len(source.Multiple))
				for idx, file := range source.Multiple {
					paths[idx] = file.Name()
				}
				source_paths[key] = dataSourceFilePath{Multiple: paths}
				transform_data.Files = slices.Concat(transform_data.Files, source.Multiple)
			}
		}

		transform_data.Input = transformScriptInputDataExternal{
			DataPaths:  source_paths,
			Properties: properties,
		}

	case service.DataStorageInternal:
		transform_data.Input = transformScriptInputDataInternal{
			Properties: properties,
		}
	default:
		panic("invalid storage")
	}

	switch transform.Destination.DataStorage() {
	case service.DataStorageExternal:
		output := transform.Destination.(*dataTypeInfoExternal)
		sources := make([]outputDataSource, len(output.Sources))
		for idx, src := range output.Sources {
			sources[idx] = outputDataSource{
				Label:       src.Label,
				Cardinality: src.Cardinality,
				Required:    src.Required,
				ExtFilter:   src.ExtFilter,
			}
		}
		transform_data.Output = transformScriptOutputDataExternal{
			Sources: sources,
		}

	case service.DataStorageInternal:
		output := transform.Destination.(*dataTypeInfoInternal)
		fields := make([]outputDataSchemaField, len(output.Schema))
		for idx, field := range output.Schema {
			fields[idx] = outputDataSchemaField{
				Label:    field.Label,
				DType:    field.DType,
				Required: field.Required,
				Nullable: field.Nullable,
			}
		}
		transform_data.Output = transformScriptOutputDataInternal{
			Cardinality: output.Cardinality,
			Fields:      fields,
		}
	}

	return transform_data, nil
}

func (d *TransformDaemon) createTransformDataSources(
	data uuid.UUID,
) (map[string]dataSourceFileValue, error) {
	values_arr, err := d.data_service.DataValuesByIds([]uuid.UUID{data})
	if err != nil {
		d.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data values")
		return nil, err
	}
	values := values_arr[0]

	if values.Storage != service.DataStorageExternal {
		panic("should not be called on data that is not externally stored")

	}

	sources := values.Values.([]service.DataSource)
	source_files, err := transformDataSourcesCreateTemp(sources)
	if err != nil {
		d.logger.With(
			"error", err,
			"sources", sources,
		).Error("could not copy data sources")
		return nil, err
	}
	return source_files, nil
}

type dataSourceFileValue struct {
	Single   *os.File
	Multiple []*os.File
}

// transformDataSourcesCreateTemp creates temporary files for each source.
// Values of the map are an *os.File if the source's cardinality is `single`
// and a []*os.File if `multiple`.
func transformDataSourcesCreateTemp(sources []service.DataSource) (map[string]dataSourceFileValue, error) {
	source_files := make(map[string]dataSourceFileValue, len(sources))
	for _, src := range sources {
		switch src.Cardinality {
		case service.DataSourceCardinalityMultiple:
			sources := src.Source.([]string)
			fs := make([]*os.File, len(sources))
			for idx, path := range sources {
				f, err := copyToTmpfile(path)
				if err != nil {
					return nil, err
				}

				fs[idx] = f
			}
			source_files[src.Label] = dataSourceFileValue{Multiple: fs}
		case service.DataSourceCardinalitySingle:
			f, err := copyToTmpfile(src.Source.(string))
			if err != nil {
				return nil, err
			}

			source_files[src.Label] = dataSourceFileValue{Single: f}
		default:
			panic("invalid cardinality")
		}
	}

	return source_files, nil
}

func (d *TransformDaemon) createTransformDataValues(
	data uuid.UUID,
	format ValuesFileFormat,
) (*os.File, error) {
	values_arr, err := d.data_service.DataValuesByIds([]uuid.UUID{data})
	if err != nil {
		d.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data values")
		return nil, err
	}
	values := values_arr[0]

	if values.Storage != service.DataStorageInternal {
		panic("should not be called on data that is not internally stored")

	}

	fields := values.Values.([]service.SchemaFieldValues)
	switch format {
	case ValuesFileFormatCsv:
		file, err := d.createTransformDataFileCsv(fields)
		if err != nil {
			d.logger.With(
				"error", err,
				"fields", fields,
			).Error("could not create data values file")
			return nil, err
		}
		return file, nil
	case ValuesFileFormatJson:
		file, err := createTransformDataFileJson(fields)
		if err != nil {
			d.logger.With(
				"error", err,
				"fields", fields,
			).Error("could not create data values file")
			return nil, err
		}
		return file, nil
	case ValuesFileFormatFeather:
		file, err := createTransformDataFileFeather(fields)
		if err != nil {
			d.logger.With(
				"error", err,
				"fields", fields,
			).Error("could not create data values file")
			return nil, err
		}
		return file, nil
	default:
		panic(fmt.Sprintf("unexpected main.ValuesFileFormat: %#v", format))
	}
}

func (d *TransformDaemon) createTransformDataFileCsv(fields []service.SchemaFieldValues) (*os.File, error) {
	out, err := d.data_service.StoredDataToCsv(fields)
	if err != nil {
		return nil, err
	}

	tmpfile, err := os.CreateTemp("", "*.csv")
	if err != nil {
		return nil, err
	}

	_, err = tmpfile.WriteString(out)
	if err != nil {
		tmpfile.Close()
		return nil, err
	}

	return tmpfile, nil
}

func createTransformDataFileJson(fields []service.SchemaFieldValues) (*os.File, error) {
	tmpfile, err := os.CreateTemp("", "*.csv")
	if err != nil {
		return nil, err
	}

	writer := json.NewEncoder(tmpfile)
	switch fields[0].Cardinality {
	case service.DataSchemaCardinalityMultiple:
		obj := make(map[string][]any, len(fields))
		for _, field := range fields {
			obj[field.Label] = field.Values.([]any)
		}
		err = writer.Encode(obj)
		if err != nil {
			tmpfile.Close()
			return nil, err
		}
	case service.DataSchemaCardinalitySingle:
		obj := make(map[string]any, len(fields))
		for _, field := range fields {
			obj[field.Label] = field.Values
		}
		err = writer.Encode(obj)
		if err != nil {
			tmpfile.Close()
			return nil, err
		}
	default:
		panic("unexpected service.DataSchemaCardinality")
	}

	return tmpfile, nil
}

func createTransformDataFileFeather(fields []service.SchemaFieldValues) (*os.File, error) {
	pool := memory.NewGoAllocator()
	arrow_fields := make([]arrow.Field, len(fields))
	cols := make([]arrow.Array, len(fields))
	for idx, field := range fields {
		arrow_field, arrow_col := fieldToArrow(field, pool)
		defer arrow_col.Release()

		arrow_fields[idx] = arrow_field
		cols[idx] = arrow_col
	}
	height := int64(cols[0].Len())
	schema := arrow.NewSchema(arrow_fields, nil)
	data := array.NewRecordBatch(schema, cols, height)

	tmpfile, err := os.CreateTemp("", "*.arrow")
	if err != nil {
		return nil, err
	}

	writer, err := ipc.NewFileWriter(tmpfile, ipc.WithSchema(schema))
	if err != nil {
		tmpfile.Close()
		return nil, err
	}
	defer writer.Close()

	err = writer.Write(data)
	if err != nil {
		tmpfile.Close()
		return nil, err
	}

	return tmpfile, nil
}

// fieldToArrow creates arrow resources for the give data schema field.
// The `arrow.Array` must be `Release`d after use.
func fieldToArrow(field service.SchemaFieldValues, pool memory.Allocator) (arrow.Field, arrow.Array) {
	arrow_type := valueTypeToArrow(field.DType)
	arrow_field := arrow.Field{
		Name: field.Label,
		Type: arrow_type,
	}

	switch field.DType {
	case service.ValueTypeBoolean:
		builder := array.NewBooleanBuilder(pool)
		defer builder.Release()

		switch field.Cardinality {
		case service.DataSchemaCardinalityMultiple:
			vals_arr := field.Values.([]any)
			values := make([]bool, len(vals_arr))
			for idx, val := range vals_arr {
				values[idx] = val.(bool)
			}

			builder.AppendValues(values, nil)
		case service.DataSchemaCardinalitySingle:
			value := field.Values.(bool)
			builder.Append(value)
		}
		values := builder.NewArray()

		return arrow_field, values

	case service.ValueTypeFloat:
		builder := array.NewFloat64Builder(pool)
		defer builder.Release()

		switch field.Cardinality {
		case service.DataSchemaCardinalityMultiple:
			vals_arr := field.Values.([]any)
			values := make([]float64, len(vals_arr))
			for idx, val := range vals_arr {
				values[idx] = val.(float64)
			}

			builder.AppendValues(values, nil)
		case service.DataSchemaCardinalitySingle:
			value := field.Values.(float64)
			builder.Append(value)
		}
		values := builder.NewArray()

		return arrow_field, values

	case service.ValueTypeInt:
		builder := array.NewInt32Builder(pool)
		defer builder.Release()

		switch field.Cardinality {
		case service.DataSchemaCardinalityMultiple:
			vals_arr := field.Values.([]any)
			values := make([]int32, len(vals_arr))
			for idx, val := range vals_arr {
				values[idx] = val.(int32)
			}

			builder.AppendValues(values, nil)
		case service.DataSchemaCardinalitySingle:
			value := field.Values.(int32)
			builder.Append(value)
		}
		values := builder.NewArray()

		return arrow_field, values

	case service.ValueTypeString:
		builder := array.NewStringBuilder(pool)
		defer builder.Release()

		switch field.Cardinality {
		case service.DataSchemaCardinalityMultiple:
			vals_arr := field.Values.([]any)
			values := make([]string, len(vals_arr))
			for idx, val := range vals_arr {
				values[idx] = val.(string)
			}

			builder.AppendValues(values, nil)
		case service.DataSchemaCardinalitySingle:
			value := field.Values.(string)
			builder.Append(value)
		}
		values := builder.NewArray()

		return arrow_field, values

	case service.ValueTypeUint:
		builder := array.NewUint32Builder(pool)
		defer builder.Release()

		switch field.Cardinality {
		case service.DataSchemaCardinalityMultiple:
			vals_arr := field.Values.([]any)
			values := make([]uint32, len(vals_arr))
			for idx, val := range vals_arr {
				values[idx] = uint32(val.(float64))
			}

			builder.AppendValues(values, nil)
		case service.DataSchemaCardinalitySingle:
			value := field.Values.(uint32)
			builder.Append(value)
		}
		values := builder.NewArray()

		return arrow_field, values

	case service.ValueTypeTimestamp:
		panic("TODO: timestamp data")
	default:
		panic("invalid value type")
	}
}

func valueTypeToArrow(kind service.ValueType) arrow.DataType {
	switch kind {
	case service.ValueTypeBoolean:
		return &arrow.BooleanType{}
	case service.ValueTypeFloat:
		return arrow.PrimitiveTypes.Float64
	case service.ValueTypeInt:
		return arrow.PrimitiveTypes.Int32
	case service.ValueTypeString:
		return &arrow.StringType{}
	case service.ValueTypeTimestamp:
		return &arrow.TimestampType{}
	case service.ValueTypeUint:
		return arrow.PrimitiveTypes.Uint32
	default:
		panic(fmt.Sprintf("unexpected service.ValueType: %#v", kind))
	}
}

func (d *TransformDaemon) transformListenerFnGetOutputDataInfo(
	conn net.Conn,
	data transformData,
) error {
	switch data.OutputStorage {
	case service.DataStorageInternal:
		type schemaField struct {
			Label    string            `json:"label"`
			DType    service.ValueType `json:"dtype"`
			Required bool              `json:"required"`
			Nullable bool              `json:"nullable"`
		}
		type outputDataInternal struct {
			Storage     service.DataStorage           `json:"storage"`
			Cardinality service.DataSchemaCardinality `json:"cardinality"`
			Fields      []schemaField                 `json:"schema"`
		}

		dataInt := data.Output.(transformScriptOutputDataInternal)
		fields := make([]schemaField, len(dataInt.Fields))
		for idx, field := range dataInt.Fields {
			fields[idx] = schemaField{
				Label:    field.Label,
				DType:    field.DType,
				Required: field.Required,
				Nullable: field.Nullable,
			}
		}
		outputData := outputDataInternal{
			Storage:     service.DataStorageInternal,
			Cardinality: dataInt.Cardinality,
			Fields:      fields,
		}

		err := writeMessage(conn, outputData)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", data,
			).Error("could not write response")
			return err
		}
	case service.DataStorageExternal:
		type outputDataExternal struct {
			Storage service.DataStorage `json:"storage"`
			Sources []outputDataSource  `json:"sources"`
		}

		dataExt := data.Output.(transformScriptOutputDataExternal)
		inputData := outputDataExternal{
			Storage: service.DataStorageExternal,
			Sources: dataExt.Sources,
		}

		err := writeMessage(conn, inputData)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", data,
			).Error("could not write response")
			return err
		}
	default:
		panic("invalid storage")
	}

	return nil
}

func (d *TransformDaemon) transformListenerFnSaveData(
	conn net.Conn,
	token uuid.UUID,
	payload uuid.UUID,
	transform_info transformInfo,
) error {
	err := writeMessage(conn, map[string]string{"status": "ok"})
	if err != nil {
		d.logger.With(
			"error", err,
			"job", token,
		).Error("could not write response")
		return err
	}

	data, err := readData(conn)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", data,
		).Error("could not read data")
		return err
	}

	if data.Token != token {
		d.logger.With(
			"expected", token,
			"recieved", data.Token,
		).Warn("invalid token")
		writeMessage(conn, map[string]string{
			"status": "err",
			"err":    "invalid token",
		})
		return nil
	}

	err = d.dataCreate(payload, transform_info, data)
	if err != nil {
		err = writeMessage(conn, map[string]string{
			"status": "err",
			"error":  err.Error(),
		})
		if err != nil {
			d.logger.With(
				"error", err,
				"job", token,
			).Error("could not write response")
			return err
		}
	}

	err = writeMessage(conn, map[string]string{
		"status": "ok",
	})
	if err != nil {
		d.logger.With(
			"error", err,
			"job", token,
		).Error("could not write response")
		return err
	}

	return nil
}

func (d *TransformDaemon) dataCreate(
	payload uuid.UUID,
	transform_info transformInfo,
	data outputData,
) error {
	tx, err := d.db.Conn.Begin(d.ctx)
	if err != nil {
		d.logger.With("error", err).Error("could not begin transaction")
		return err
	}
	defer tx.Rollback(d.ctx)

	var visibility service.Visibility
	visibility_query := "SELECT visibility FROM data_ WHERE _id=$1"
	err = tx.QueryRow(d.ctx, visibility_query, payload).Scan(&visibility)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not get data visibility")
		return err
	}

	owners_query := fmt.Sprintf(
		"SELECT _user FROM data_user_permission_ WHERE _data=$1 AND _permission='%s'",
		service.DataPermissionKeyOwner,
	)
	rows, err := tx.Query(d.ctx, owners_query, payload)
	data_owners, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not get data owners")
		return err
	}

	var project_id uuid.UUID
	var membership_creator_id uuid.UUID
	project_query :=
		`SELECT _project, _creator FROM project_data_membership_ 
		WHERE _data=$1`
	err = tx.QueryRow(d.ctx, project_query, payload).Scan(&project_id, &membership_creator_id)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not get project data membership")
		return err
	}

	var data_type uuid.UUID
	switch transform_info.Destination.DataStorage() {
	case service.DataStorageExternal:
		output := transform_info.Destination.(*dataTypeInfoExternal)
		data_type = output.Id
	case service.DataStorageInternal:
		output := transform_info.Destination.(*dataTypeInfoInternal)
		data_type = output.Id
	default:
		panic("unexpected service.DataStorage")
	}

	var data_id uuid.UUID
	data_query := fmt.Sprintf(
		`INSERT INTO data_ (_type, _creator_type, timestamp, visibility)
		VALUES ($1, '%s', $2, $3) RETURNING _id`,
		service.DataCreatorTypeTransform,
	)
	err = tx.QueryRow(
		d.ctx,
		data_query,
		data_type,
		time.Now(),
		visibility,
	).Scan(&data_id)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not create data")
		return err
	}

	permission_args := make([]any, len(data_owners)+1)
	permission_args[0] = data_id
	var permission_query strings.Builder
	permission_query.WriteString(
		"INSERT INTO data_user_permission_ (_data, _user, _permission) VALUES ",
	)
	for idx, owner := range data_owners {
		if idx > 0 {
			permission_query.WriteString(", ")
		}

		idx_arg := idx + 1
		fmt.Fprintf(
			&permission_query,
			"($1, $%d, '%s')",
			idx_arg+1,
			service.DataPermissionKeyOwner,
		)

		permission_args[idx_arg] = owner
	}
	_, err = tx.Exec(d.ctx, permission_query.String(), permission_args...)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not create data permissions")
		return err
	}

	creator_query :=
		`INSERT INTO data_creator_transform_ (_data, _creator) 
		VALUES ($1, $2)`
	_, err = tx.Exec(d.ctx, creator_query, data_id, transform_info.Id)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not create data creator")
		return err
	}

	membership_query :=
		`INSERT INTO project_data_membership_ (_project, _data, _creator)
		VALUES ($1, $2 ,$3)`
	_, err = tx.Exec(d.ctx, membership_query, project_id, data_id, membership_creator_id)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not create project data membership")
		return err
	}

	if len(data.Properties) > 0 {
		const PROPERTY_ARGS_OFFSET = 1
		const PROPERTY_ARGS_PER_RECORD = 3
		property_args := make([]any, len(data.Properties)*PROPERTY_ARGS_PER_RECORD+PROPERTY_ARGS_OFFSET)
		property_args[0] = data_id

		var property_query strings.Builder
		property_query.WriteString(
			"INSERT INTO data_property_ (_data, _key, _type, value) VALUES ",
		)
		for idx, property := range data.Properties {
			if idx > 0 {
				property_query.WriteString(", ")
			}

			idx_key := idx*PROPERTY_ARGS_PER_RECORD + PROPERTY_ARGS_OFFSET
			idx_type := idx_key + 1
			idx_value := idx_type + 1
			fmt.Fprintf(
				&property_query,
				"($1, $%d, $%d, $%d)",
				idx_key+1,
				idx_type+1,
				idx_value+1,
			)

			property_args[idx_key] = property.Key
			property_args[idx_type] = property.Type
			property_args[idx_value] = property.Value
		}
		_, err = tx.Exec(d.ctx, property_query.String(), property_args...)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", payload,
				"query", property_query.String(),
			).Error("could not create data properties")
			return err
		}
	}

	if len(data.Tags) > 0 {
		tags_args := make([]any, len(data.Tags)+2)
		tags_args[0] = project_id
		tags_args[1] = data_id
		var tags_query strings.Builder
		tags_query.WriteString("INSERT INTO project_data_tag_ (_project, _data, _tag) VALUES ")
		for idx, tag := range data.Tags {
			if idx > 0 {
				tags_query.WriteString(", ")
			}

			idx_tag := idx + 2
			fmt.Fprintf(&tags_query, "($1, $2, $%d)", idx_tag+1)
			tags_args[idx_tag] = tag
		}
		_, err = tx.Exec(d.ctx, tags_query.String(), tags_args...)
		if err != nil {
			d.logger.With(
				"error", err,
				"data", payload,
			).Error("could not create project data tags")
			return err
		}
	}

	err = d.dataCreateValues(tx, transform_info.Destination, data_id, data.Values)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not store data values")
		return err
	}

	err = tx.Commit(d.ctx)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not commit data transform")
		return err
	}

	return nil
}

func (d *TransformDaemon) dataCreateValues(
	tx pgx.Tx,
	dst_info service.DataType,
	data_id uuid.UUID,
	values map[string]any,
) error {
	switch dst_info.DataStorage() {
	case service.DataStorageExternal:
		dst := dst_info.(*dataTypeInfoExternal)
		return d.dataCreateValuesExternal(tx, dst, data_id, values)
	case service.DataStorageInternal:
		dst := dst_info.(*dataTypeInfoInternal)
		return d.dataCreateValuesInternal(tx, dst, data_id, values)
	default:
		panic("unexpected service.DataStorage")
	}
}

func (d *TransformDaemon) dataCreateValuesExternal(
	tx pgx.Tx,
	dst *dataTypeInfoExternal,
	data_id uuid.UUID,
	values map[string]any,
) error {
	panic("TODO: dataCreateValuesExternal")
}

func (d *TransformDaemon) dataCreateValuesInternal(
	tx pgx.Tx,
	dst *dataTypeInfoInternal,
	data_id uuid.UUID,
	values map[string]any,
) error {
	table := service.DataStorageTableNameFromSchemaId(dst.SchemaId)
	fields := make([]string, 1, len(values)+1)
	args := make([]any, 1, len(values)+1)
	fields[0] = "_data"
	args[0] = data_id
	for key, vals := range values {
		fields = append(fields, key)
		args = append(args, vals)
	}
	args_idx := make([]string, len(args))
	for idx := range args {
		args_idx[idx] = fmt.Sprintf("$%d", idx+1)
	}
	var query strings.Builder
	fmt.Fprintf(
		&query,
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(fields, ", "),
		strings.Join(args_idx, ", "),
	)

	_, err := tx.Exec(d.ctx, query.String(), args...)
	if err != nil {
		d.logger.With(
			"error", err,
			"table", table,
			"fields", fields,
			"query", query.String(),
		).Error("could not store data values")
		return err
	}

	return nil
}

func script_path_from_tranform_id(app_dir string, id uuid.UUID) string {
	// TODO: Update to match creation.
	script_name := fmt.Sprintf("%s.%s", id.String(), "py")
	return filepath.Join(app_dir, string(service.AppDataDirTransform), script_name)
}

// copyToTmpfile copies a file to a temporary file and returns the path
// to the temporary file.
func copyToTmpfile(path string) (*os.File, error) {
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	ext := filepath.Ext(path)
	tmpfile, err := os.CreateTemp("", fmt.Sprintf("*%s", ext))
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(src, tmpfile)
	if err != nil {
		return nil, err
	}

	return tmpfile, nil
}
