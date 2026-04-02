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
			continue
		}

		var transform_ids []uuid.UUID
		for _, job := range jobs {
			if !slices.Contains(transform_ids, job.Transform) {
				transform_ids = append(transform_ids, job.Transform)
			}
		}
		transforms_info, err := d.getTransformsById(transform_ids)
		if err != nil {
			continue
		}

		var data_schema_ids []uuid.UUID
		for _, transform := range transforms_info {
			if !slices.Contains(data_schema_ids, transform.Input) {
				data_schema_ids = append(data_schema_ids, transform.Input)
			}
			// TODO: account for multiple outputs
			if !slices.Contains(data_schema_ids, transform.Output[0]) {
				data_schema_ids = append(data_schema_ids, transform.Output[0])
			}
		}
		data_schemas, err := d.getDataSchemasById(data_schema_ids)

		for _, job := range jobs {
			transform_idx := slices.IndexFunc(transforms_info, func(info TransformInfo) bool {
				return info.Id == job.Transform
			})
			if transform_idx < 0 {
				d.logger.With("transform", job.Transform).Error("invalid transform")
				panic("invalid transform")
			}
			transform := transforms_info[transform_idx]

			input_idx := slices.IndexFunc(data_schemas, func(schema DataSchemaInfo) bool {
				return schema.Id == transform.Input
			})
			if input_idx < 0 {
				d.logger.With("transform", transform).Error("invalid input schema")
				panic("invalid input data schema")
			}
			input_schema := data_schemas[input_idx]

			output_idx := slices.IndexFunc(data_schemas, func(schema DataSchemaInfo) bool {
				return schema.Id == transform.Output[0] // TODO: account for multiple outputs
			})
			if output_idx < 0 {
				d.logger.With("transform", transform).Error("invalid output schema")
				panic("invalid output data schema")
			}
			output_schema := data_schemas[output_idx]

			go d.runTransform(
				job.Id,
				job.Transform,
				job.Payload,
				input_schema,
				output_schema,
				transform.Script,
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

type transformJobInfo struct {
	Id        uuid.UUID
	Transform uuid.UUID
	Payload   uuid.UUID
}

func (d *TransformDaemon) pollPending() ([]transformJobInfo, error) {
	query := fmt.Sprintf(
		"SELECT _id, _transform, _payload FROM _data_type_transform_queue_ WHERE status='%s'",
		transformJobStatusPending,
	)
	rows, _ := d.db.Conn.Query(d.ctx, query)
	jobs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (transformJobInfo, error) {
		var job transformJobInfo
		err := row.Scan(
			&job.Id,
			&job.Transform,
			&job.Payload,
		)
		return job, err
	})
	if err != nil {
		d.logger.With("error", err).Error("could not poll transform queue")
		return nil, err
	}

	return jobs, nil
}

type TransformInfo struct {
	Id     uuid.UUID
	Input  uuid.UUID
	Output []uuid.UUID
	Script string
}

func (d *TransformDaemon) getTransformsById(transforms []uuid.UUID) ([]TransformInfo, error) {
	// TODO
	return nil, nil

	// query := "SELECT _id, _input, _script FROM data_type_transform_ WHERE _id=ANY($1)"
	// rows, _ := d.db.Conn.Query(d.ctx, query, transforms)
	// info, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (TransformInfo, error) {
	// 	var info TransformInfo
	// 	err := row.Scan(&info.Id, &info.Input, &info.Output, &info.Script)
	// 	return info, err
	// })
	// if err != nil {
	// 	d.logger.With(
	// 		"error", err,
	// 		"transforms", transforms,
	// 	).Error("could not get transforms")
	// 	return nil, err
	// }

	// return info, nil
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
	transform uuid.UUID,
	payload uuid.UUID,
	source DataSchemaInfo,
	destination DataSchemaInfo,
	script_path string,
) {
	start_query := fmt.Sprintf(
		"UPDATE _transform_queue_ SET status='%s', started=$1 WHERE _id=$2",
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

	data_path, err := d.createTransformDataFile(payload)
	cmd := exec.Command(
		// "python",
		"C:\\Users\\carls\\.venv\\Scripts\\python.exe", // TODO: Change to transform data
		script_path,
		job.String(),
		payload.String(),
		data_path,
		destination.Id.String(),
		transform.String(),
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
			`UPDATE _transform_queue_ 
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
			`UPDATE _transform_queue_ 
			SET status='%s', finished=$2
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

type sampleInfo struct {
	Properties map[string]service.Property
}

type transformScriptData struct {
	DataPath   string         `json:"data_path"`
	Properties map[string]any `json:"properties"`
}

func (d *TransformDaemon) createTransformDataFile(
	payload uuid.UUID,
) (string, error) {
	sample_properties_query :=
		`SELECT p._key, p._type, p.value 
		FROM sample_property_ as p JOIN sample_ as s ON p._sample=s._id
		WHERE s._id=$1`
	rows, _ := d.db.Conn.Query(d.ctx, sample_properties_query, payload)
	sample_properties, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (service.Property, error) {
		var property service.Property
		err := row.Scan(&property.Key, &property.Type, &property.Value)
		return property, err
	})
	if err != nil {
		d.logger.With(
			"error", err,
			"sample data", payload,
		).Error("could not get sample properties")
		return "", err
	}

	sample_data_properties_query :=
		"SELECT _key, _type, value FROM sample_data_property_ WHERE _sample_data=$1"
	rows, _ = d.db.Conn.Query(d.ctx, sample_data_properties_query, payload)
	sample_data_properties, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (service.Property, error) {
		var property service.Property
		err := row.Scan(&property.Key, &property.Type, &property.Value)
		return property, err
	})
	if err != nil {
		d.logger.With(
			"error", err,
			"sample data", payload,
		).Error("could not get sample data properties")
		return "", err
	}

	properties := make(map[string]any)
	for _, prop := range sample_properties {
		properties[prop.Key] = prop.Value
	}
	for _, prop := range sample_data_properties {
		properties[prop.Key] = prop.Value
	}

	data_file, err := d.createTransformDataFileData(payload)
	if err != nil {
		d.logger.With(
			"err", err,
			"sample_data", payload,
		).Error("could not create data file")
		return "", err
	}
	defer data_file.Close()

	transform_data := transformScriptData{
		DataPath:   data_file.Name(),
		Properties: properties,
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
	encoder.Encode(transform_data)

	return transform_file.Name(), nil
}

func (d *TransformDaemon) createTransformDataFileData(sample_data uuid.UUID) (*os.File, error) {
	stored_data_arr, err := d.data_service.SampleDataStoredById([]uuid.UUID{sample_data})
	if err != nil {
		d.logger.With(
			"error", err,
			"sample data", sample_data,
		).Error("could not get stored sample data")
		return nil, err
	}
	stored_data := stored_data_arr[0]

	var tmpfile *os.File
	switch stored_data.Storage {
	case service.DataStorageExternal:
		info := stored_data.Data.(service.SampleDataPayloadExternal)
		src, err := os.Open(info.Path)
		defer src.Close()
		if err != nil {
			d.logger.With(
				"error", err,
				"file path", info.Path,
			).Error("could not read file data")

			return nil, err
		}

		tmpfile, err = os.CreateTemp("", "")
		if err != nil {
			d.logger.With(
				"error", err,
				"sample data", sample_data,
			).Error("could not create temporary data file")
			return nil, err
		}

		_, err = io.Copy(src, tmpfile)
		if err != nil {
			d.logger.With(
				"error", err,
				"sample data", sample_data,
			).Error("could not copy data file")
			return nil, err
		}

	case service.DataStorageInternal:
		data, err := d.data_service.StoredDataToCsv(stored_data.Data.([]service.ColumnData))
		if err != nil {
			d.logger.With("stored data", stored_data).Error("could not get stored sample data")
			return nil, err
		}

		tmpfile, err = os.CreateTemp("", "*.csv")
		if err != nil {
			d.logger.With(
				"error", err,
				"sample data", sample_data,
			).Error("could not create temporary data file")
			return nil, err
		}
		_, err = tmpfile.Write(data)
		if err != nil {
			d.logger.With(
				"error", err,
				"sample data", sample_data,
			).Error("could not write data to file")
			return nil, err
		}
	default:
		panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored_data.Storage))
	}

	return tmpfile, nil
}

func script_path_from_tranform_id(app_dir string, id uuid.UUID) string {
	// TODO: Update to match creation.
	script_name := fmt.Sprintf("%s.%s", id.String(), "py")
	return filepath.Join(app_dir, string(service.AppDataDirTransform), script_name)
}
