package app

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type DataType string

const (
	DATA_TYPE_STRING    = DataType("string")
	DATA_TYPE_INT       = DataType("int")
	DATA_TYPE_UINT      = DataType("uint")
	DATA_TYPE_FLOAT     = DataType("float")
	DATA_TYPE_BOOLEAN   = DataType("boolean")
	DATA_TYPE_TIMESTAMP = DataType("timestamp")
)

type SaveDataHierarchy string

const (
	SAVE_DATA_HIERARCHY_DATA_SCHEMA = SaveDataHierarchy("data_schema")
	SAVE_DATA_HIERARCHY_SAMPLE      = SaveDataHierarchy("sample")
)

type DataService struct {
	ctx          context.Context
	logger       *slog.Logger
	db           *DbConnection
	app_state    *AppState
	fs_service   *FsService
	user_service *UserService
}

func NewDataService(
	logger *slog.Logger,
	db *DbConnection,
	app_state *AppState,
	fs_service *FsService,
	user_service *UserService,
) *DataService {
	return &DataService{
		logger:       logger,
		db:           db,
		app_state:    app_state,
		fs_service:   fs_service,
		user_service: user_service,
	}
}

func (s *DataService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

func (s *DataService) user_id() uuid.UUID {
	s.app_state._lock.RLock()
	defer s.app_state._lock.RUnlock()
	return s.app_state.user_id
}

type ColumnSchema struct {
	Label string   `json:"label"`
	DType DataType `json:"dtype"`
}

type DataStorage string

const (
	DATA_STORAGE_INTERNAL = DataStorage("internal")
	DATA_STORAGE_EXTERNAL = DataStorage("external")
)

type DataSchema struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Schema      []ColumnSchema
	Storage     DataStorage
	Label       string
	Description string
}

func (s *DataService) GetDataSchemasAll() ([]DataSchema, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return []DataSchema{}, &UserNotAuthenticatedError{}
	}

	data_schema_query := "SELECT (_id, _creator, _schema, _storage, label, description) FROM data_schema_ ORDER BY _id DESC"
	rows, err := s.db.conn.Query(s.ctx, data_schema_query)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schemas")
		return []DataSchema{}, err
	}

	schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchema, error) {
		s.logger.Debug("v", "desc", row.FieldDescriptions())
		var schema DataSchema
		err := row.Scan(&schema)
		return schema, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schemas")
		return []DataSchema{}, err
	}

	return schemas, nil
}

func (s *DataService) GetDataSchemasById(schema_ids []uuid.UUID) ([]DataSchema, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return []DataSchema{}, &UserNotAuthenticatedError{}
	}
	if len(schema_ids) == 0 {
		return []DataSchema{}, nil
	}

	data_schema_query := `
		SELECT (_id, _creator, _schema, _storage, label, description) 
		FROM data_schema_ 
		WHERE _id=ANY($1)
		ORDER BY _id DESC`
	rows, err := s.db.conn.Query(s.ctx, data_schema_query, schema_ids)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schemas")
		return []DataSchema{}, err
	}

	schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchema, error) {
		s.logger.Debug("v", "desc", row.FieldDescriptions())
		var schema DataSchema
		err := row.Scan(&schema)
		return schema, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schemas")
		return []DataSchema{}, err
	}

	return schemas, nil
}

type InvalidSampleDataColumnLabels struct {
	Labels []string
}

func (e *InvalidSampleDataColumnLabels) Error() string {
	return fmt.Sprintf(
		"INVALID_SAMPLE_DATA_COLUMN_LABELS [%s]",
		strings.Join(e.Labels, ","),
	)
}

func validate_sample_table_column_labels(schema []ColumnSchema) error {
	err := &InvalidSampleDataColumnLabels{}
	for _, col := range schema {
		if !(is_valid_table_column_label(col.Label)) {
			err.Labels = append(err.Labels, col.Label)
		}
	}

	if len(err.Labels) > 0 {
		return err
	} else {
		return nil

	}
}

func is_valid_table_column_label(label string) bool {
	const PATTERN = `^[\w_]+$`
	match, err := regexp.MatchString(PATTERN, label)
	if err != nil {
		panic(err)
	}

	return match
}

type DataSchemaCreate struct {
	Schema      []ColumnSchema
	Storage     DataStorage
	Label       string
	Description string
}

func (s *DataService) DataSchemaCreate(data_schema DataSchemaCreate) (Ok, error) {
	user_id := s.user_id()
	user_role, err := s.user_service.UserRole()
	if err != nil || (user_role != USER_ROLE_OWNER && user_role != USER_ROLE_ADMIN) {
		s.logger.With("user", user_id).Debug(
			"insufficient permissions to create data schema for user",
		)
		return Ok{}, &InsufficientPermissionsError{}
	}

	err = validate_sample_table_column_labels(data_schema.Schema)
	if err != nil {
		s.logger.With("error", err).Error("invalid data schema column labels")
		return Ok{}, err
	}

	schema_id, err := s.data_schema_create(user_id, data_schema)
	if err != nil {
		return Ok{}, err
	}

	switch data_schema.Storage {
	case DATA_STORAGE_INTERNAL:
		err = s.data_schema_storage_table_internal_create(schema_id, data_schema.Schema)
		if err != nil {
			return Ok{}, err
		}
	case DATA_STORAGE_EXTERNAL:
		err = s.data_schema_storage_table_external_create(schema_id)
		if err != nil {
			return Ok{}, err
		}
	default:
		panic(fmt.Sprintf("invalid data storage %s", data_schema.Storage))
	}

	return Ok{}, nil
}

