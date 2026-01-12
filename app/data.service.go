package app

import (
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

const (
	DATA_TYPE_STRING    = DataType("string")
	DATA_TYPE_INT       = DataType("int")
	DATA_TYPE_UINT      = DataType("uint")
	DATA_TYPE_FLOAT     = DataType("float")
	DATA_TYPE_BOOLEAN   = DataType("boolean")
	DATA_TYPE_TIMESTAMP = DataType("timestamp")
)

type DataType string

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

const (
	DATA_STORAGE_INTERNAL = DataStorage("internal")
	DATA_STORAGE_FILE     = DataStorage("file")
)

type DataStorage string

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
	case DATA_STORAGE_FILE:
		err = s.data_schema_storage_table_file_create(schema_id)
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

const SAMPLE_DATA_STORAGE_TABLE_FILE_COL_LABEL = "file_path"

func (s *DataService) data_schema_storage_table_file_create(schema_id uuid.UUID) error {
	create_table_query := fmt.Sprintf(
		`CREATE TABLE $1 (
			_sample_data UUID REFERENCES sample_data_(_id),
			%s VARCHAR(4096)
		)`,
		SAMPLE_DATA_STORAGE_TABLE_FILE_COL_LABEL,
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
	Storage DataStorage
	Data    any
}

// GetSampleDataStored gets the actual data associated with a sample data entry.
func (s *DataService) GetSampleDataStoredById(sample_data_id uuid.UUID) (StoredData, error) {
	var schema_id uuid.UUID
	err := s.db.conn.QueryRow(
		s.ctx,
		"SELECT _schema FROM sample_data_ WHERE _id=$1",
		sample_data_id,
	).Scan(&schema_id)
	if err != nil {
		s.logger.With(
			"error", err,
			"sample data", sample_data_id,
		).Error("could not get sample data schema")

		return StoredData{}, err
	}

	var data_storage DataStorage
	var data_schema []ColumnSchema
	err = s.db.conn.QueryRow(
		s.ctx,
		"SELECT _storage, _schema FROM data_schema_ WHERE _id=$1",
		schema_id,
	).Scan(&data_storage, &data_schema)
	if err != nil {
		s.logger.With(
			"error", err,
			"data schema", schema_id,
		).Error("could not get data schema")

		return StoredData{}, err
	}

	var data any
	switch data_storage {
	case DATA_STORAGE_FILE:
		data, err = s.get_sample_data_stored_by_id_storage_file_data(sample_data_id, schema_id)
		if err != nil {
			return StoredData{}, err
		}
	case DATA_STORAGE_INTERNAL:
		data, err = s.get_sample_data_stored_by_id_storage_internal_data(sample_data_id, schema_id, data_schema)
		if err != nil {
			return StoredData{}, err
		}
	}

	stored_data := StoredData{
		Storage: data_storage,
		Data:    data,
	}
	return stored_data, nil
}

// get_sample_data_stored_by_id_storage_file_data gets the file path of a sample data
// with file storage
func (s *DataService) get_sample_data_stored_by_id_storage_file_data(
	sample_data_id uuid.UUID,
	data_schema_id uuid.UUID,
) (string, error) {
	var file_path string
	data_query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE _sample_data=$1",
		SAMPLE_DATA_STORAGE_TABLE_FILE_COL_LABEL,
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
	stored_data, err := s.GetSampleDataStoredById(sample_data_id)
	if err != nil {
		return "", err
	}

	switch stored_data.Storage {
	case DATA_STORAGE_FILE:
		return s.save_sample_data_single_data_storage_file(stored_data.Data.(string))
	case DATA_STORAGE_INTERNAL:
		return s.save_sample_data_single_data_storage_internal(stored_data.Data.([]ColumnData))
	default:
		panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored_data.Storage))
	}
}

func (s *DataService) save_sample_data_single_data_storage_internal(data []ColumnData) (string, error) {
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
		return "", err
	}

	return s.fs_service.SaveFileSingle([]byte(data_bytes.String()), "Save data", []FileFilter{})
}

func (s *DataService) save_sample_data_single_data_storage_file(file_path string) (string, error) {
	data, err := os.ReadFile(file_path)
	if err != nil {
		s.logger.With(
			"error", err,
			"file path", file_path,
		).Error("could not read file data")

		return "", err
	}

	s.logger.Debug("", "data", data)
	return "", nil
}
