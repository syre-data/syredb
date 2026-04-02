package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syredb/database"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ValueType string

const (
	ValueTypeString    ValueType = "string"
	ValueTypeInt       ValueType = "int"
	ValueTypeUint      ValueType = "uint"
	ValueTypeFloat     ValueType = "float"
	ValueTypeBoolean   ValueType = "boolean"
	ValueTypeTimestamp ValueType = "timestamp"
)

type SaveDataHierarchy string

const (
	SaveDataHierarchyFlat             SaveDataHierarchy = "flat"
	SaveDataHierarchyDataSchema       SaveDataHierarchy = "data_schema"
	SaveDataHierarchySample           SaveDataHierarchy = "sample"
	SaveDataHierarchySampleDataSchema SaveDataHierarchy = "sample-data_scmeha"
	SaveDataHierarchyDataSchemaSample SaveDataHierarchy = "data_schema-sample"
)

func ParseSaveDataHierarchy(value string) (SaveDataHierarchy, error) {
	switch value {
	case "flat":
		return SaveDataHierarchyFlat, nil
	case "data_schema":
		return SaveDataHierarchyDataSchema, nil
	case "sample":
		return SaveDataHierarchySample, nil
	case "sample-data_scmeha":
		return SaveDataHierarchySampleDataSchema, nil
	case "data_schema-sample":
		return SaveDataHierarchyDataSchemaSample, nil
	default:
		return "", errors.New("invalid value")
	}
}

type DataService struct {
	ctx          context.Context
	logger       *slog.Logger
	db           *database.DBConnection
	app_service  *AppService
	user_service *UserService
}

func NewDataService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DBConnection,
	app_service *AppService,
	user_service *UserService,
) *DataService {
	return &DataService{
		ctx:          ctx,
		logger:       logger,
		db:           db,
		app_service:  app_service,
		user_service: user_service,
	}
}

type DataStorage string

const (
	DataStorageInternal DataStorage = "internal"
	DataStorageExternal DataStorage = "external"
)

type DataSourceCardinality string

const (
	DataSourceCardinalitySingle   DataSourceCardinality = "single"
	DataSourceCardinalityMultiple DataSourceCardinality = "multiple"
)

type DataTypeSourceRecord struct {
	Id              uuid.UUID             `db:"_id"`
	DataType        uuid.UUID             `db:"_data_type"`
	Cardinality     DataSourceCardinality `db:"_cardinality"`
	Required        bool                  `db:"_required"`
	ExtensionFilter []string              `db:"extension_filter"`
	Label           string                `db:"label"`
	Description     string                `db:"description"`
}

type DataTypeRecord struct {
	Id          uuid.UUID `db:"_id"`
	Creator     uuid.UUID `db:"_creator"`
	Recipe      uuid.UUID `db:"recipe"`
	Schema      uuid.UUID `db:"_schema"`
	Label       string    `db:"label"`
	Description string    `db:"description"`
	Active      bool      `db:"active"`
}

type DataType struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Schema      uuid.UUID
	Recipe      uuid.UUID
	Label       string
	Description string
	Active      bool
	Sources     []DataTypeSourceRecord
}