func (s *DataService) data_schema_create(user_id uuid.UUID, data_schema DataSchemaCreate) (uuid.UUID, error) {
	create_schema_query := "INSERT INTO data_schema_ (_creator, _schema, _storage, label, description) VALUES ($1, $2, $3, $4, $5) RETURNING _id"
	var schema_id uuid.UUID
	err := s.db.conn.QueryRow(
		s.ctx,
		create_schema_query,
		user_id,
		data_schema.Schema,
		data_schema.Storage,
		data_schema.Label,
		data_schema.Description,
	).Scan(&schema_id)

	if err != nil {
		s.logger.With("error", err).Error("could not create data schema")
		return uuid.Nil, err
	}

	return schema_id, nil
}

func sample_data_table_name_from_schema_id(schema_id uuid.UUID) string {
	const SAMPLE_DATA_TABLE_NAME_PREFIX = "sample_data"
	schema_name := strings.ReplaceAll(schema_id.String(), "-", "_")
	return fmt.Sprintf(
		"%s_%s_",
		SAMPLE_DATA_TABLE_NAME_PREFIX,
		schema_name,
	)
}

func (s *DataService) data_schema_storage_table_internal_create(schema_id uuid.UUID, schema []ColumnSchema) error {
	table_cols := make([]string, len(schema))
	for idx, col := range schema {
		var dtype string
		switch col.DType {
		case DATA_TYPE_BOOLEAN:
			dtype = "BOOLEAN[]"
		case DATA_TYPE_FLOAT:
			dtype = "DOUBLE PRECISION[]"
		case DATA_TYPE_INT:
			dtype = "INTEGER[]"
		case DATA_TYPE_UINT:
			dtype = "INTEGER[]"
		case DATA_TYPE_STRING:
			dtype = "TEXT[]"
		case DATA_TYPE_TIMESTAMP:
			dtype = "TIMESTAMP WITH TIME ZONE[]"
		default:
			panic(fmt.Sprintf("unexpected datatype %s for column %s", col.DType, col.Label))
		}

		table_cols[idx] = fmt.Sprintf(
			"%s %s NOT NULL",
			col.Label,
			dtype,
		)

	}

	table_name := sample_data_table_name_from_schema_id(schema_id)
	create_table_query := fmt.Sprintf(
		"CREATE TABLE %s (_sample_data UUID REFERENCES sample_data_(_id), %s)",
		table_name,
		strings.Join(table_cols, ", "),
	)

	table_col_constraints := make([]string, len(schema)-1)
	for idx, col := range schema[1:] {
		table_col_constraints[idx] = fmt.Sprintf(
			"array_length(%s, 1) = array_length(%s, 1)",
			schema[0].Label,
			col.Label,
		)
	}
	table_constraint_query :=
		fmt.Sprintf(
			"ALTER TABLE %s ADD CONSTRAINT equal_column_length_check CHECK (%s)",
			table_name,
			strings.Join(table_col_constraints, " AND "),
		)

	tx, err := s.db.conn.Begin(s.ctx)
	if err != nil {

	}
	defer tx.Rollback(s.ctx)

	_, err = tx.Exec(s.ctx, create_table_query)
	if err != nil {
		s.logger.With("error", err, "schema", schema_id).Error("could not create data table for schema")
		return err
	}

	_, err = tx.Exec(s.ctx, table_constraint_query)
	if err != nil {
		s.logger.With("error", err, "schema", schema_id).Error("could not create data table constraint for schema")
		return err
	}

	tx.Commit(s.ctx)
	return nil
}

const SAMPLE_DATA_STORAGE_TABLE_EXTERNAL_COL_LABEL = "path"

func (s *DataService) data_schema_storage_table_external_create(schema_id uuid.UUID) error {
	create_table_query := fmt.Sprintf(
		`CREATE TABLE $1 (
			_sample_data UUID REFERENCES sample_data_(_id),
			%s VARCHAR(4096)
		)`,
		SAMPLE_DATA_STORAGE_TABLE_EXTERNAL_COL_LABEL,
	)

	_, err := s.db.conn.Exec(
		s.ctx,
		create_table_query,
		sample_data_table_name_from_schema_id(schema_id),
	)
	if err != nil {
		s.logger.With("error", err, "schema", schema_id).Error("could not create data table for schema")
		return err
	}

	return nil
}

func (s *DataService) ParseDataFileToSchema(file_path string, schema_id uuid.UUID) ([]ColumnData, error) {
	file, err := os.Open(file_path)
	if err != nil {
		s.logger.With("error", err, "file", file_path).Error("could not open data file")
		return []ColumnData{}, err
	}

	schema, err := s.get_data_schema(schema_id)
	if err != nil {
		return []ColumnData{}, err
	}

	ext := filepath.Ext(file_path)
	data, err := parse_data_file_to_schema(ext, file, schema)
	if err != nil {
		return []ColumnData{}, err
	}

	return data, nil
}

