package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

	data_path, err := d.createTransformDataFile(payload, transform)
	if err != nil {
		d.logger.With(
			"error", err,
			"data", payload,
		).Error("could not create transform data file")

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
	cmd := exec.Command(
		transform.Cmd.Cmd,
		transform.Cmd.Path,
		job.String(),
		data_path,
	)
	var cmd_out strings.Builder
	var cmd_err strings.Builder
	cmd.Stdout = &cmd_out
	cmd.Stderr = &cmd_err

	err = cmd.Run()
	finish_time := time.Now().Format(time.RFC3339)
	if err != nil {
		d.logger.With(
			"error", err,
			"job", job,
			"cmd", cmd.String(),
			"err", cmd_err.String(),
		).Error("error running command")

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
		}
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

type transformInfoFile struct {
	Input         any                 `json:"input"` // `transformScriptInputDataInternal` or `transformScriptInputDataExternal`
	Output        any                 `json:"output"`
	InputStorage  service.DataStorage `json:"input_storage"`
	OutputStorage service.DataStorage `json:"output_storage"`
}

type transformScriptInputDataInternal struct {
	DataPath   string         `json:"data_path"`
	Tags       []string       `json:"tags"`
	Properties map[string]any `json:"properties"`
}

type transformScriptInputDataExternal struct {
	DataPaths  map[string]any `json:"sources"`
	Tags       []string       `json:"tags"`
	Properties map[string]any `json:"properties"`
}

type outputDataSchemaField struct {
	Label        string                              `json:"label"`
	DType        service.ValueType                   `json:"dtype"`
	Availability service.DataSchemaFieldAvailability `json:"availability"`
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

// TODO: Include project specific properties?
func (d *TransformDaemon) createTransformDataFile(
	payload uuid.UUID,
	transform transformInfo,
) (string, error) {
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
		return "", err
	}

	properties := make(map[string]any)
	for _, prop := range data_properties {
		properties[prop.Key] = prop.Value
	}

	data_paths, err := d.createTransformDataFileData(payload)
	if err != nil {
		d.logger.With(
			"err", err,
			"sample_data", payload,
		).Error("could not create data file")
		return "", err
	}

	transform_info := transformInfoFile{
		InputStorage:  transform.Source.DataStorage(),
		OutputStorage: transform.Destination.DataStorage(),
	}

	switch transform.Source.DataStorage() {
	case service.DataStorageExternal:
		transform_info.Input = transformScriptInputDataExternal{
			DataPaths:  data_paths.(map[string]any),
			Properties: properties,
		}

	case service.DataStorageInternal:
		transform_info.Input = transformScriptInputDataInternal{
			DataPath:   data_paths.(string),
			Properties: properties,
		}
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
		transform_info.Output = transformScriptOutputDataExternal{
			Sources: sources,
		}

	case service.DataStorageInternal:
		output := transform.Destination.(*dataTypeInfoInternal)
		fields := make([]outputDataSchemaField, len(output.Schema))
		for idx, field := range output.Schema {
			fields[idx] = outputDataSchemaField{
				Label:        field.Label,
				DType:        field.DType,
				Availability: field.Availability,
			}
		}
		transform_info.Output = transformScriptOutputDataInternal{
			Cardinality: output.Cardinality,
			Fields:      fields,
		}
	default:
		panic("unexpected service.DataStorage")
	}

	transform_file, err := os.CreateTemp("", "*.json")
	if err != nil {
		d.logger.With(
			"error", err,
		).Error("could not create temp file")
		return "", err
	}
	defer transform_file.Close()
	encoder := json.NewEncoder(transform_file)
	encoder.Encode(transform_info)

	return transform_file.Name(), nil
}

// createTransformDataFileData creates temporary files holding the data's values.
// If the data is internally stored it returns a single data path (`string`).
// If the data is externally stored it returns a map from the source labels to the data path(s)
// (`map[string]any` where values are `string` if the source has `single` cardinality
// or `[]string` if it has `multiple` cardinality).
func (d *TransformDaemon) createTransformDataFileData(
	data uuid.UUID,
) (any, error) {
	values_arr, err := d.data_service.DataValuesById([]uuid.UUID{data})
	if err != nil {
		d.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data values")
		return nil, err
	}
	values := values_arr[0]

	switch values.Storage {
	case service.DataStorageExternal:
		sources := values.Values.([]service.DataSource)
		data_paths, err := createTransformDataFileDataExternal(sources)
		if err != nil {
			d.logger.With(
				"error", err,
				"sources", sources,
			).Error("could not copy data sources")
			return nil, err
		}
		return data_paths, nil

	case service.DataStorageInternal:
		fields := values.Values.([]service.SchemaFieldValues)
		data_path, err := createTransformDataFileDataInternal(fields)
		if err != nil {
			d.logger.With(
				"error", err,
				"fields", fields,
			).Error("could not create data values file")
			return nil, err
		}
		return data_path, nil
	}

	panic("unreachable")
}

// createTransformDataFileDataExternal creates temporary files for each source.
// Values of the map are a string if the source's cardinality is `single`
// and a []string if `multiple`.
func createTransformDataFileDataExternal(sources []service.DataSource) (map[string]any, error) {
	data_paths := make(map[string]any, len(sources))
	for _, src := range sources {
		switch src.Cardinality {
		case service.DataSourceCardinalityMultiple:
			sources := src.Source.([]string)
			paths := make([]string, len(sources))
			for idx, path := range sources {
				tmp_path, err := copyToTmpfile(path)
				if err != nil {
					return nil, err
				}

				paths[idx] = tmp_path
			}
			data_paths[src.Label] = paths

		case service.DataSourceCardinalitySingle:
			tmp_path, err := copyToTmpfile(src.Source.(string))
			if err != nil {
				return nil, err
			}

			data_paths[src.Label] = tmp_path
		}
	}

	return data_paths, nil
}

// copyToTmpfile copies a file to a temporary file and returns the path
// to the temporary file.
func copyToTmpfile(path string) (string, error) {
	src, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer src.Close()

	ext := filepath.Ext(path)
	tmpfile, err := os.CreateTemp("", fmt.Sprintf("*%s", ext))
	if err != nil {
		return "", err
	}

	_, err = io.Copy(src, tmpfile)
	if err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

func createTransformDataFileDataInternal(fields []service.SchemaFieldValues) (string, error) {
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
		return "", err
	}
	defer tmpfile.Close()

	writer, err := ipc.NewFileWriter(tmpfile, ipc.WithSchema(schema))
	if err != nil {
		return "", err
	}
	defer writer.Close()

	err = writer.Write(data)
	if err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
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
	}

	panic("unreachable")
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

func script_path_from_tranform_id(app_dir string, id uuid.UUID) string {
	// TODO: Update to match creation.
	script_name := fmt.Sprintf("%s.%s", id.String(), "py")
	return filepath.Join(app_dir, string(service.AppDataDirTransform), script_name)
}