func (s *DataService) DataTypesGetAll() ([]DataType, error) {
	data_type_query :=
		`SELECT _id, _creator, _schema, recipe, label, description, active
		FROM data_type_ ORDER BY label`
	rows, _ := s.db.Conn.Query(s.ctx, data_type_query)
	data_type_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not get data types")
		return nil, err
	}

	data_types := make([]DataType, len(data_type_rxs))
	for idx, rx := range data_type_rxs {
		data_types[idx].Id = rx.Id
		data_types[idx].Creator = rx.Creator
		data_types[idx].Schema = rx.Schema
		data_types[idx].Recipe = rx.Recipe
		data_types[idx].Label = rx.Label
		data_types[idx].Description = rx.Description
		data_types[idx].Active = rx.Active
	}

	data_type_ids := make([]uuid.UUID, len(data_types))
	for idx, data_type := range data_types {
		data_type_ids[idx] = data_type.Id
	}
	source_query :=
		`SELECT _id, _data_type, _cardinality, _required, extension_filter, label, description
		FROM data_type_source_ WHERE _data_type=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, source_query, data_type_ids)
	sources, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeSourceRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not get data type sources")
		return nil, err
	}

	for _, source := range sources {
		data_type_idx := slices.IndexFunc(data_types, func(data_type DataType) bool {
			return source.DataType == data_type.Id
		})
		if data_type_idx < 0 {
			s.logger.With(
				"source", source,
				"data types", data_types,
			).Error("invalid data type source")
			panic("invalid data type source")
		}

		data_types[data_type_idx].Sources = append(data_types[data_type_idx].Sources, source)
	}

	return data_types, nil
}

func (s *DataService) DataTypeGetById(id uuid.UUID) (DataType, error) {
	data_type_query :=
		`SELECT _id, _creator, _schema, recipe, label, description, active
		FROM data_type_ WHERE _id=$1`
	rows, _ := s.db.Conn.Query(s.ctx, data_type_query, id)
	rx, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataTypeRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not get data type")
		return DataType{}, err
	}

	data_type := DataType{
		Id:          rx.Id,
		Creator:     rx.Creator,
		Schema:      rx.Schema,
		Recipe:      rx.Recipe,
		Label:       rx.Label,
		Description: rx.Description,
		Active:      rx.Active,
	}

	source_query :=
		`SELECT _id, _data_type, _cardinality, _required, extension_filter, label, description
		FROM data_type_source_ WHERE _data_type=$1`
	rows, _ = s.db.Conn.Query(s.ctx, source_query, id)
	sources, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeSourceRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not get data type sources")
		return DataType{}, err
	}

	for _, source := range sources {
		data_type.Sources = append(data_type.Sources, source)
	}

	return data_type, nil
}

func (s *DataService) DataTypeRecipeCreate(tx pgx.Tx, recipe *multipart.FileHeader) (uuid.UUID, error) {
	recipe_dir, err := s.app_service.AppDataDir(AppDataDirRecipe)
	if err != nil {
		s.logger.With(
			"error", err,
			"dir", AppDataDirRecipe,
		).Error("could not get app recipe dir")
		return uuid.Nil, err
	}

	filename := fmt.Sprintf(
		"%s.%s",
		rand.Text(),
		recipe.Filename,
	)
	path := filepath.Join(recipe_dir, filename)

	var id uuid.UUID
	query :=
		`INSERT INTO data_type_recipe_ (_path, _cmd, _args) VALUES ($1, $2, $3)
		RETURNING _id`
	err = tx.QueryRow(s.ctx, query, path, "python", []string{}).Scan(&id)
	if err != nil {
		s.logger.With(
			"error", err,
			"recipe", recipe,
		).Error("could not create data type recipe")
		return uuid.Nil, err
	}

	err = SaveFormFile(recipe, path)
	if err != nil {
		s.logger.With(
			"error", err,
			"path", path,
		).Error("could not save recipe file")
		return uuid.Nil, err
	}

	return id, nil
}

type DataTypeSourceCreate struct {
	Cardinality     DataSourceCardinality
	Required        bool
	ExtensionFilter []string
	Label           string
	Description     *string
}

func (s *DataService) DataTypeSourcesCreate(tx pgx.Tx, data_type uuid.UUID, sources []DataTypeSourceCreate) ([]uuid.UUID, error) {
	if len(sources) == 0 {
		return []uuid.UUID{}, nil
	}

	var query strings.Builder
	query.WriteString(
		`INSERT INTO data_type_source_ 
		(_data_type, _cardinality, _required, extension_filter, label, description)
		VALUES`,
	)

	const FieldsPerRecord = 5
	const RecordOffset = 1
	args := make([]any, len(sources)*FieldsPerRecord+RecordOffset)
	args[0] = data_type
	for idx, source := range sources {
		cardinality_idx := idx*FieldsPerRecord + RecordOffset
		required_idx := cardinality_idx + 1
		extension_filter_idx := required_idx + 1
		label_idx := extension_filter_idx + 1
		description_idx := label_idx + 1

		args[cardinality_idx] = source.Cardinality
		args[required_idx] = source.Required
		args[label_idx] = source.Label
		if len(source.ExtensionFilter) == 0 {
			args[extension_filter_idx] = nil
		} else {
			args[extension_filter_idx] = source.ExtensionFilter
		}
		if source.Description == nil {
			args[description_idx] = nil
		} else {
			args[description_idx] = *source.Description
		}

		if idx > 0 {
			query.WriteString(", ")
		}
		fmt.Fprintf(
			&query,
			"($1, $%d, $%d, $%d, $%d, $%d)",
			cardinality_idx+1,
			required_idx+1,
			extension_filter_idx+1,
			label_idx+1,
			description_idx+1,
		)
	}
	query.WriteString(" RETURNING _id")

	rows, _ := tx.Query(s.ctx, query.String(), args...)
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		s.logger.With(
			"error", err,
			"data type", data_type,
			"sources", sources,
		).Error("could not create data type sources")
		return nil, err
	}

	return ids, nil
}

func (s *DataService) DataTypeCreate(
	creator uuid.UUID,
	label string,
	description *string,
	sources []DataTypeSourceCreate,
	schema uuid.UUID,
	recipe *multipart.FileHeader,
) error {
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return err
	}
	defer tx.Rollback(s.ctx)

	fields := []string{"_creator", "label"}
	value_args := []any{creator, label}

	if description != nil {
		fields = append(fields, "description")
		value_args = append(value_args, *description)
	}

	if recipe != nil {
		recipe_idx, err := s.DataTypeRecipeCreate(tx, recipe)
		if err != nil {
			s.logger.With(
				"error", err,
			).Error("could not save recipe file")
			return err
		}
		fields = append(fields, "recipe")
		value_args = append(value_args, recipe_idx)
	}
	if schema != uuid.Nil {
		fields = append(fields, "_schema")
		value_args = append(value_args, schema)
	}

	var id uuid.UUID
	query := fmt.Sprintf(
		"INSERT INTO data_type_ (%s) VALUES (%s) RETURNING _id",
		strings.Join(fields, ", "),
		SqlArgsPlaceholderList(len(value_args)),
	)
	err = tx.QueryRow(s.ctx, query, value_args...).Scan(&id)
	if err != nil {
		s.logger.With("error", err).Error("could not create data type")
		return err
	}

	_, err = s.DataTypeSourcesCreate(tx, id, sources)
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not create data type sources")
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not commit data type create transaction")
		return err
	}

	return nil
}

type DataTypeSourceUpdate struct {
	Id              uuid.UUID
	Description     string
	ExtensionFilter []string
}

type DataTypeUpdate struct {
	Id          uuid.UUID
	Active      bool
	Label       string
	Description string
	Sources     []DataTypeSourceUpdate
}

func (s *DataService) DataTypeUpdate(update DataTypeUpdate) error {
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return err
	}
	defer tx.Rollback(s.ctx)

	query :=
		`UPDATE data_type_ SET active=$1, label=$2, description=$3 
		WHERE _id=$4`
	_, err = tx.Exec(s.ctx, query, update.Active, update.Label, update.Description, update.Id)
	if err != nil {
		s.logger.With(
			"error", err,
			"update", update,
		).Error("could not update data type")
		return err
	}

	for _, source := range update.Sources {
		err = s.DataTypeSourceUpdate(tx, source)
		if err != nil {
			s.logger.With(
				"error", err,
				"update", source,
			).Error("could not update data type source")
			return err
		}
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With(
			"error", err,
			"update", update,
		).Error("could not commit data type update")
		return err
	}

	return nil
}

func (s *DataService) DataTypeSourceUpdate(tx pgx.Tx, update DataTypeSourceUpdate) error {
	query :=
		`UPDATE data_type_source_ SET description=$1, extension_filter=$2 
		WHERE _id=$3`
	_, err := tx.Exec(s.ctx, query, update.Description, update.ExtensionFilter, update.Id)
	if err != nil {
		s.logger.With(
			"error", err,
			"update", update,
		).Error("could not update data type source")
		return err
	}

	return nil
}

type DataSchemaCardinality string

const (
	DataSchemaCardinalitySingle   DataSchemaCardinality = "single"
	DataSchemaCardinalityMultiple DataSchemaCardinality = "multiple"
)

type DataSchemaRecord struct {
	Id          uuid.UUID             `db:"_id"`
	Creator     uuid.UUID             `db:"_creator"`
	Cardinality DataSchemaCardinality `db:"_cardinality"`
	Label       string                `db:"label"`
	Description string                `db:"description"`
}

type DataSchemaFieldRecord struct {
	Id          uuid.UUID `db:"_id"`
	Label       string    `db:"_label"`
	DType       ValueType `db:"_dtype"`
	Description string    `db:"description"`
}

type DataSchemaField struct {
	Label       string    `db:"_label"`
	DType       ValueType `db:"_dtype"`
	Description string    `db:"description"`
}

type DataSchema struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Cardinality DataSchemaCardinality
	Label       string
	Description string
	Schema      []DataSchemaField
}

func (s *DataService) DataSchemasGetAll() ([]DataSchema, error) {
	schemas_query :=
		`SELECT _id, _creator, _cardinality, label, description
		FROM data_schema_ ORDER BY _id DESC`
	rows, err := s.db.Conn.Query(s.ctx, schemas_query)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schemas")
		return nil, err
	}

	schemas_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schemas")
		return nil, err
	}

	fields_query := `SELECT _id, _label, _dtype, description FROM data_schema_field_`
	rows, err = s.db.Conn.Query(s.ctx, fields_query)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schema fields")
		return nil, err
	}

	fields_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaFieldRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schema fields")
		return nil, err
	}

	schemas := make([]DataSchema, len(schemas_rx))
	for idx, schema_rx := range schemas_rx {
		var fields []DataSchemaField
		for _, field := range fields_rx {
			if field.Id == schema_rx.Id {
				fields = append(fields, DataSchemaField{
					Label:       field.Label,
					DType:       field.DType,
					Description: field.Description,
				})
			}
		}

		schemas[idx] = DataSchema{
			Id:          schema_rx.Id,
			Creator:     schema_rx.Creator,
			Cardinality: schema_rx.Cardinality,
			Label:       schema_rx.Label,
			Description: schema_rx.Description,
			Schema:      fields,
		}
	}

	return schemas, nil
}

func (s *DataService) DataSchemasGetById(schema_ids []uuid.UUID) ([]DataSchema, error) {
	if len(schema_ids) == 0 {
		return nil, nil
	}

	schemas_query := `
		SELECT _id, _creator, _cardinality, label, description
		FROM data_schema_ WHERE _id=ANY($1) ORDER BY _id DESC`
	rows, err := s.db.Conn.Query(s.ctx, schemas_query, schema_ids)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schemas")
		return nil, err
	}

	schemas_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schemas")
		return nil, err
	}

	fields_query :=
		`SELECT _id, _label, _dtype, description 
		FROM data_schema_field_ WHERE _id=ANY($1)`
	rows, err = s.db.Conn.Query(s.ctx, fields_query, schema_ids)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schema fields")
		return nil, err
	}

	fields_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaFieldRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schema fields")
		return nil, err
	}

	schemas := make([]DataSchema, len(schemas_rx))
	for idx, schema_rx := range schemas_rx {
		var fields []DataSchemaField
		for _, field := range fields_rx {
			if field.Id == schema_rx.Id {
				fields = append(fields, DataSchemaField{
					Label:       field.Label,
					DType:       field.DType,
					Description: field.Description,
				})
			}
		}

		schemas[idx] = DataSchema{
			Id:          schema_rx.Id,
			Creator:     schema_rx.Creator,
			Cardinality: schema_rx.Cardinality,
			Label:       schema_rx.Label,
			Description: schema_rx.Description,
			Schema:      fields,
		}
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

func validate_data_schema_storage_table_column_labels(schema []DataSchemaField) error {
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
	Cardinality DataSchemaCardinality
	Schema      []DataSchemaField
	Label       string
	Description string
}

func (s *DataService) DataSchemaCreate(user_id uuid.UUID, data_schema DataSchemaCreate) error {
	has_permission, err := s.user_service.UserHasPermission(user_id, DbPermissionIdDataSchemaCreate)
	if err != nil {
		return err
	}
	if !has_permission {
		s.logger.With("user", user_id).Debug(
			"insufficient permissions to create data schema for user",
		)
		return &InsufficientPermissionsError{}
	}

	err = validate_data_schema_storage_table_column_labels(data_schema.Schema)
	if err != nil {
		s.logger.With("error", err).Error("invalid data schema column labels")
		return err
	}

	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction to create data schema")
		return err
	}
	defer tx.Rollback(s.ctx)

	schema_id, err := s.data_schema_create(tx, user_id, data_schema)
	if err != nil {
		return err
	}

	err = s.data_schema_create_schema(tx, schema_id, data_schema.Schema)
	if err != nil {
		return err
	}

	err = s.data_schema_storage_table_create(
		tx,
		schema_id,
		data_schema.Cardinality,
		data_schema.Schema,
	)
	if err != nil {
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not commit transaction to create data schema")
		return err
	}

	return nil
}

func (s *DataService) data_schema_create(
	tx pgx.Tx,
	user_id uuid.UUID,
	data_schema DataSchemaCreate,
) (uuid.UUID, error) {
	create_schema_query :=
		`INSERT INTO data_schema_ (_creator, _cardinality, label, description) 
		VALUES ($1, $2, $3, $4) RETURNING _id`
	var schema_id uuid.UUID
	err := tx.QueryRow(
		s.ctx,
		create_schema_query,
		user_id,
		data_schema.Cardinality,
		data_schema.Label,
		data_schema.Description,
	).Scan(&schema_id)

	if err != nil {
		s.logger.With("error", err).Error("could not create data schema")
		return uuid.Nil, err
	}

	return schema_id, nil
}

func (s *DataService) data_schema_create_schema(
	tx pgx.Tx,
	schema_id uuid.UUID,
	schema []DataSchemaField,
) error {
	var query strings.Builder
	query.WriteString(
		`INSERT INTO data_schema_field_ 
		(_id, _label, _dtype, description) VALUES `,
	)

	const NumFields = 3
	const ArgsOffset = 1
	args := make([]any, len(schema)*NumFields+ArgsOffset)
	args[0] = schema_id
	for idx, field := range schema {
		if idx > 0 {
			query.WriteString(", ")
		}

		idx_label := idx*NumFields + ArgsOffset
		idx_dtype := idx_label + 1
		idx_description := idx_dtype + 1
		fmt.Fprintf(
			&query,
			"($1, $%d, $%d, $%d)",
			idx_label+1,
			idx_dtype+1,
			idx_description+1,
		)

		description := new(string)
		if len(field.Description) > 0 {
			description = &field.Description
		}

		args[idx_label] = field.Label
		args[idx_dtype] = field.DType
		args[idx_description] = description
	}

	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"schema_id", schema_id,
			"schema", schema,
			"query", query.String(),
			"args", args,
		).Error("could not create data schema fields")
		return err
	}

	return nil
}

func data_storage_table_columns_from_schema(cardinality DataSchemaCardinality, schema []DataSchemaField) []string {
	table_cols := make([]string, len(schema))
	for idx, col := range schema {
		var col_def string
		switch col.DType {
		case ValueTypeBoolean:
			col_def = "BOOLEAN"
		case ValueTypeFloat:
			col_def = "DOUBLE PRECISION"
		case ValueTypeInt:
			col_def = "INTEGER"
		case ValueTypeUint:
			col_def = "INTEGER"
		case ValueTypeString:
			col_def = "TEXT"
		case ValueTypeTimestamp:
			col_def = "TIMESTAMP WITH TIME ZONE"
		default:
			panic(fmt.Sprintf("unexpected value type %s for column %s", col.DType, col.Label))
		}

		if cardinality == DataSchemaCardinalityMultiple {
			col_def += "[]"
		}

		table_cols[idx] = fmt.Sprintf(
			"%s %s NOT NULL",
			col.Label,
			col_def,
		)
	}

	return table_cols
}

func data_storage_table_name_from_schema_id(schema_id uuid.UUID) string {
	const TABLE_NAME_PREFIX = "data_schema"
	schema_name := strings.ReplaceAll(schema_id.String(), "-", "_")
	return fmt.Sprintf(
		"%s_%s_",
		TABLE_NAME_PREFIX,
		schema_name,
	)
}

func data_schema_table_equal_column_length_constraint_query(
	table_name string,
	schema []DataSchemaField,
) string {
	constraints := make([]string, len(schema)-1)
	for idx, col := range schema[1:] {
		constraints[idx] = fmt.Sprintf(
			"array_length(%s, 1) = array_length(%s, 1)",
			schema[0].Label,
			col.Label,
		)
	}

	query :=
		fmt.Sprintf(
			"ALTER TABLE %s ADD CONSTRAINT equal_column_length_check CHECK (%s)",
			table_name,
			strings.Join(constraints, " AND "),
		)

	return query
}

func (s *DataService) data_schema_storage_table_create(
	tx pgx.Tx,
	schema_id uuid.UUID,
	cardinality DataSchemaCardinality,
	schema []DataSchemaField,
) error {
	table_cols := data_storage_table_columns_from_schema(cardinality, schema)
	table_name := data_storage_table_name_from_schema_id(schema_id)
	create_table_query := fmt.Sprintf(
		`CREATE TABLE %s (
			_data UUID REFERENCES data_(_id) PRIMARY KEY,
			%s
		)`,
		table_name,
		strings.Join(table_cols, ", "),
	)

	_, err := tx.Exec(s.ctx, create_table_query)
	if err != nil {
		s.logger.With(
			"error", err,
			"schema", schema_id,
		).Error("could not create data table for schema")
		return err
	}

	if cardinality == DataSchemaCardinalityMultiple && len(schema) > 1 {
		table_constraint_query :=
			data_schema_table_equal_column_length_constraint_query(
				table_name,
				schema,
			)

		_, err = tx.Exec(s.ctx, table_constraint_query)
		if err != nil {
			s.logger.With(
				"error", err,
				"schema", schema_id,
				"query", table_constraint_query, // remove
			).Error("could not create data table constraint for schema")
			return err
		}
	}

	return nil
}

type DataSchemaUpdate struct {
	Id          uuid.UUID
	Label       string
	Description *string
}

func (s *DataService) DataSchemaUpdate(update DataSchemaUpdate) error {
	query := "UPDATE data_schema_ SET label=$1, description=$2 WHERE _id=$3"
	_, err := s.db.Conn.Exec(s.ctx, query, update.Label, update.Description, update.Id)
	if err != nil {
		s.logger.With(
			"error", err,
			"update", update,
		).Error("could not update data schema")
	}

	return nil
}

type DataSchemaResources struct {
	DataSchema DataSchema
	Creator    User
}

func (s *DataService) DataSchemaGetResources(schema_id uuid.UUID) (DataSchemaResources, error) {
	schema_query := `
		SELECT _id, _creator, _cardinality, label, description
		FROM data_schema_ WHERE _id=$1 ORDER BY _id DESC`
	rows, err := s.db.Conn.Query(s.ctx, schema_query, schema_id)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schemas")
		return DataSchemaResources{}, err
	}

	schema_rx, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataSchemaRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schemas")
		return DataSchemaResources{}, err
	}

	fields_query :=
		`SELECT _id, _label, _dtype, description 
		FROM data_schema_field_ WHERE _id=$1`
	rows, err = s.db.Conn.Query(s.ctx, fields_query, schema_id)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schema fields")
		return DataSchemaResources{}, err
	}

	fields_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaFieldRecord])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schema fields")
		return DataSchemaResources{}, err
	}

	fields := make([]DataSchemaField, len(fields_rx))
	for idx, field := range fields_rx {
		fields[idx] = DataSchemaField{
			Label:       field.Label,
			DType:       field.DType,
			Description: field.Description,
		}
	}

	schema := DataSchema{
		Id:          schema_rx.Id,
		Creator:     schema_rx.Creator,
		Cardinality: schema_rx.Cardinality,
		Label:       schema_rx.Label,
		Description: schema_rx.Description,
		Schema:      fields,
	}

	creator_query := "SELECT _id, account_status, email, name FROM user_ WHERE _id=$1"
	rows, _ = s.db.Conn.Query(s.ctx, creator_query, schema.Creator)
	creator_rx, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[UserRecord])
	if err != nil {
		s.logger.With("error", err, "user", schema.Creator).Error("could not get data schema creator")
		return DataSchemaResources{}, err
	}

	permissions_query := "SELECT _permission FROM db_user_permission_ WHERE _user=$1"
	rows, _ = s.db.Conn.Query(s.ctx, permissions_query, schema.Creator)
	permissions, err := pgx.CollectRows(rows, pgx.RowTo[DbPermissionId])
	if err != nil {
		s.logger.With("error", err, "user", schema.Creator).Error("could not get data schema creator permissions")
		return DataSchemaResources{}, err
	}

	creator := User{
		Id:            creator_rx.Id,
		AccountStatus: creator_rx.AccountStatus,
		Email:         creator_rx.Email,
		Name:          creator_rx.Name,
		DbPermissions: permissions,
	}

	return DataSchemaResources{
		DataSchema: schema,
		Creator:    creator,
	}, nil
}

func (s *DataService) ParseDataFileToSchema(file_path string, schema_id uuid.UUID) ([]ColumnData, error) {
	file, err := os.Open(file_path)
	if err != nil {
		s.logger.With("error", err, "file", file_path).Error("could not open data file")
		return nil, err
	}

	fields, err := s.get_data_schema_fields(schema_id)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(file_path)
	data, err := parse_data_file_to_schema(ext, file, fields)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *DataService) get_data_schema_fields(schema_id uuid.UUID) ([]DataSchemaField, error) {
	query := `SELECT _label, _dtype, description FROM data_schema_field_ WHERE _id=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, schema_id)
	fields, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaField])
	if err != nil {
		s.logger.With("error", err, "schema", schema_id).Error("could not get data schema")
		return nil, err
	}

	return fields, nil
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
	Label  string
	DType  ValueType
	Values []any
}