func (s *DataService) get_data_schema(schema_id uuid.UUID) (DataSchema, error) {
	var schema DataSchema
	query := `SELECT _id, _creator, _schema, _storage, label, description 
				FROM data_schema_ WHERE _id=$1`
	err := s.db.conn.QueryRow(
		s.ctx,
		query,
		schema_id,
	).Scan(
		&schema.Id,
		&schema.Creator,
		&schema.Schema,
		&schema.Storage,
		&schema.Label,
		&schema.Description,
	)
	if err != nil {
		s.logger.With("error", err, "schema", schema_id).Error("could not get data schema")
		return DataSchema{}, err
	}

	return schema, nil
}

type InvalidFileExtensionError struct{}

func (e *InvalidFileExtensionError) Error() string {
	return "INVALID_FILE_EXTENSION"
}

type IncompatibleDataSizeError struct {
	expected int
	found    int
}

func (e *IncompatibleDataSizeError) Error() string {
	return "INCOMPATIBLE_DATA_SIZE"
}

type ColumnData struct {
	Label string
	DType DataType
	Data  []any
}

func parse_data_file_to_schema(ext string, file *os.File, schema DataSchema) ([]ColumnData, error) {
	switch ext {
	case ".csv", ".tsv":
		return parse_data_file_to_schema_csv(file, schema)
	default:
		return []ColumnData{}, &InvalidFileExtensionError{}
	}
}

type ParseCsvError struct {
	Record uint
	Column int
	Err    error
}

func (e *ParseCsvError) Error() string {
	return fmt.Sprintf(
		"record %d, column %d: `%v`",
		e.Record,
		e.Column,
		e.Err,
	)
}

type InvalidDataTypeError struct {
	Value string
	DType DataType
}

func (e *InvalidDataTypeError) Error() string {
	return fmt.Sprintf(
		"invalid data type, could not parse `%s` as `%s`",
		e.Value,
		e.DType,
	)
}

func parse_data_file_to_schema_csv(file *os.File, schema DataSchema) ([]ColumnData, error) {
	reader := csv.NewReader(file)
	var record_idx uint = 0
	errs := []ParseCsvError{}
	data := make([]ColumnData, len(schema.Schema))

	for idx := range data {
		data[idx].Label = schema.Schema[idx].Label
		data[idx].DType = schema.Schema[idx].DType
	}

	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			errs = append(errs, ParseCsvError{Record: record_idx, Column: 0, Err: err})
			continue
		}

		if len(record) != len(schema.Schema) {
			return []ColumnData{}, &IncompatibleDataSizeError{expected: len(schema.Schema), found: len(record)}
		}

		for idx, val_str := range record {
			switch schema.Schema[idx].DType {

			case DATA_TYPE_BOOLEAN:
				var val bool
				switch val_str {
				case "true":
					val = true
				case "false":
					val = false
				default:
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				data[idx].Data = append(data[idx].Data, val)

			case DATA_TYPE_FLOAT:
				val, err := strconv.ParseFloat(val_str, 64)
				if err != nil {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				data[idx].Data = append(data[idx].Data, val)

			case DATA_TYPE_INT:
				val, err := strconv.ParseInt(val_str, 0, 32)
				if err != nil {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				data[idx].Data = append(data[idx].Data, val)
			case DATA_TYPE_UINT:
				val, err := strconv.ParseInt(val_str, 0, 32)
				if err != nil {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				if val < 0 {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: errors.New("value less than 0")})
					continue
				}
				data[idx].Data = append(data[idx].Data, uint(val))
			case DATA_TYPE_STRING:
				data[idx].Data = append(data[idx].Data, val_str)
			case DATA_TYPE_TIMESTAMP:
				val, err := time.Parse(time.RFC3339, val_str)
				if err != nil {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				data[idx].Data = append(data[idx].Data, val)
			default:
				return []ColumnData{}, errors.New("unexpected app.DataType")
			}
		}
	}

	if len(errs) > 0 {
		err_msgs := make([]string, len(errs))
		for idx, err := range errs {
			err_msgs[idx] = err.Error()
		}
		msg := fmt.Sprintf("invalid data file: [%s]", strings.Join(err_msgs, ", "))
		return []ColumnData{}, errors.New(msg)
	}

	return data, nil
}

// StoredData represents teh actual data stored for a sample data.
// Data is []ColumnData if Storage is `internal`.
// Data is a string if Storage is `file`.
type StoredData struct {
	SampleData uuid.UUID
	Storage    DataStorage
	Data       any
}

// GetSampleDataStored gets the data associated with sample data entries.
func (s *DataService) GetSampleDataStoredById(sample_data_ids []uuid.UUID) ([]StoredData, error) {
	if len(sample_data_ids) == 0 {
		return []StoredData{}, nil
	}

	type SampleDataSchema struct {
		SampleData uuid.UUID
		DataSchema uuid.UUID
	}
	rows, _ := s.db.conn.Query(
		s.ctx,
		"SELECT _id, _schema FROM sample_data_ WHERE _id=ANY($1)",
		sample_data_ids,
	)
	sample_data_schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleDataSchema, error) {
		var record SampleDataSchema
		err := row.Scan(&record.SampleData, &record.DataSchema)
		return record, err
	})

	if err != nil {
		s.logger.With(
			"error", err,
			"sample data", sample_data_ids,
		).Error("could not get sample data schemas")

		return nil, err
	}

	data_schema_ids := make([]uuid.UUID, len(sample_data_ids))
	for _, record := range sample_data_schemas {
		if slices.Index(data_schema_ids, record.DataSchema) < 0 {
			data_schema_ids = append(data_schema_ids, record.DataSchema)
		}
	}

	type DataSchemaRecord struct {
		Id      uuid.UUID
		Storage DataStorage
		Schema  []ColumnSchema
	}

	rows, err = s.db.conn.Query(
		s.ctx,
		"SELECT _id, _storage, _schema FROM data_schema_ WHERE _id=ANY($1)",
		data_schema_ids,
	)

	data_schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchemaRecord, error) {
		var record DataSchemaRecord
		err := row.Scan(&record.Id, &record.Storage, &record.Schema)
		return record, err
	})
	if err != nil {
		s.logger.With(
			"error", err,
			"data schemas", data_schema_ids,
		).Error("could not get data schemas")

		return nil, err
	}

	stored_data := make([]StoredData, len(sample_data_schemas))
	for idx, sample_data_schema := range sample_data_schemas {
		data_schema_idx := slices.IndexFunc(data_schemas, func(data_schema DataSchemaRecord) bool {
			return data_schema.Id == sample_data_schema.DataSchema
		})

		data_schema := data_schemas[data_schema_idx]
		var data any
		switch data_schema.Storage {
		case DATA_STORAGE_EXTERNAL:
			data, err = s.get_sample_data_stored_by_id_storage_external_data(
				sample_data_schema.SampleData,
				sample_data_schema.DataSchema,
			)
			if err != nil {
				s.logger.With(
					"error", err,
					"sample data", sample_data_schema.SampleData,
					"data schema", data_schema,
				).Error("could not get stored sample data")
				return nil, err
			}
		case DATA_STORAGE_INTERNAL:
			data, err = s.get_sample_data_stored_by_id_storage_internal_data(
				sample_data_schema.SampleData,
				sample_data_schema.DataSchema,
				data_schema.Schema,
			)
			if err != nil {
				s.logger.With(
					"error", err,
					"sample data", sample_data_schema.SampleData,
					"data schema", data_schema,
				).Error("could not get stored sample data")
				return nil, err
			}
		}
		stored_data[idx] = StoredData{
			SampleData: sample_data_schema.SampleData,
			Storage:    data_schema.Storage,
			Data:       data,
		}
	}

	return stored_data, nil
}

// get_sample_data_stored_by_id_storage_external_data gets the file path of a sample data
// with file storage
func (s *DataService) get_sample_data_stored_by_id_storage_external_data(
	sample_data_id uuid.UUID,
	data_schema_id uuid.UUID,
) (string, error) {
	var file_path string
	data_query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE _sample_data=$1",
		SAMPLE_DATA_STORAGE_TABLE_EXTERNAL_COL_LABEL,
		sample_data_table_name_from_schema_id(data_schema_id),
	)
	err := s.db.conn.QueryRow(
		s.ctx,
		data_query,
		sample_data_id,
	).Scan(file_path)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", data_query,
			"sample data", sample_data_id,
		).Error("could not get stored data")
		return "", err
	}

	return file_path, nil
}

func (s *DataService) get_sample_data_stored_by_id_storage_internal_data(
	sample_data_id uuid.UUID,
	data_schema_id uuid.UUID,
	data_schema []ColumnSchema,
) ([]ColumnData, error) {
	column_labels := make([]string, len(data_schema))
	for idx, col := range data_schema {
		column_labels[idx] = col.Label
	}

	data_query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE _sample_data=$1",
		strings.Join(column_labels, ", "),
		sample_data_table_name_from_schema_id(data_schema_id),
	)
	rows, err := s.db.conn.Query(
		s.ctx,
		data_query,
		sample_data_id,
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", data_query,
			"sample data", sample_data_id,
		).Error("could not get stored data")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		s.logger.With(
			"query", data_query,
			"sample data", sample_data_id,
		).Error("sample data not found")
		return nil, pgx.ErrNoRows
	}

	field_descs := rows.FieldDescriptions()
	if len(data_schema) != len(field_descs) {
		s.logger.With(
			"data schema", data_schema,
			"field descriptions", field_descs,
		).Error("stored data incompatible with data schema")
		panic("stored data incompatible with data schema")
	}

	col_data := make([]any, len(data_schema))
	scan_target := make([]any, len(data_schema))
	for idx := range col_data {
		scan_target[idx] = &col_data[idx]
	}
	err = rows.Scan(scan_target...)
	if err != nil {
		s.logger.With("error", err).Error("could not collect sample data")
		return nil, err
	}

	col_names := make([]string, len(field_descs))
	for idx, field := range field_descs {
		col_names[idx] = field.Name
	}

	data := make([]ColumnData, len(data_schema))
	for idx, col := range data_schema {
		col_data_idx := slices.Index(col_names, col.Label)
		if col_data_idx < 0 {
			s.logger.With(
				"data schema", data_schema,
				"field description", field_descs,
			).Error("field description incompatible with data schema")
			panic("field description incompatible with data schema")
		}
		data[idx].Label = col.Label
		data[idx].DType = col.DType
		data[idx].Data = col_data[col_data_idx].([]any)
	}

	return data, nil
}