func parse_data_file_to_schema(ext string, file *os.File, fields []DataSchemaField) ([]ColumnData, error) {
	switch ext {
	case ".csv", ".tsv":
		return parse_data_file_to_schema_csv(file, fields)
	default:
		return nil, &InvalidFileExtensionError{}
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
	DType ValueType
}

func (e *InvalidDataTypeError) Error() string {
	return fmt.Sprintf(
		"invalid data type, could not parse `%s` as `%s`",
		e.Value,
		e.DType,
	)
}

func parse_data_file_to_schema_csv(file *os.File, fields []DataSchemaField) ([]ColumnData, error) {
	reader := csv.NewReader(file)
	var record_idx uint = 0
	errs := []ParseCsvError{}
	data := make([]ColumnData, len(fields))

	for idx := range data {
		data[idx].Label = fields[idx].Label
		data[idx].DType = fields[idx].DType
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

		if len(record) != len(fields) {
			return []ColumnData{}, &IncompatibleDataSizeError{expected: len(fields), found: len(record)}
		}

		for idx, val_str := range record {
			switch fields[idx].DType {

			case ValueTypeBoolean:
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
				data[idx].Values = append(data[idx].Values, val)

			case ValueTypeFloat:
				val, err := strconv.ParseFloat(val_str, 64)
				if err != nil {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				data[idx].Values = append(data[idx].Values, val)

			case ValueTypeInt:
				val, err := strconv.ParseInt(val_str, 0, 32)
				if err != nil {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				data[idx].Values = append(data[idx].Values, val)
			case ValueTypeUint:
				val, err := strconv.ParseInt(val_str, 0, 32)
				if err != nil {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				if val < 0 {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: errors.New("value less than 0")})
					continue
				}
				data[idx].Values = append(data[idx].Values, uint(val))
			case ValueTypeString:
				data[idx].Values = append(data[idx].Values, val_str)
			case ValueTypeTimestamp:
				val, err := time.Parse(time.RFC3339, val_str)
				if err != nil {
					errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
					continue
				}
				data[idx].Values = append(data[idx].Values, val)
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
func (s *DataService) SampleDataStoredById(sample_data_ids []uuid.UUID) ([]StoredData, error) {
	if len(sample_data_ids) == 0 {
		return []StoredData{}, nil
	}

	type SampleDataSchema struct {
		SampleData uuid.UUID
		DataSchema uuid.UUID
	}
	rows, _ := s.db.Conn.Query(
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
		Id     uuid.UUID
		Schema []DataSchemaField
	}

	rows, err = s.db.Conn.Query(
		s.ctx,
		"SELECT _id, _schema FROM data_schema_ WHERE _id=ANY($1)",
		data_schema_ids,
	)

	// data_schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchemaRecord, error) {
	// 	var record DataSchemaRecord
	// 	err := row.Scan(&record.Id, &record.Schema)
	// 	return record, err
	// })
	// if err != nil {
	// 	s.logger.With(
	// 		"error", err,
	// 		"data schemas", data_schema_ids,
	// 	).Error("could not get data schemas")

	// 	return nil, err
	// }

	stored_data := make([]StoredData, len(sample_data_schemas))
	// for idx, sample_data_schema := range sample_data_schemas {
	// 	data_schema_idx := slices.IndexFunc(data_schemas, func(data_schema DataSchemaRecord) bool {
	// 		return data_schema.Id == sample_data_schema.DataSchema
	// 	})

	// 	data_schema := data_schemas[data_schema_idx]
	// 	var data any
	// 	switch data_schema.Storage {
	// 	case DataStorageExternal:
	// 		data, err = s.get_sample_data_stored_by_id_storage_external_data(
	// 			sample_data_schema.SampleData,
	// 			sample_data_schema.DataSchema,
	// 		)
	// 		if err != nil {
	// 			s.logger.With(
	// 				"error", err,
	// 				"sample data", sample_data_schema.SampleData,
	// 				"data schema", data_schema,
	// 			).Error("could not get stored sample data")
	// 			return nil, err
	// 		}
	// 	case DataStorageInternal:
	// 		data, err = s.get_sample_data_stored_by_id_storage_internal_data(
	// 			sample_data_schema.SampleData,
	// 			sample_data_schema.DataSchema,
	// 			data_schema.Schema,
	// 		)
	// 		if err != nil {
	// 			s.logger.With(
	// 				"error", err,
	// 				"sample data", sample_data_schema.SampleData,
	// 				"data schema", data_schema,
	// 			).Error("could not get stored sample data")
	// 			return nil, err
	// 		}
	// 	}
	// 	stored_data[idx] = StoredData{
	// 		SampleData: sample_data_schema.SampleData,
	// 		Storage:    data_schema.Storage,
	// 		Data:       data,
	// 	}
	// }

	return stored_data, nil
}

// // get_sample_data_stored_by_id_storage_external_data gets the file path of a sample data
// // with file storage
// func (s *DataService) get_sample_data_stored_by_id_storage_external_data(
// 	sample_data_id uuid.UUID,
// 	data_schema_id uuid.UUID,
// ) (SampleDataPayloadExternal, error) {
// 	var data SampleDataPayloadExternal
// 	data_query := fmt.Sprintf(
// 		"SELECT %s, %s FROM %s WHERE _sample_data=$1",
// 		DATA_STORAGE_TABLE_EXTERNAL_COL_PATH_LABEL,
// 		DATA_STORAGE_TABLE_EXTERNAL_COL_FILENAME_LABEL,
// 		data_storage_table_name_from_schema_id(data_schema_id),
// 	)
// 	err := s.db.Conn.QueryRow(
// 		s.ctx,
// 		data_query,
// 		sample_data_id,
// 	).Scan(&data.Path, &data.Filename)
// 	if err != nil {
// 		s.logger.With(
// 			"error", err,
// 			"query", data_query,
// 			"sample data", sample_data_id,
// 		).Error("could not get stored data")
// 		return SampleDataPayloadExternal{}, err
// 	}

// 	return data, nil
// }

// func (s *DataService) get_sample_data_stored_by_id_storage_internal_data(
// 	sample_data_id uuid.UUID,
// 	data_schema_id uuid.UUID,
// 	data_schema []DataSchemaField,
// ) ([]ColumnData, error) {
// 	column_labels := make([]string, len(data_schema))
// 	for idx, col := range data_schema {
// 		column_labels[idx] = col.Label
// 	}

// 	data_query := fmt.Sprintf(
// 		"SELECT %s FROM %s WHERE _sample_data=$1",
// 		strings.Join(column_labels, ", "),
// 		data_storage_table_name_from_schema_id(data_schema_id),
// 	)
// 	rows, err := s.db.Conn.Query(
// 		s.ctx,
// 		data_query,
// 		sample_data_id,
// 	)
// 	if err != nil {
// 		s.logger.With(
// 			"error", err,
// 			"query", data_query,
// 			"sample data", sample_data_id,
// 		).Error("could not get stored data")
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	if !rows.Next() {
// 		s.logger.With(
// 			"query", data_query,
// 			"sample data", sample_data_id,
// 		).Error("sample data not found")
// 		return nil, pgx.ErrNoRows
// 	}

// 	field_descs := rows.FieldDescriptions()
// 	if len(data_schema) != len(field_descs) {
// 		s.logger.With(
// 			"data schema", data_schema,
// 			"field descriptions", field_descs,
// 		).Error("stored data incompatible with data schema")
// 		panic("stored data incompatible with data schema")
// 	}

// 	col_data := make([]any, len(data_schema))
// 	scan_target := make([]any, len(data_schema))
// 	for idx := range col_data {
// 		scan_target[idx] = &col_data[idx]
// 	}
// 	err = rows.Scan(scan_target...)
// 	if err != nil {
// 		s.logger.With("error", err).Error("could not collect sample data")
// 		return nil, err
// 	}

// 	col_names := make([]string, len(field_descs))
// 	for idx, field := range field_descs {
// 		col_names[idx] = field.Name
// 	}

// 	data := make([]ColumnData, len(data_schema))
// 	for idx, col := range data_schema {
// 		col_data_idx := slices.Index(col_names, col.Label)
// 		if col_data_idx < 0 {
// 			s.logger.With(
// 				"data schema", data_schema,
// 				"field description", field_descs,
// 			).Error("field description incompatible with data schema")
// 			panic("field description incompatible with data schema")
// 		}
// 		data[idx].Label = col.Label
// 		data[idx].DType = col.DType
// 		data[idx].Values = col_data[col_data_idx].([]any)
// 	}

// 	return data, nil
// }

// SaveSampleDataSingle saves a single data to the user's disk.
// Returns the path the user selected.
func (s *DataService) SaveSampleDataSingle(sample_data_id uuid.UUID) (string, error) {
	stored_datas, err := s.SampleDataStoredById([]uuid.UUID{sample_data_id})
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
	case DataStorageExternal:
		data, err = s.data_storage_external_get_data(stored_data.Data.(string))
		if err != nil {
			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
			return "", err
		}
	case DataStorageInternal:
		data, err = s.StoredDataToCsv(stored_data.Data.([]ColumnData))
		if err != nil {
			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
			return "", err
		}
	default:
		panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored_data.Storage))
	}

	panic("todo")
	panic(data)
	// return s.fs_service.SaveFileSingle(data, "Save data", []FileFilter{})
}

func (s *DataService) StoredDataToCsv(data []ColumnData) ([]byte, error) {
	records := make([][]string, len(data[0].Values))
	for row_idx := range records {
		row := make([]string, len(data))
		for col_idx := range row {
			entry := data[col_idx].Values[row_idx]
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

	stored_data, err := s.SampleDataStoredById(sample_data)
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
	rows, _ := s.db.Conn.Query(s.ctx, data_sample_query, sample_data)
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
	rows, err = s.db.Conn.Query(s.ctx, sample_label_query, project, sample_ids)
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
	rows, _ = s.db.Conn.Query(s.ctx, schema_query, data_schema_ids)
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
		case DataStorageExternal:
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
		case DataStorageInternal:
			file_name = fmt.Sprintf(
				"%s-%s.%s.csv",
				data_info.Timestamp.Format(time.DateOnly),
				data_info.Timestamp.Format(time.TimeOnly),
				stored.SampleData.String(),
			)
			data, err = s.StoredDataToCsv(stored.Data.([]ColumnData))
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

	panic("todo")
	// save_filter := FileFilter{
	// 	DisplayName: "ZIP archive",
	// 	Pattern:     "*.zip",
	// }

	// return s.fs_service.SaveFileSingle(buf.Bytes(), "Save data", []FileFilter{save_filter})
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
	rows, _ := s.db.Conn.Query(s.ctx, sample_query, project, data_schema)
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
	stored_data, err := s.SampleDataStoredById(sample_data_ids)
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
		case DataStorageExternal:
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
		case DataStorageInternal:
			file_name = fmt.Sprintf(
				"%s-%s.%s.csv",
				sample_info.Timestamp.Format(time.DateOnly),
				sample_info.Timestamp.Format(time.TimeOnly),
				stored.SampleData.String(),
			)
			data, err = s.StoredDataToCsv(stored.Data.([]ColumnData))
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

	panic("todo")
	// save_filter := FileFilter{
	// 	DisplayName: "ZIP archive",
	// 	Pattern:     "*.zip",
	// }
	// return s.fs_service.SaveFileSingle(buf.Bytes(), "Save data", []FileFilter{save_filter})
}

func (s *DataService) save_data_schema_sample_data_file_path(
	hierarchy []SaveDataHierarchy,
	file_name_base string,
	sample_label string,
) (string, error) {
	hierarchy_components := map[SaveDataHierarchy]string{
		SaveDataHierarchySample: sample_label,
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

	sample_label, file_name_sample := hierarchy_components[SaveDataHierarchySample]
	if file_name_sample {
		file_path.WriteString(sample_label)
		file_path.WriteString(".")
	}
	schema_label, file_name_schema := hierarchy_components[SaveDataHierarchyDataSchema]
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
	rows, _ := s.db.Conn.Query(s.ctx, sample_query, project)
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
	rows, _ = s.db.Conn.Query(s.ctx, data_query, sample_ids)
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
	rows, _ = s.db.Conn.Query(s.ctx, schema_query, data_schema_ids)
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
	stored_data, err := s.SampleDataStoredById(sample_data_ids)
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
		case DataStorageExternal:
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
		case DataStorageInternal:
			file_name = fmt.Sprintf(
				"%s-%s.%s.csv",
				data_info.Timestamp.Format(time.DateOnly),
				data_info.Timestamp.Format(time.TimeOnly),
				stored.SampleData.String(),
			)
			data, err = s.StoredDataToCsv(stored.Data.([]ColumnData))
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

	panic("todo")
	// save_filter := FileFilter{
	// 	DisplayName: "ZIP archive",
	// 	Pattern:     "*.zip",
	// }
	// return s.fs_service.SaveFileSingle(buf.Bytes(), "Save data", []FileFilter{save_filter})
}

func (s *DataService) save_data_file_path(
	hierarchy []SaveDataHierarchy,
	file_name_base string,
	sample_label string,
	data_schema_label string,
) (string, error) {
	hierarchy_components := map[SaveDataHierarchy]string{
		SaveDataHierarchyDataSchema: data_schema_label,
		SaveDataHierarchySample:     sample_label,
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

	sample_label, file_name_sample := hierarchy_components[SaveDataHierarchySample]
	if file_name_sample {
		file_path.WriteString(sample_label)
		file_path.WriteString(".")
	}
	schema_label, file_name_schema := hierarchy_components[SaveDataHierarchyDataSchema]
	if file_name_schema {
		file_path.WriteString(schema_label)
		file_path.WriteString(".")
	}
	file_path.WriteString(file_name_base)

	return file_path.String(), nil
}

func (s *DataService) RawDataById(id uuid.UUID) (DataRecord, error) {
	query := `SELECT _id, _sample, _creator, _path, _type, _filename, label, timestamp, visibility 
		FROM raw_data_ WHERE _id=$1`
	rows, err := s.db.Conn.Query(s.ctx, query, id)
	if err != nil {
		s.logger.With(
			"error", err,
			"id", id,
		).Error("could not get sample data")
		return DataRecord{}, err
	}
	raw_data, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataRecord])
	if err != nil {
		s.logger.With(
			"error", err,
			"id", id,
		).Error("could not get sample data")
		return DataRecord{}, err
	}

	return raw_data, nil
}

type SampleDataUserPermission string

const (
	SampleDataUserPermissionOwner            SampleDataUserPermission = "owner"
	SampleDataUserPermissionRead             SampleDataUserPermission = "read"
	SampleDataUserPermissionCreateNote       SampleDataUserPermission = "create_note"
	SampleDataUserPermissionModifyProperties SampleDataUserPermission = "modify_properties"
)

type SampleDataUserPermissions struct {
	SampleData  uuid.UUID
	Permissions []SampleDataUserPermission
}

func (s *DataService) GetSampleDataUserPermission(sample_data []uuid.UUID, user uuid.UUID) ([]SampleDataUserPermissions, error) {
	type permissionRecord struct {
		SampleData uuid.UUID
		Permission SampleDataUserPermission
	}

	query :=
		`SELECT _sample_data, _permission FROM sample_data_user_permission_ 
		WHERE _sample_data=ANY($1) AND _user=$2`
	rows, _ := s.db.Conn.Query(s.ctx, query, sample_data, user)
	records, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (permissionRecord, error) {
		var permission permissionRecord
		err := row.Scan(&permission.SampleData, &permission.Permission)
		return permission, err
	})
	if err != nil {
		s.logger.With(
			"error", err,
			"sample data", sample_data,
			"user", user,
		).Error("could not get sample data user permissions")
		return nil, err
	}

	permissions := make([]SampleDataUserPermissions, len(sample_data))
	for idx, sample_data_id := range sample_data {
		permissions[idx].SampleData = sample_data_id
	}

	for _, record := range records {
		permissions_idx := slices.IndexFunc(permissions, func(entry SampleDataUserPermissions) bool {
			return entry.SampleData == record.SampleData
		})
		if permissions_idx < 0 {
			s.logger.With(
				"sample data", record.SampleData,
				"records", permissions,
			).Error("could not find record for sample data")
			panic("could not find record for sample data")
		}

		permissions[permissions_idx].Permissions = append(permissions[permissions_idx].Permissions, record.Permission)
	}

	return permissions, nil
}

func (s *DataService) ProjectRawDataAll(project uuid.UUID) ([]DataRecord, error) {
	query := `SELECT s._id, s._sample, s._schema, s._creator, s.timestamp, s.visibility, s.label 
		FROM sample_data_ AS s JOIN project_sample_membership_ as p ON s._sample=p._sample
		WHERE p._project=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, project)
	sample_data, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataRecord])
	return sample_data, err
}

type DataTypeTransformRecord struct {
	Id          uuid.UUID `db:"_id"`
	Source      uuid.UUID `db:"_source"`
	Destination uuid.UUID `db:"_destination"`
	Script      string    `db:"_script"`
	Creator     User      `db:"_creator"`
	Label       string    `db:"label"`
	Description string    `db:"description"`
}

func (s *DataService) DataTypeTransformsGetAll() ([]DataTypeTransformRecord, error) {
	query :=
		`SELECT _id, _source, _destination, _script, _creator, label, description
		FROM data_type_transform_`
	rows, _ := s.db.Conn.Query(s.ctx, query)
	transforms, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeTransformRecord])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get data type transforms")
		return nil, err
	}

	return transforms, nil
}

type DataTypeTransformCreate struct {
	Source      uuid.UUID
	Destination uuid.UUID
	Script      *multipart.FileHeader
	Label       string
	Description string
}

func (s *DataService) DataTypeTransformCreate(user uuid.UUID, transform DataTypeTransformCreate) (uuid.UUID, error) {
	transform_path, err := s.app_service.AppDataDir(AppDataDirTransform)
	if err != nil {
		s.logger.With(
			"error", err,
			"key", AppDataKeyDataPath,
		).Error("could not get app data path")
		return uuid.Nil, err
	}

	transform_id, err := uuid.NewV7()
	if err != nil {
		s.logger.With("error", err).Error("could not create transform id")
		return uuid.Nil, err
	}

	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return uuid.Nil, err
	}
	defer tx.Rollback(s.ctx)

	script_ext := filepath.Ext(transform.Script.Filename)
	transform_script_name := fmt.Sprintf("%s%s", transform_id, script_ext)
	script_path := filepath.Join(transform_path, transform_script_name)

	query :=
		`INSERT INTO data_type_transform_ (_id, _source, _destination, _script, _creator, label, description) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.Exec(
		s.ctx,
		query,
		transform_id,
		transform.Source,
		transform.Destination,
		script_path,
		user,
		transform.Label,
		transform.Description,
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user,
			"transform", transform,
		).Error("could not create transform")
		return uuid.Nil, err
	}

	trigger_fn_name := fmt.Sprintf("enqueue_transform_job_%s", uuid_to_sql_string(transform_id))
	trigger_fn_query := fmt.Sprintf(
		`CREATE FUNCTION %s()
		RETURNS TRIGGER
		LANGUAGE plpgsql
		AS $$
		BEGIN
			INSERT INTO _transform_queue_ (_transform, _payload)
			VALUES (
				'%s'::uuid,
				NEW._sample_data
			);

			RETURN NEW;
		END;
		$$;`,
		trigger_fn_name,
		transform_id,
	)
	_, err = tx.Exec(s.ctx, trigger_fn_query)
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not create transform trigger function")
		return uuid.Nil, err
	}

	trigger_query := fmt.Sprintf(
		`CREATE TRIGGER %s_after_insert
		AFTER INSERT ON %s
		FOR EACH ROW
		EXECUTE FUNCTION %s();`,
		trigger_fn_name,
		data_storage_table_name_from_schema_id(transform.Source),
		trigger_fn_name,
	)
	_, err = tx.Exec(s.ctx, trigger_query)
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not create transform trigger")
		return uuid.Nil, err
	}

	err = os.MkdirAll(transform_path, os.ModePerm)
	if err != nil {
		s.logger.With(
			"error", err,
			"path", transform_path,
		).Error("could not create transform directory")
		return uuid.Nil, err
	}

	dst, err := os.Create(script_path)
	if err != nil {
		s.logger.With(
			"error", err,
			"path", script_path,
		).Error("could not create transform script file")
		return uuid.Nil, err
	}
	defer dst.Close()

	src, err := transform.Script.Open()
	if err != nil {
		s.logger.With(
			"error", err,
			"script", transform.Script.Filename,
		).Error("could not open transform script file")
		return uuid.Nil, err
	}
	defer src.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		s.logger.With(
			"error", err,
			"path", script_path,
		).Error("could not write to transform script file")
		return uuid.Nil, err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not commit transaction")
		return uuid.Nil, err
	}
	return transform_id, nil
}