// SaveSampleDataSingle saves a single data to the user's disk.
// Returns the path the user selected.
func (s *DataService) SaveSampleDataSingle(sample_data_id uuid.UUID) (string, error) {
	stored_datas, err := s.GetSampleDataStoredById([]uuid.UUID{sample_data_id})
	if err != nil {
		return "", err
	}
	if len(stored_datas) != 1 {
		s.logger.With("sample data", sample_data_id, "stored data", stored_datas).Error("multiple data found")
		panic("unexpectedly found multiple data")
	}
	stored_data := stored_datas[0]

	var data []byte
	switch stored_data.Storage {
	case DATA_STORAGE_EXTERNAL:
		data, err = s.data_storage_external_get_data(stored_data.Data.(string))
		if err != nil {
			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
			return "", err
		}
	case DATA_STORAGE_INTERNAL:
		data, err = s.data_storage_internal_get_data(stored_data.Data.([]ColumnData))
		if err != nil {
			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
			return "", err
		}
	default:
		panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored_data.Storage))
	}

	return s.fs_service.SaveFileSingle(data, "Save data", []FileFilter{})
}

func (s *DataService) data_storage_internal_get_data(data []ColumnData) ([]byte, error) {
	records := make([][]string, len(data[0].Data))
	for row_idx := range records {
		row := make([]string, len(data))
		for col_idx := range row {
			entry := data[col_idx].Data[row_idx]
			row[col_idx] = fmt.Sprintf("%v", entry)
		}
		records[row_idx] = row
	}

	var data_bytes strings.Builder
	csv_builder := csv.NewWriter(&data_bytes)
	err := csv_builder.WriteAll(records)
	if err != nil {
		s.logger.With("error", err).Error("could not write data to csv")
		return nil, err
	}

	return []byte(data_bytes.String()), nil
}

func (s *DataService) data_storage_external_get_data(file_path string) ([]byte, error) {
	data, err := os.ReadFile(file_path)
	if err != nil {
		s.logger.With(
			"error", err,
			"file path", file_path,
		).Error("could not read file data")

		return nil, err
	}

	return data, nil
}

// SaveSampleDataMultiple saves multiple data into a zip archive.
// It returns the path of the save location.
func (s *DataService) SaveSampleDataMultiple(
	sample_data []uuid.UUID,
	project uuid.UUID,
	data_hierarchy []SaveDataHierarchy,
) (string, error) {
	if len(sample_data) == 0 {
		return "", nil
	}

	stored_data, err := s.GetSampleDataStoredById(sample_data)
	if err != nil {
		return "", err
	}
	if len(stored_data) != len(sample_data) {
		s.logger.With("sample data", sample_data, "stored data", stored_data).Error("incompatible number of data found")
		panic("found invalid number of data")
	}

	type SampleDataInfo struct {
		SampleData uuid.UUID
		Sample     uuid.UUID
		DataSchema uuid.UUID
		Timestamp  time.Time
	}
	data_sample_query := "SELECT _id, _sample, _schema, timestamp FROM sample_data_ WHERE _id=ANY($1)"
	rows, _ := s.db.conn.Query(s.ctx, data_sample_query, sample_data)
	sample_data_info, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleDataInfo, error) {
		var record SampleDataInfo
		err := row.Scan(&record.SampleData, &record.Sample, &record.DataSchema, &record.Timestamp)
		return record, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not retrive data samples")
		return "", err
	}

	type SampleInfo struct {
		Id    uuid.UUID
		Label string
	}
	var sample_ids []uuid.UUID
	for _, data_sample := range sample_data_info {
		if !slices.Contains(sample_ids, data_sample.Sample) {
			sample_ids = append(sample_ids, data_sample.Sample)
		}
	}
	sample_label_query := "SELECT _sample, label FROM project_sample_membership_ where _project=$1 AND _sample=ANY($2)"
	rows, err = s.db.conn.Query(s.ctx, sample_label_query, project, sample_ids)
	if err != nil {
		s.logger.With("error", err).Error("could not get sample labels")
		return "", err
	}
	sample_info, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleInfo, error) {
		var info SampleInfo
		err = rows.Scan(&info.Id, &info.Label)
		return info, err
	})
	if err != nil {
		s.logger.With("error", err, "samples", sample_ids).Error("could not get sample info")
	}

	type DataSchemaRecord struct {
		Id    uuid.UUID
		Label string
	}
	data_schema_ids := []uuid.UUID{}
	for _, data := range sample_data_info {
		if !slices.Contains(data_schema_ids, data.DataSchema) {
			data_schema_ids = append(data_schema_ids, data.DataSchema)
		}
	}
	schema_query := "SELECT _id, label FROM data_schema_ WHERE _id=ANY($1)"
	rows, _ = s.db.conn.Query(s.ctx, schema_query, data_schema_ids)
	data_schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchemaRecord, error) {
		var record DataSchemaRecord
		err := row.Scan(&record.Id, &record.Label)
		return record, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not get project data schemas")
		return "", err
	}

	buf := new(bytes.Buffer)
	archive := zip.NewWriter(buf)
	for _, stored := range stored_data {
		data_sample_idx := slices.IndexFunc(sample_data_info, func(info SampleDataInfo) bool {
			return info.SampleData == stored.SampleData
		})
		if data_sample_idx < 0 {
			s.logger.With("sample data", stored.SampleData).Error("could not find sample data label record")
			panic("could not find sample data label record")
		}
		data_info := sample_data_info[data_sample_idx]

		sample_info_idx := slices.IndexFunc(sample_info, func(info SampleInfo) bool {
			return info.Id == data_info.Sample
		})
		sample_info := sample_info[sample_info_idx]

		data_schema_idx := slices.IndexFunc(data_schemas, func(record DataSchemaRecord) bool {
			return record.Id == data_info.DataSchema
		})
		data_schema := data_schemas[data_schema_idx]

		var file_name string
		var data []byte
		switch stored.Storage {
		case DATA_STORAGE_EXTERNAL:
			file_path := stored.Data.(string)
			base := filepath.Base(file_path)
			ext := filepath.Ext(base)
			fname := base[:-(len(ext) + 1)]
			file_name = fmt.Sprintf(
				"%s.%s.%s",
				fname,
				stored.SampleData.String(),
				ext,
			)
			data, err = s.data_storage_external_get_data(file_path)
			if err != nil {
				s.logger.With("stored data", stored_data).Error("could not get stored sample data")
				return "", err
			}
		case DATA_STORAGE_INTERNAL:
			file_name = fmt.Sprintf(
				"%s-%s.%s.csv",
				data_info.Timestamp.Format(time.DateOnly),
				data_info.Timestamp.Format(time.TimeOnly),
				stored.SampleData.String(),
			)
			data, err = s.data_storage_internal_get_data(stored.Data.([]ColumnData))
			if err != nil {
				s.logger.With("stored data", stored_data).Error("could not get stored sample data")
				return "", err
			}
		default:
			panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored.Storage))
		}

		file_path, err := s.save_data_file_path(data_hierarchy, file_name, sample_info.Label, data_schema.Label)
		if err != nil {
			return "", err
		}

		file, err := archive.Create(file_path)
		if err != nil {
			s.logger.With(
				"error", err,
				"sample data", stored.SampleData,
			).Error("could not create archive file")
			return "", err
		}

		_, err = file.Write(data)
		if err != nil {
			s.logger.With(
				"error", err,
				"stored data", stored,
			).Error("could not write data to archive file")
		}
	}

	err = archive.Close()
	if err != nil {
		s.logger.With("error", err).Error("could not close archive")
		return "", nil
	}

	save_filter := FileFilter{
		DisplayName: "ZIP archive",
		Pattern:     "*.zip",
	}
	return s.fs_service.SaveFileSingle(buf.Bytes(), "Save data", []FileFilter{save_filter})
}

// SaveSampleDataMultiple saves multiple data into a zip archive.
// It returns the path of the save location.
func (s *DataService) SaveDataSchemaSampleDataAll(
	data_schema uuid.UUID,
	project uuid.UUID,
	data_hierarchy []SaveDataHierarchy,
) (string, error) {
	type SampleRecord struct {
		Sample     uuid.UUID
		SampleData uuid.UUID
		Label      string
		Timestamp  time.Time
	}
	sample_query :=
		`SELECT sample_._sample, data_._id, sample_.label, data_.timestamp
		FROM project_sample_membership_ as sample_ 
		JOIN sample_data_ as data_ 
		ON sample_._sample=data_._sample
		WHERE sample_._project=$1 AND data_._schema=$2`
	rows, _ := s.db.conn.Query(s.ctx, sample_query, project, data_schema)
	samples, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleRecord, error) {
		var record SampleRecord
		err := row.Scan(&record.Sample, &record.SampleData, &record.Label, &record.Timestamp)
		return record, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not get project data schema samples")
		return "", err
	}
	if len(samples) == 0 {
		return "", nil
	}

	sample_data_ids := make([]uuid.UUID, len(samples))
	for idx, data := range samples {
		sample_data_ids[idx] = data.SampleData
	}
	stored_data, err := s.GetSampleDataStoredById(sample_data_ids)
	if err != nil {
		return "", err
	}
	if len(stored_data) != len(sample_data_ids) {
		s.logger.With(
			"sample data", sample_data_ids,
			"stored data", stored_data,
		).Error("incompatible number of data found")
		panic("found invalid number of data")
	}

	buf := new(bytes.Buffer)
	archive := zip.NewWriter(buf)
	for _, stored := range stored_data {
		sample_idx := slices.IndexFunc(samples, func(record SampleRecord) bool {
			return record.SampleData == stored.SampleData
		})
		if sample_idx < 0 {
			s.logger.With("sample data", stored.SampleData).Error("could not find sample data label record")
			panic("could not find sample data label record")
		}
		sample_info := samples[sample_idx]

		var file_name string
		var data []byte
		switch stored.Storage {
		case DATA_STORAGE_EXTERNAL:
			file_path := stored.Data.(string)
			base := filepath.Base(file_path)
			ext := filepath.Ext(base)
			fname := base[:-(len(ext) + 1)]
			file_name = fmt.Sprintf(
				"%s.%s.%s",
				fname,
				stored.SampleData.String(),
				ext,
			)
			data, err = s.data_storage_external_get_data(file_path)
			if err != nil {
				s.logger.With("stored data", stored_data).Error("could not get stored sample data")
				return "", err
			}
		case DATA_STORAGE_INTERNAL:
			file_name = fmt.Sprintf(
				"%s-%s.%s.csv",
				sample_info.Timestamp.Format(time.DateOnly),
				sample_info.Timestamp.Format(time.TimeOnly),
				stored.SampleData.String(),
			)
			data, err = s.data_storage_internal_get_data(stored.Data.([]ColumnData))
			if err != nil {
				s.logger.With("stored data", stored_data).Error("could not get stored sample data")
				return "", err
			}
		default:
			panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored.Storage))
		}

		file_path, err := s.save_data_schema_sample_data_file_path(data_hierarchy, file_name, sample_info.Label)
		if err != nil {
			return "", err
		}

		file, err := archive.Create(file_path)
		if err != nil {
			s.logger.With(
				"error", err,
				"sample data", stored.SampleData,
			).Error("could not create archive file")
			return "", err
		}

		_, err = file.Write(data)
		if err != nil {
			s.logger.With(
				"error", err,
				"stored data", stored,
			).Error("could not write data to archive file")
		}
	}

	err = archive.Close()
	if err != nil {
		s.logger.With("error", err).Error("could not close archive")
		return "", nil
	}

	save_filter := FileFilter{
		DisplayName: "ZIP archive",
		Pattern:     "*.zip",
	}
	return s.fs_service.SaveFileSingle(buf.Bytes(), "Save data", []FileFilter{save_filter})
}

func (s *DataService) save_data_schema_sample_data_file_path(
	hierarchy []SaveDataHierarchy,
	file_name_base string,
	sample_label string,
) (string, error) {
	hierarchy_components := map[SaveDataHierarchy]string{
		SAVE_DATA_HIERARCHY_SAMPLE: sample_label,
	}
	var file_path strings.Builder
	for _, level := range hierarchy {
		component, present := hierarchy_components[level]
		if !present {
			s.logger.With("levels", hierarchy).Error("repeated save data hierarchy level")
			return "", errors.New("invalid save data hierarchy, repeated level")
		}
		file_path.WriteString(component)
		file_path.WriteString("/")
		delete(hierarchy_components, level)
	}

	sample_label, file_name_sample := hierarchy_components[SAVE_DATA_HIERARCHY_SAMPLE]
	if file_name_sample {
		file_path.WriteString(sample_label)
		file_path.WriteString(".")
	}
	schema_label, file_name_schema := hierarchy_components[SAVE_DATA_HIERARCHY_DATA_SCHEMA]
	if file_name_schema {
		file_path.WriteString(schema_label)
		file_path.WriteString(".")
	}
	file_path.WriteString(file_name_base)

	return file_path.String(), nil
}

// SaveProjectDataAll saves all sample data in a project into a zip archive.
// It returns the path of the save location.
func (s *DataService) SaveProjectDataAll(project uuid.UUID, hierarchy []SaveDataHierarchy) (string, error) {
	type ProjectSampleRecord struct {
		Sample uuid.UUID
		Label  string
	}
	sample_query := "SELECT _sample, label FROM project_sample_membership_ WHERE _project=$1"
	rows, _ := s.db.conn.Query(s.ctx, sample_query, project)
	project_samples, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ProjectSampleRecord, error) {
		var record ProjectSampleRecord
		err := row.Scan(&record.Sample, &record.Label)
		return record, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not get project samples")
		return "", err
	}
	if len(project_samples) == 0 {
		return "", nil
	}

	type SampleDataRecord struct {
		Id         uuid.UUID
		Sample     uuid.UUID
		DataSchema uuid.UUID
		Timestamp  time.Time
	}
	sample_ids := make([]uuid.UUID, len(project_samples))
	for idx, sample := range project_samples {
		sample_ids[idx] = sample.Sample
	}
	data_query := "SELECT _id, _sample, _schema, timestamp FROM sample_data_ WHERE _sample=ANY($1)"
	rows, _ = s.db.conn.Query(s.ctx, data_query, sample_ids)
	sample_data, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleDataRecord, error) {
		var record SampleDataRecord
		err := row.Scan(&record.Id, &record.Sample, &record.DataSchema, &record.Timestamp)
		return record, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not get project sample data")
		return "", err
	}

	type DataSchemaRecord struct {
		Id    uuid.UUID
		Label string
	}
	data_schema_ids := []uuid.UUID{}
	for _, data := range sample_data {
		if !slices.Contains(data_schema_ids, data.DataSchema) {
			data_schema_ids = append(data_schema_ids, data.DataSchema)
		}
	}
	schema_query := "SELECT _id, label FROM data_schema_ WHERE _id=ANY($1)"
	rows, _ = s.db.conn.Query(s.ctx, schema_query, data_schema_ids)
	data_schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchemaRecord, error) {
		var record DataSchemaRecord
		err := row.Scan(&record.Id, &record.Label)
		return record, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not get project data schemas")
		return "", err
	}

	sample_data_ids := make([]uuid.UUID, len(sample_data))
	for idx, data := range sample_data {
		sample_data_ids[idx] = data.Id
	}
	stored_data, err := s.GetSampleDataStoredById(sample_data_ids)
	if err != nil {
		return "", err
	}
	if len(stored_data) != len(sample_data) {
		s.logger.With("sample data", sample_data, "stored data", stored_data).Error("incompatible number of data found")
		panic("found invalid number of data")
	}

	buf := new(bytes.Buffer)
	archive := zip.NewWriter(buf)
	for _, stored := range stored_data {
		data_sample_idx := slices.IndexFunc(sample_data, func(record SampleDataRecord) bool {
			return record.Id == stored.SampleData
		})
		if data_sample_idx < 0 {
			s.logger.With("sample data", stored.SampleData).Error("could not find sample data label record")
			panic("could not find sample data label record")
		}
		data_info := sample_data[data_sample_idx]

		project_sample_idx := slices.IndexFunc(project_samples, func(record ProjectSampleRecord) bool {
			return record.Sample == data_info.Sample
		})
		project_sample := project_samples[project_sample_idx]

		data_schema_idx := slices.IndexFunc(data_schemas, func(record DataSchemaRecord) bool {
			return record.Id == data_info.DataSchema
		})
		data_schema := data_schemas[data_schema_idx]

		var file_name string
		var data []byte
		switch stored.Storage {
		case DATA_STORAGE_EXTERNAL:
			file_path := stored.Data.(string)
			base := filepath.Base(file_path)
			ext := filepath.Ext(base)
			fname := base[:-(len(ext) + 1)]
			file_name = fmt.Sprintf(
				"%s.%s.%s",
				fname,
				stored.SampleData.String(),
				ext,
			)
			data, err = s.data_storage_external_get_data(file_path)
			if err != nil {
				s.logger.With("stored data", stored_data).Error("could not get stored sample data")
				return "", err
			}
		case DATA_STORAGE_INTERNAL:
			file_name = fmt.Sprintf(
				"%s-%s.%s.csv",
				data_info.Timestamp.Format(time.DateOnly),
				data_info.Timestamp.Format(time.TimeOnly),
				stored.SampleData.String(),
			)
			data, err = s.data_storage_internal_get_data(stored.Data.([]ColumnData))
			if err != nil {
				s.logger.With("stored data", stored_data).Error("could not get stored sample data")
				return "", err
			}
		default:
			panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored.Storage))
		}

		file_path, err := s.save_data_file_path(hierarchy, file_name, project_sample.Label, data_schema.Label)
		if err != nil {
			return "", err
		}

		file, err := archive.Create(file_path)
		if err != nil {
			s.logger.With(
				"error", err,
				"sample data", stored.SampleData,
			).Error("could not create archive file")
			return "", err
		}

		_, err = file.Write(data)
		if err != nil {
			s.logger.With(
				"error", err,
				"stored data", stored,
			).Error("could not write data to archive file")
		}
	}

	err = archive.Close()
	if err != nil {
		s.logger.With("error", err).Error("could not close archive")
		return "", nil
	}

	save_filter := FileFilter{
		DisplayName: "ZIP archive",
		Pattern:     "*.zip",
	}
	return s.fs_service.SaveFileSingle(buf.Bytes(), "Save data", []FileFilter{save_filter})
}

func (s *DataService) save_data_file_path(
	hierarchy []SaveDataHierarchy,
	file_name_base string,
	sample_label string,
	data_schema_label string,
) (string, error) {
	hierarchy_components := map[SaveDataHierarchy]string{
		SAVE_DATA_HIERARCHY_DATA_SCHEMA: data_schema_label,
		SAVE_DATA_HIERARCHY_SAMPLE:      sample_label,
	}
	var file_path strings.Builder
	for _, level := range hierarchy {
		component, present := hierarchy_components[level]
		if !present {
			s.logger.With("levels", hierarchy).Error("repeated save data hierarchy level")
			return "", errors.New("invalid save data hierarchy, repeated level")
		}
		file_path.WriteString(component)
		file_path.WriteString("/")
		delete(hierarchy_components, level)
	}

	sample_label, file_name_sample := hierarchy_components[SAVE_DATA_HIERARCHY_SAMPLE]
	if file_name_sample {
		file_path.WriteString(sample_label)
		file_path.WriteString(".")
	}
	schema_label, file_name_schema := hierarchy_components[SAVE_DATA_HIERARCHY_DATA_SCHEMA]
	if file_name_schema {
		file_path.WriteString(schema_label)
		file_path.WriteString(".")
	}
	file_path.WriteString(file_name_base)

	return file_path.String(), nil
}
