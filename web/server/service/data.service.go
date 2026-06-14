package service

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syredb/database"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

type DataTypeInternalStorageRx struct {
	DataType uuid.UUID `db:"_data_type"`
	Schema   uuid.UUID `db:"_schema"`
}

type DataTypeExternalSourceRx struct {
	Id          uuid.UUID             `db:"_id"`
	DataType    uuid.UUID             `db:"_data_type"`
	Label       string                `db:"_label"`
	Required    bool                  `db:"_required"`
	Cardinality DataSourceCardinality `db:"_cardinality"`
	Description string                `db:"description"`
	ExtFilter   []string              `db:"ext_filter"`
}

type DataTypeRx struct {
	Id          uuid.UUID   `db:"_id"`
	Creator     uuid.UUID   `db:"_creator"`
	Storage     DataStorage `db:"_storage"`
	Label       string      `db:"label"`
	Description string      `db:"description"`
	Active      bool        `db:"active"`
}

type DataType interface {
	DataStorage() DataStorage
}

type DataTypeInternal struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Storage     DataStorage
	Label       string
	Description string
	Active      bool
	Schema      uuid.UUID
}

func (t DataTypeInternal) DataStorage() DataStorage {
	return DataStorageInternal
}

type DataTypeExternal struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Storage     DataStorage
	Label       string
	Description string
	Active      bool
	Sources     []DataTypeExternalSourceRx
}

func (t DataTypeExternal) DataStorage() DataStorage {
	return DataStorageExternal
}

type DataCreator interface {
	Type() DataCreatorType
}

type DataCreatorUser struct {
	Id     uuid.UUID
	Origin uuid.UUID
}

func (t DataCreatorUser) Type() DataCreatorType {
	return DataCreatorTypeUser
}

type DataCreatorTransform struct {
	Id uuid.UUID
}

func (t DataCreatorTransform) Type() DataCreatorType {
	return DataCreatorTypeTransform
}

func (s *DataService) DataTypesAll() ([]DataType, error) {
	data_type_query :=
		`SELECT _id, _creator, _storage, label, description, active
		FROM data_type_ ORDER BY label`
	rows, _ := s.db.Conn.Query(s.ctx, data_type_query)
	data_type_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeRx])
	if err != nil {
		s.logger.With("error", err).Error("could not get data types")
		return nil, err
	}

	data_type_internal := make([]DataTypeInternal, 0, len(data_type_rxs))
	data_type_external := make([]DataTypeExternal, 0, len(data_type_rxs))
	for _, rx := range data_type_rxs {
		switch rx.Storage {
		case DataStorageInternal:
			data := DataTypeInternal{
				Id:          rx.Id,
				Creator:     rx.Creator,
				Storage:     rx.Storage,
				Label:       rx.Label,
				Description: rx.Description,
				Active:      rx.Active,
			}
			data_type_internal = append(data_type_internal, data)
		case DataStorageExternal:
			data := DataTypeExternal{
				Id:          rx.Id,
				Creator:     rx.Creator,
				Storage:     rx.Storage,
				Label:       rx.Label,
				Description: rx.Description,
				Active:      rx.Active,
			}
			data_type_external = append(data_type_external, data)
		}
	}

	internal_ids := make([]uuid.UUID, len(data_type_internal))
	for idx, rx := range data_type_internal {
		internal_ids[idx] = rx.Id
	}
	internal_query :=
		`SELECT _data_type, _schema FROM data_type_schema_
		WHERE _data_type=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, internal_query, internal_ids)
	internal_schemas, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeInternalStorageRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data types", internal_ids,
		).Error("could not get data type internal storage")
		return nil, err
	}
	for _, rx := range internal_schemas {
		idx := slices.IndexFunc(data_type_internal, func(data DataTypeInternal) bool {
			return data.Id == rx.DataType
		})
		if idx < 0 {
			panic("invalid data type internal storage")
		}

		data_type_internal[idx].Schema = rx.Schema
	}

	external_ids := make([]uuid.UUID, len(data_type_external))
	for idx, rx := range data_type_external {
		external_ids[idx] = rx.Id
	}
	external_query :=
		`SELECT _id, _data_type, _label, _required, _cardinality, description, ext_filter
		FROM data_type_source_ WHERE _data_type=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, external_query, external_ids)
	external_sources, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeExternalSourceRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data types", external_ids,
		).Error("could not get data type sources")
		return nil, err
	}
	for _, source := range external_sources {
		idx := slices.IndexFunc(data_type_external, func(data DataTypeExternal) bool {
			return data.Id == source.DataType
		})
		if idx < 0 {
			panic("invalid data type source")
		}

		data_type_external[idx].Sources = append(data_type_external[idx].Sources, source)
	}

	data_types := make([]DataType, 0, len(data_type_rxs))
	for _, data := range data_type_internal {
		data_types = append(data_types, data)
	}
	for _, data := range data_type_external {
		data_types = append(data_types, data)
	}

	return data_types, nil
}

func (s *DataService) DataTypesById(ids []uuid.UUID) ([]DataType, error) {
	data_type_query :=
		`SELECT _id, _creator, _storage, label, description, active
		FROM data_type_ WHERE _id=ANY($1)`
	rows, _ := s.db.Conn.Query(s.ctx, data_type_query, ids)
	data_type_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeRx])
	if err != nil {
		s.logger.With("error", err).Error("could not get data types")
		return nil, err
	}

	data_type_internal := make([]DataTypeInternal, 0, len(data_type_rxs))
	data_type_external := make([]DataTypeExternal, 0, len(data_type_rxs))
	for _, rx := range data_type_rxs {
		switch rx.Storage {
		case DataStorageInternal:
			data := DataTypeInternal{
				Id:          rx.Id,
				Creator:     rx.Creator,
				Storage:     rx.Storage,
				Label:       rx.Label,
				Description: rx.Description,
				Active:      rx.Active,
			}
			data_type_internal = append(data_type_internal, data)
		case DataStorageExternal:
			data := DataTypeExternal{
				Id:          rx.Id,
				Creator:     rx.Creator,
				Storage:     rx.Storage,
				Label:       rx.Label,
				Description: rx.Description,
				Active:      rx.Active,
			}
			data_type_external = append(data_type_external, data)
		}
	}

	internal_ids := make([]uuid.UUID, len(data_type_internal))
	for idx, rx := range data_type_internal {
		internal_ids[idx] = rx.Id
	}
	internal_query :=
		`SELECT _data_type, _schema FROM data_type_schema_
		WHERE _data_type=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, internal_query, internal_ids)
	internal_schemas, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeInternalStorageRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data types", internal_ids,
		).Error("could not get data type internal storage")
		return nil, err
	}
	for _, rx := range internal_schemas {
		idx := slices.IndexFunc(data_type_internal, func(data DataTypeInternal) bool {
			return data.Id == rx.DataType
		})
		if idx < 0 {
			panic("invalid data type internal storage")
		}

		data_type_internal[idx].Schema = rx.Schema
	}

	external_ids := make([]uuid.UUID, len(data_type_external))
	for idx, rx := range data_type_external {
		external_ids[idx] = rx.Id
	}
	external_query :=
		`SELECT _id, _data_type, _label, _required, _cardinality, description, ext_filter
		FROM data_type_source_ WHERE _data_type=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, external_query, external_ids)
	external_sources, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeExternalSourceRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data types", external_ids,
		).Error("could not get data type sources")
		return nil, err
	}
	for _, source := range external_sources {
		idx := slices.IndexFunc(data_type_external, func(data DataTypeExternal) bool {
			return data.Id == source.DataType
		})
		if idx < 0 {
			panic("invalid data type source")
		}

		data_type_external[idx].Sources = append(data_type_external[idx].Sources, source)
	}

	data_types := make([]DataType, 0, len(data_type_rxs))
	for _, data := range data_type_internal {
		data_types = append(data_types, data)
	}
	for _, data := range data_type_external {
		data_types = append(data_types, data)
	}

	return data_types, nil
}

func (s *DataService) DataTypeById(id uuid.UUID) (DataType, error) {
	data_type_query :=
		`SELECT _id, _creator, _storage, label, description, active
		FROM data_type_ WHERE _id=$1`
	rows, _ := s.db.Conn.Query(s.ctx, data_type_query, id)
	data_type, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataTypeRx])
	if err != nil {
		s.logger.With("error", err).Error("could not get data types")
		return nil, err
	}

	return s.dataTypeRxInfo(data_type)
}

func (s *DataService) DataTypeByLabel(label string) (DataType, error) {
	data_type_query :=
		`SELECT _id, _creator, _storage, label, description, active
		FROM data_type_ WHERE label=$1`
	rows, _ := s.db.Conn.Query(s.ctx, data_type_query, label)
	data_type, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataTypeRx])
	if err != nil {
		s.logger.With("error", err).Error("could not get data types")
		return nil, err
	}

	return s.dataTypeRxInfo(data_type)
}

func (s *DataService) dataTypeRxInfo(data_type DataTypeRx) (DataType, error) {
	switch data_type.Storage {
	case DataStorageInternal:
		data_type_internal := DataTypeInternal{
			Id:          data_type.Id,
			Creator:     data_type.Creator,
			Storage:     DataStorageInternal,
			Label:       data_type.Label,
			Description: data_type.Description,
			Active:      data_type.Active,
		}
		internal_query :=
			`SELECT _schema FROM data_type_schema_
			WHERE _data_type=$1`
		err := s.db.Conn.QueryRow(s.ctx, internal_query, data_type.Id).Scan(&data_type_internal.Schema)
		if err != nil {
			s.logger.With(
				"error", err,
				"data type", data_type.Id,
			).Error("could not get data type internal storage")
			return nil, err
		}

		return data_type_internal, nil

	case DataStorageExternal:
		data_type_external := DataTypeExternal{
			Id:          data_type.Id,
			Creator:     data_type.Creator,
			Storage:     DataStorageInternal,
			Label:       data_type.Label,
			Description: data_type.Description,
			Active:      data_type.Active,
		}
		external_query :=
			`SELECT _id, _data_type, _label, _required, _cardinality, description, ext_filter
			FROM data_type_source_ WHERE _data_type=$1`
		rows, _ := s.db.Conn.Query(s.ctx, external_query, data_type.Id)
		external_sources, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeExternalSourceRx])
		if err != nil {
			s.logger.With(
				"error", err,
				"data type", data_type.Id,
			).Error("could not get data type sources")
			return nil, err
		}

		data_type_external.Sources = external_sources

		return data_type_external, nil

	default:
		panic("invalid data_type.Storage")
	}

}

type ExternalSourceCreate struct {
	Cardinality     DataSourceCardinality
	Required        bool
	ExtensionFilter []string
	Label           string
	Description     *string
}

func (s *DataService) DataTypeSourcesCreate(tx pgx.Tx, data_type uuid.UUID, sources []ExternalSourceCreate) ([]uuid.UUID, error) {
	if len(sources) == 0 {
		return []uuid.UUID{}, nil
	}

	var query strings.Builder
	query.WriteString(
		`INSERT INTO data_type_source_ 
		(_data_type, _cardinality, _required, extension_filter, label, description)
		VALUES`,
	)

	const fieldsPerRecord = 5
	const recordOffset = 1
	args := make([]any, len(sources)*fieldsPerRecord+recordOffset)
	args[0] = data_type
	for idx, source := range sources {
		cardinality_idx := idx*fieldsPerRecord + recordOffset
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

func (s *DataService) dataTypeCreate(
	tx pgx.Tx,
	creator uuid.UUID,
	storage DataStorage,
	label string,
	description *string,
) (uuid.UUID, error) {
	var type_id uuid.UUID
	type_query :=
		`INSERT INTO data_type_ (_creator, _storage, label, description)
		VALUES ($1, $2, $3, $4) RETURNING _id`
	err := tx.QueryRow(
		s.ctx,
		type_query,
		creator,
		storage,
		label,
		description,
	).Scan(&type_id)
	if err != nil {
		s.logger.With(
			"error", err,
			"creator", creator,
			"storage", storage,
			"label", label,
			"description", description,
		).Error("could not create data type")
		return uuid.Nil, err
	}

	return type_id, nil
}

func (s *DataService) DataTypeCreateInternal(
	creator uuid.UUID,
	label string,
	description *string,
	data_schema uuid.UUID,
) error {
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return err
	}
	defer tx.Rollback(s.ctx)

	type_id, err := s.dataTypeCreate(
		tx,
		creator,
		DataStorageInternal,
		label,
		description,
	)
	if err != nil {
		return err
	}

	storage_query := "INSERT INTO data_type_schema_ (_data_type, _schema) VALUES ($1, $2)"
	_, err = tx.Exec(s.ctx, storage_query, type_id, data_schema)
	if err != nil {
		s.logger.With(
			"error", err,
			"data schema", data_schema,
		).Error("could not create data type storage")
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not commit data type create transaction")
		return err
	}

	return nil
}

func (s *DataService) DataTypeCreateExternal(
	creator uuid.UUID,
	label string,
	description *string,
	sources []ExternalSourceCreate,
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

type DataSchemaRx struct {
	Id          uuid.UUID             `db:"_id"`
	Creator     uuid.UUID             `db:"_creator"`
	Cardinality DataSchemaCardinality `db:"_cardinality"`
	Label       string                `db:"label"`
	Description string                `db:"description"`
}

type DataSchemaFieldRx struct {
	Id          uuid.UUID `db:"_id"`
	Label       string    `db:"_label"`
	DType       ValueType `db:"_dtype"`
	Required    bool      `db:"_required"`
	Nullable    bool      `db:"_nullable"`
	Index       uint      `db:"index"`
	Description string    `db:"description"`
}

type DataSchemaField struct {
	Label       string    `db:"_label"`
	DType       ValueType `db:"_dtype"`
	Required    bool      `db:"_required"`
	Nullable    bool      `db:"_nullable"`
	Index       uint      `db:"index"`
	Description string    `db:"description"`
}

type DataSchema struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Cardinality DataSchemaCardinality
	Label       string
	Description string
	Fields      []DataSchemaField
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

	schemas_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaRx])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schemas")
		return nil, err
	}

	fields_query :=
		`SELECT _id, _label, _dtype, _required, _nullable, index, description 
		FROM data_schema_field_`
	rows, err = s.db.Conn.Query(s.ctx, fields_query)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schema fields")
		return nil, err
	}

	fields_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaFieldRx])
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
					Index:       field.Index,
					Description: field.Description,
				})
			}
		}
		slices.SortFunc(fields, func(a DataSchemaField, b DataSchemaField) int {
			return int(a.Index) - int(b.Index)
		})

		schemas[idx] = DataSchema{
			Id:          schema_rx.Id,
			Creator:     schema_rx.Creator,
			Cardinality: schema_rx.Cardinality,
			Label:       schema_rx.Label,
			Description: schema_rx.Description,
			Fields:      fields,
		}
	}

	return schemas, nil
}

func (s *DataService) DataSchemasById(schema_ids []uuid.UUID) ([]DataSchema, error) {
	if len(schema_ids) == 0 {
		return nil, nil
	}

	schemas_query :=
		`SELECT _id, _creator, _cardinality, label, description
		FROM data_schema_ WHERE _id=ANY($1) ORDER BY _id DESC`
	rows, err := s.db.Conn.Query(s.ctx, schemas_query, schema_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"data schemas", schema_ids,
		).Error("could not get data schemas")
		return nil, err
	}

	schemas_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaRx])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schemas")
		return nil, err
	}

	fields_query :=
		`SELECT _id, _label, _dtype, _required, _nullable, index, description 
		FROM data_schema_field_ WHERE _id=ANY($1)`
	rows, err = s.db.Conn.Query(s.ctx, fields_query, schema_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"data schemas", schema_ids,
		).Error("could not get data schema fields")
		return nil, err
	}

	fields_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaFieldRx])
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
					Index:       field.Index,
					Description: field.Description,
				})
			}
		}

		slices.SortFunc(fields, func(a DataSchemaField, b DataSchemaField) int {
			return int(a.Index) - int(b.Index)
		})
		schemas[idx] = DataSchema{
			Id:          schema_rx.Id,
			Creator:     schema_rx.Creator,
			Cardinality: schema_rx.Cardinality,
			Label:       schema_rx.Label,
			Description: schema_rx.Description,
			Fields:      fields,
		}
	}

	return schemas, nil
}

func (s *DataService) DataSchemaById(schema_id uuid.UUID) (DataSchema, error) {
	schemas_query :=
		`SELECT _id, _creator, _cardinality, label, description
		FROM data_schema_ WHERE _id=$1`
	rows, _ := s.db.Conn.Query(s.ctx, schemas_query, schema_id)
	schema_rx, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataSchemaRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"schema", schema_id,
		).Error("could not get data schema")
		return DataSchema{}, err
	}

	fields_query :=
		`SELECT _id, _label, _dtype, _required, _nullable, index, description 
		FROM data_schema_field_ WHERE _id=$1`
	rows, err = s.db.Conn.Query(s.ctx, fields_query, schema_id)
	if err != nil {
		s.logger.With(
			"error", err,
			"schema", schema_id,
		).Error("could not get data schema fields")
		return DataSchema{}, err
	}

	fields_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaFieldRx])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schema fields")
		return DataSchema{}, err
	}

	var fields []DataSchemaField
	for _, field := range fields_rx {
		if field.Id == schema_rx.Id {
			fields = append(fields, DataSchemaField{
				Label:       field.Label,
				DType:       field.DType,
				Required:    field.Required,
				Nullable:    field.Nullable,
				Index:       field.Index,
				Description: field.Description,
			})
		}
	}

	schema := DataSchema{
		Id:          schema_rx.Id,
		Creator:     schema_rx.Creator,
		Cardinality: schema_rx.Cardinality,
		Label:       schema_rx.Label,
		Description: schema_rx.Description,
		Fields:      fields,
	}

	return schema, nil
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
	const pattern = `^[\w_]+$`
	match, err := regexp.MatchString(pattern, label)
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
	has_permission, err := s.user_service.UserHasPermission(user_id, DbPermissionDataSchemaCreate)
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

	err = s.dataSchemaCreateSchema(tx, schema_id, data_schema.Schema)
	if err != nil {
		return err
	}

	err = s.dataSchemaStorageTableCreate(
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

func (s *DataService) dataSchemaCreateSchema(
	tx pgx.Tx,
	schema_id uuid.UUID,
	schema []DataSchemaField,
) error {
	var query strings.Builder
	query.WriteString(
		`INSERT INTO data_schema_field_ 
		(_id, _label, _dtype, index, description) VALUES `,
	)

	const numFields = 3
	const argsOffset = 1
	args := make([]any, len(schema)*numFields+argsOffset)
	args[0] = schema_id
	for idx, field := range schema {
		if idx > 0 {
			query.WriteString(", ")
		}

		idx_label := idx*numFields + argsOffset
		idx_dtype := idx_label + 1
		idx_index := idx_dtype + 1
		idx_description := idx_index + 1
		fmt.Fprintf(
			&query,
			"($1, $%d, $%d, $%d, $%d)",
			idx_label+1,
			idx_dtype+1,
			idx_index+1,
			idx_description+1,
		)

		description := new(string)
		if len(field.Description) > 0 {
			description = &field.Description
		}

		args[idx_label] = field.Label
		args[idx_dtype] = field.DType
		args[idx_index] = idx
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

func dataStorageTableColumnsFromSchema(
	schema []DataSchemaField,
	cardinality DataSchemaCardinality,
) []string {
	table_cols := make([]string, len(schema))
	for idx, col := range schema {
		var dtype string
		switch col.DType {
		case ValueTypeBoolean:
			dtype = "BOOLEAN"
		case ValueTypeFloat:
			dtype = "DOUBLE PRECISION"
		case ValueTypeInt:
			dtype = "BIGINT"
		case ValueTypeUint:
			dtype = "BIGINT"
		case ValueTypeString:
			dtype = "TEXT"
		case ValueTypeTimestamp:
			dtype = "TIMESTAMP WITH TIME ZONE"
		default:
			panic(fmt.Sprintf("unexpected value type %s for column %s", col.DType, col.Label))
		}

		if cardinality == DataSchemaCardinalityMultiple {
			dtype += "[]"
		}

		var nullable string
		if col.Nullable {
			nullable = ""
		} else {
			nullable = "NOT NULL"
		}

		table_cols[idx] = fmt.Sprintf(
			"%s %s %s",
			col.Label,
			dtype,
			nullable,
		)
	}

	return table_cols
}

func DataStorageTableNameFromSchemaId(schema_id uuid.UUID) string {
	const tableNamePrefix = "data_schema"
	schema_name := strings.ReplaceAll(schema_id.String(), "-", "_")
	return fmt.Sprintf(
		"%s_%s_",
		tableNamePrefix,
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

func (s *DataService) dataSchemaStorageTableCreate(
	tx pgx.Tx,
	schema_id uuid.UUID,
	cardinality DataSchemaCardinality,
	schema []DataSchemaField,
) error {
	// TODO: Ensure no schema field has label `_data`
	table_cols := dataStorageTableColumnsFromSchema(schema, cardinality)
	table_name := DataStorageTableNameFromSchemaId(schema_id)
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

	schema_rx, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataSchemaRx])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schemas")
		return DataSchemaResources{}, err
	}

	fields_query :=
		`SELECT _id, _label, _dtype, _required, _nullable, index, description 
		FROM data_schema_field_ WHERE _id=$1`
	rows, err = s.db.Conn.Query(s.ctx, fields_query, schema_id)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schema fields")
		return DataSchemaResources{}, err
	}

	fields_rx, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataSchemaFieldRx])
	if err != nil {
		s.logger.With("error", err).Error("could not collect data schema fields")
		return DataSchemaResources{}, err
	}

	fields := make([]DataSchemaField, len(fields_rx))
	for idx, field := range fields_rx {
		fields[idx] = DataSchemaField{
			Label:       field.Label,
			DType:       field.DType,
			Index:       field.Index,
			Description: field.Description,
		}
	}
	slices.SortFunc(fields, func(a DataSchemaField, b DataSchemaField) int {
		return int(a.Index) - int(b.Index)
	})

	schema := DataSchema{
		Id:          schema_rx.Id,
		Creator:     schema_rx.Creator,
		Cardinality: schema_rx.Cardinality,
		Label:       schema_rx.Label,
		Description: schema_rx.Description,
		Fields:      fields,
	}

	creator_query := "SELECT _id, account_status, email, name FROM user_ WHERE _id=$1"
	rows, _ = s.db.Conn.Query(s.ctx, creator_query, schema.Creator)
	creator_rx, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[UserRx])
	if err != nil {
		s.logger.With("error", err, "user", schema.Creator).Error("could not get data schema creator")
		return DataSchemaResources{}, err
	}

	permissions_query := "SELECT _permission FROM db_user_permission_ WHERE _user=$1"
	rows, _ = s.db.Conn.Query(s.ctx, permissions_query, schema.Creator)
	permissions, err := pgx.CollectRows(rows, pgx.RowTo[DbPermission])
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

func (s *DataService) ParseDataFileToSchema(file_path string, schema_id uuid.UUID) ([]SchemaFieldValues, error) {
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

// Values a single `DType` if Cardinality is `single`,
// a slice of `DType` if Cardinality is `multiple`.
type SchemaFieldValues struct {
	Label       string
	DType       ValueType
	Cardinality DataSchemaCardinality
	Values      any
}

func parse_data_file_to_schema(ext string, file *os.File, fields []DataSchemaField) ([]SchemaFieldValues, error) {
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

func parse_data_file_to_schema_csv(file *os.File, fields []DataSchemaField) ([]SchemaFieldValues, error) {
	panic("TODO: parse_data_file_to_schema_csv")

	// reader := csv.NewReader(file)
	// var record_idx uint = 0
	// errs := []ParseCsvError{}
	// data := make([]SchemaFieldValues, len(fields))

	// for idx := range data {
	// 	data[idx].Label = fields[idx].Label
	// 	data[idx].DType = fields[idx].DType
	// }

	// for {
	// 	record, err := reader.Read()
	// 	if err != nil {
	// 		if errors.Is(err, io.EOF) {
	// 			break
	// 		}

	// 		errs = append(errs, ParseCsvError{Record: record_idx, Column: 0, Err: err})
	// 		continue
	// 	}

	// 	if len(record) != len(fields) {
	// 		return []SchemaFieldValues{}, &IncompatibleDataSizeError{expected: len(fields), found: len(record)}
	// 	}

	// 	for idx, val_str := range record {
	// 		switch fields[idx].DType {

	// 		case ValueTypeBoolean:
	// 			var val bool
	// 			switch val_str {
	// 			case "true":
	// 				val = true
	// 			case "false":
	// 				val = false
	// 			default:
	// 				errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
	// 				continue
	// 			}
	// 			data[idx].Values = append(data[idx].Values, val)

	// 		case ValueTypeFloat:
	// 			val, err := strconv.ParseFloat(val_str, 64)
	// 			if err != nil {
	// 				errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
	// 				continue
	// 			}
	// 			data[idx].Values = append(data[idx].Values, val)

	// 		case ValueTypeInt:
	// 			val, err := strconv.ParseInt(val_str, 0, 32)
	// 			if err != nil {
	// 				errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
	// 				continue
	// 			}
	// 			data[idx].Values = append(data[idx].Values, val)
	// 		case ValueTypeUint:
	// 			val, err := strconv.ParseInt(val_str, 0, 32)
	// 			if err != nil {
	// 				errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
	// 				continue
	// 			}
	// 			if val < 0 {
	// 				errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: errors.New("value less than 0")})
	// 				continue
	// 			}
	// 			data[idx].Values = append(data[idx].Values, uint(val))
	// 		case ValueTypeString:
	// 			data[idx].Values = append(data[idx].Values, val_str)
	// 		case ValueTypeTimestamp:
	// 			val, err := time.Parse(time.RFC3339, val_str)
	// 			if err != nil {
	// 				errs = append(errs, ParseCsvError{Record: record_idx, Column: idx, Err: err})
	// 				continue
	// 			}
	// 			data[idx].Values = append(data[idx].Values, val)
	// 		default:
	// 			return []SchemaFieldValues{}, errors.New("unexpected app.DataType")
	// 		}
	// 	}
	// }

	// if len(errs) > 0 {
	// 	err_msgs := make([]string, len(errs))
	// 	for idx, err := range errs {
	// 		err_msgs[idx] = err.Error()
	// 	}
	// 	msg := fmt.Sprintf("invalid data file: [%s]", strings.Join(err_msgs, ", "))
	// 	return []SchemaFieldValues{}, errors.New(msg)
	// }

	// return data, nil
}

type IngestionScriptRx struct {
	Id          uuid.UUID `db:"_id"`
	Type        uuid.UUID `db:"_type"`
	Creator     uuid.UUID `db:"_creator"`
	Cmd         uuid.UUID `db:"cmd"`
	Label       string    `db:"label"`
	Description string    `db:"description"`
}

type IngestionScriptCmdRx struct {
	Id      uuid.UUID `db:"_id"`
	Creator uuid.UUID `db:"_creator"`
	Path    string    `db:"_path"`
	Cmd     string    `db:"_cmd"`
	Args    []string  `db:"_args"`
}

type IngestionScriptSourceRx struct {
	Id          uuid.UUID             `db:"_id"`
	Script      uuid.UUID             `db:"_script"`
	Label       string                `db:"_label"`
	Required    bool                  `db:"_required"`
	Cardinality DataSourceCardinality `db:"_cardinality"`
	Description string                `db:"description"`
	ExtFilter   []string              `db:"ext_filter"`
}

type IngestionScript struct {
	Id          uuid.UUID
	Type        uuid.UUID
	Creator     uuid.UUID
	Label       string
	Description string
	Cmd         IngestionScriptCmdRx
	Sources     []IngestionScriptSourceRx
}

func (s *DataService) IngestionScriptsGetAll() ([]IngestionScript, error) {
	script_query :=
		`SELECT _id, _type, _creator, cmd, label, description
		FROM ingestion_script_`
	rows, _ := s.db.Conn.Query(s.ctx, script_query)
	script_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[IngestionScriptRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get ingestion scripts")
		return nil, err
	}
	cmd_ids := make([]uuid.UUID, len(script_rxs))
	for idx, script := range script_rxs {
		cmd_ids[idx] = script.Cmd
	}

	cmd_query :=
		`SELECT _id, _creator, _path, _cmd, _args 
		FROM ingestion_script_cmd_ WHERE _id=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, cmd_query, cmd_ids)
	cmd_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[IngestionScriptCmdRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get ingestion script commands")
		return nil, err
	}

	scripts := make([]IngestionScript, len(script_rxs))
	for idx := range scripts {
		scripts[idx].Id = script_rxs[idx].Id
		scripts[idx].Type = script_rxs[idx].Type
		scripts[idx].Creator = script_rxs[idx].Creator
		scripts[idx].Label = script_rxs[idx].Label
		scripts[idx].Description = script_rxs[idx].Description
		cmd_idx := slices.IndexFunc(cmd_rxs, func(cmd IngestionScriptCmdRx) bool {
			return cmd.Id == script_rxs[idx].Cmd
		})
		if cmd_idx < 0 {
			panic("invalid ingestion script command")
		}

		scripts[idx].Cmd = cmd_rxs[cmd_idx]
	}

	return scripts, nil
}

func (s *DataService) IngestionScriptsGetForDataType(data_type uuid.UUID) ([]IngestionScript, error) {
	script_query :=
		`SELECT _id, _type, _creator, cmd, label, description
		FROM ingestion_script_ WHERE _type=$1`
	rows, _ := s.db.Conn.Query(s.ctx, script_query, data_type)
	script_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[IngestionScriptRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get ingestion scripts")
		return nil, err
	}
	cmd_ids := make([]uuid.UUID, len(script_rxs))
	for idx, script := range script_rxs {
		cmd_ids[idx] = script.Cmd
	}

	cmd_query :=
		`SELECT _id, _creator, _path, _cmd, _args 
		FROM ingestion_script_cmd_ WHERE _id=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, cmd_query, cmd_ids)
	cmd_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[IngestionScriptCmdRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get ingestion script commands")
		return nil, err
	}

	script_ids := make([]uuid.UUID, len(script_rxs))
	for idx, script := range script_rxs {
		script_ids[idx] = script.Id
	}
	source_query :=
		`SELECT _id, _script, _label, _required, _cardinality, description, ext_filter
		FROM ingestion_script_source_ WHERE _script=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, source_query, script_ids)
	sources, err := pgx.CollectRows(rows, pgx.RowToStructByName[IngestionScriptSourceRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get ingestion script source")
		return nil, err
	}

	scripts := make([]IngestionScript, len(script_rxs))
	for idx := range scripts {
		script_id := script_rxs[idx].Id
		scripts[idx].Id = script_id
		scripts[idx].Type = script_rxs[idx].Type
		scripts[idx].Creator = script_rxs[idx].Creator
		scripts[idx].Label = script_rxs[idx].Label
		scripts[idx].Description = script_rxs[idx].Description
		cmd_idx := slices.IndexFunc(cmd_rxs, func(cmd IngestionScriptCmdRx) bool {
			return cmd.Id == script_rxs[idx].Cmd
		})
		if cmd_idx < 0 {
			panic("invalid ingestion script command")
		}
		scripts[idx].Cmd = cmd_rxs[cmd_idx]

		for _, source := range sources {
			if source.Script == script_id {
				scripts[idx].Sources = append(scripts[idx].Sources, source)
			}
		}
	}

	return scripts, nil
}

func (s *DataService) IngestionScriptGet(id uuid.UUID) (IngestionScript, error) {
	script_query :=
		`SELECT _id, _type, _creator, cmd, label, description
		FROM ingestion_script_ WHERE _id=$1`
	rows, _ := s.db.Conn.Query(s.ctx, script_query, id)
	script_rx, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[IngestionScriptRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get ingestion scripts")
		return IngestionScript{}, err
	}

	cmd_query :=
		`SELECT _id, _creator, _path, _cmd, _args 
		FROM ingestion_script_cmd_ WHERE _id=$1`
	rows, _ = s.db.Conn.Query(s.ctx, cmd_query, script_rx.Cmd)
	cmd_rx, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[IngestionScriptCmdRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get ingestion script command")
		return IngestionScript{}, err
	}

	source_query :=
		`SELECT _id, _script, _label, _required, _cardinality, description, ext_filter
		FROM ingestion_script_source_ WHERE _script=$1`
	rows, _ = s.db.Conn.Query(s.ctx, source_query, id)
	sources, err := pgx.CollectRows(rows, pgx.RowToStructByName[IngestionScriptSourceRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get ingestion script source")
		return IngestionScript{}, err
	}

	script := IngestionScript{
		Id:          id,
		Type:        script_rx.Type,
		Creator:     script_rx.Creator,
		Label:       script_rx.Label,
		Description: script_rx.Description,
		Cmd:         cmd_rx,
		Sources:     sources,
	}

	return script, nil
}

type IngestionScriptCreate struct {
	Type        uuid.UUID
	Creator     uuid.UUID
	Label       string
	Description string
	Path        string
	Cmd         string
	Args        []string
	Sources     []ExternalSourceCreate
}

func (s *DataService) IngestionScriptCreate(script IngestionScriptCreate, file *multipart.FileHeader) error {
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return err
	}
	defer tx.Rollback(s.ctx)

	var cmd_id uuid.UUID
	cmd_query :=
		`INSERT INTO ingestion_script_cmd_ (_creator, _path, _cmd, _args) 
		VALUES ($1, $2, $3, $4) RETURNING _id`
	err = tx.QueryRow(s.ctx, cmd_query, script.Creator, script.Path, script.Cmd, script.Args).Scan(&cmd_id)
	if err != nil {
		s.logger.With("script", script).Error("could not create ingestion script command")
		return err
	}

	var script_id uuid.UUID
	script_query :=
		`INSERT INTO ingestion_script_ (_type, _creator, cmd, label, description)
		VALUES ($1, $2, $3, $4, $5) RETURNING _id`
	err = tx.QueryRow(
		s.ctx,
		script_query,
		script.Type,
		script.Creator,
		cmd_id,
		script.Label,
		script.Description,
	).Scan(&script_id)
	if err != nil {
		s.logger.With("script", script).Error("could not create ingestion script")
		return err
	}

	const sourceQueryFields = 5
	const sourceQueryOffset = 1
	source_query_args := make([]any, sourceQueryOffset+sourceQueryFields*len(script.Sources))
	source_query_args[0] = script_id
	var source_query strings.Builder
	source_query.WriteString(
		`INSERT INTO ingestion_script_source_ 
		(_script, _label, _required, _cardinality, description, ext_filter)
		VALUES `,
	)
	for idx, source := range script.Sources {
		if idx > 0 {
			source_query.WriteString(", ")
		}

		label_idx := idx*sourceQueryFields + sourceQueryOffset
		required_idx := label_idx + 1
		cardinality_idx := required_idx + 1
		description_idx := cardinality_idx + 1
		ext_filter_idx := description_idx + 1
		source_query_args[label_idx] = source.Label
		source_query_args[required_idx] = source.Required
		source_query_args[cardinality_idx] = source.Cardinality
		source_query_args[description_idx] = source.Description
		source_query_args[ext_filter_idx] = source.ExtensionFilter
		fmt.Fprintf(
			&source_query,
			"($1, $%d, $%d, $%d, $%d, $%d)",
			label_idx+1,
			required_idx+1,
			cardinality_idx+1,
			description_idx+1,
			ext_filter_idx+1,
		)
	}

	_, err = tx.Exec(s.ctx, source_query.String(), source_query_args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", source_query.String(),
			"args", source_query_args,
		).Error("could not create ingestion script sources")
		return err
	}

	err = SaveFormFile(file, script.Path)
	if err != nil {
		s.logger.With("error", err).Error("could not save ingestion script file")
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not commit transaction")
		return err
	}

	return nil
}

const DATA_ORIGIN_WEB_CLIENT_LABEL = "__web_client__"

type DataOriginRx struct {
	Id          uuid.UUID `db:"_id"`
	Label       string    `db:"label"`
	Description string    `db:"description"`
	Active      bool      `db:"active"`
}

func (s *DataService) DataOriginByLabel(label string) (DataOriginRx, error) {
	query :=
		`SELECT _id, label, description, active FROM data_origin_
		WHERE label=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, label)
	origin, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataOriginRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"label", label,
		).Error("could not get data origin")
		return DataOriginRx{}, err
	}
	return origin, nil
}

func (s *DataService) DataOriginsByIds(ids []uuid.UUID) ([]DataOriginRx, error) {
	query :=
		`SELECT _id, label, description, active FROM data_origin_
		WHERE _id=ANY($1)`
	rows, _ := s.db.Conn.Query(s.ctx, query, ids)
	origins, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataOriginRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"ids", ids,
		).Error("could not get data origins")
		return nil, err
	}
	return origins, nil
}

// DataValues represents the actual data stored.
// Values is []SchemaFieldValues if Storage is `internal`.
// Values is a []DataSource if Storage is `external`.
type DataValues struct {
	Data    uuid.UUID
	Storage DataStorage
	Values  any
}

// DataValuesById gets the values associated with a data.
func (s *DataService) DataValuesById(data uuid.UUID) (DataValues, error) {

	type dataStorageInfo struct {
		Data     uuid.UUID   `db:"data"`
		DataType uuid.UUID   `db:"type"`
		Storage  DataStorage `db:"storage"`
	}
	storage_query :=
		`SELECT d._id as data, t._id as type, t._storage as storage
		FROM data_ as d JOIN data_type_ as t ON d._type=t._id
		WHERE d._id=$1`
	rows, _ := s.db.Conn.Query(s.ctx, storage_query, data)
	info, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dataStorageInfo])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data storage info")

		return DataValues{}, err
	}

	var vals any
	switch info.Storage {
	case DataStorageExternal:
		vals, err = s.dataValuesByIdExternalSource(info.Data)
		if err != nil {
			s.logger.With(
				"error", err,
				"data", info.Data,
				"storage", DataStorageExternal,
			).Error("could not get data values")
			return DataValues{}, err
		}
	case DataStorageInternal:
		vals, err = s.dataValuesByIdInternal(info.Data)
		if err != nil {
			s.logger.With(
				"error", err,
				"data", info.Data,
				"storage", DataStorageInternal,
			).Error("could not get data values")
			return DataValues{}, err
		}
	}

	values := DataValues{
		Data:    info.Data,
		Storage: info.Storage,
		Values:  vals,
	}

	return values, nil
}

// DataValuesByIds gets the values associated with data.
func (s *DataService) DataValuesByIds(data_ids []uuid.UUID) ([]DataValues, error) {
	if len(data_ids) == 0 {
		return []DataValues{}, nil
	}

	type dataStorageInfo struct {
		Data     uuid.UUID   `db:"data"`
		DataType uuid.UUID   `db:"type"`
		Storage  DataStorage `db:"storage"`
	}
	storage_query :=
		`SELECT d._id as data, t._id as type, t._storage as storage
		FROM data_ as d JOIN data_type_ as t ON d._type=t._id
		WHERE d._id=ANY($1)`
	rows, _ := s.db.Conn.Query(s.ctx, storage_query, data_ids)
	data_info, err := pgx.CollectRows(rows, pgx.RowToStructByName[dataStorageInfo])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data_ids,
		).Error("could not get data storage info")

		return nil, err
	}

	data_values := make([]DataValues, len(data_info))
	for idx, info := range data_info {
		var values any
		switch info.Storage {
		case DataStorageExternal:
			values, err = s.dataValuesByIdExternalSource(info.Data)
			if err != nil {
				s.logger.With(
					"error", err,
					"data", info.Data,
					"storage", DataStorageExternal,
				).Error("could not get data values")
				return nil, err
			}
		case DataStorageInternal:
			values, err = s.dataValuesByIdInternal(info.Data)
			if err != nil {
				s.logger.With(
					"error", err,
					"data", info.Data,
					"storage", DataStorageInternal,
				).Error("could not get data values")
				return nil, err
			}
		}

		data_values[idx] = DataValues{
			Data:    info.Data,
			Storage: info.Storage,
			Values:  values,
		}
	}

	return data_values, nil
}

// DataSource is an externally stored data source.
// `Sources` is a single path if `Cardinality` is `single`.
// `Sources` is an array of paths if `Cardinality` is `multiple`.
type DataSource struct {
	Label       string
	Cardinality DataSourceCardinality
	Source      any
}

// get_sample_data_stored_by_id_storage_external_data gets the file path of a sample data
// with file storage
func (s *DataService) dataValuesByIdExternalSource(
	data uuid.UUID,
) ([]DataSource, error) {
	type sourceInfo struct {
		Cardinality DataSourceCardinality `db:"cardinality"`
		Label       string                `db:"label"`
		Path        string                `db:"path"`
	}

	query :=
		`SELECT s._path as path, t._cardinality as cardinality, t._label as label
		FROM data_source s JOIN data_type_external_source_ t
		WHERE s._data=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, data)
	info, err := pgx.CollectRows(rows, pgx.RowToStructByName[sourceInfo])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data sources")
		return nil, err
	}

	single_sources := make(map[string]string)
	multisources := make(map[string][]string)
	for _, i := range info {
		if i.Cardinality == DataSourceCardinalitySingle {
			single_sources[i.Label] = i.Path
		} else {
			srcs := multisources[i.Label]
			multisources[i.Label] = append(srcs, i.Path)
		}
	}

	sources := make([]DataSource, 0, len(single_sources)+len(multisources))
	for key, path := range single_sources {
		sources = append(sources, DataSource{
			Label:       key,
			Cardinality: DataSourceCardinalitySingle,
			Source:      path,
		})
	}
	for key, paths := range multisources {
		sources = append(sources, DataSource{
			Label:       key,
			Cardinality: DataSourceCardinalityMultiple,
			Source:      paths,
		})
	}

	return sources, nil
}

func (s *DataService) dataValuesByIdInternal(
	data uuid.UUID,
) ([]SchemaFieldValues, error) {
	type schemaFieldInfo struct {
		Label string    `db:"_label"`
		DType ValueType `db:"_dtype"`
		Index uint      `db:"index"`
	}

	var schema_id uuid.UUID
	schema_id_query :=
		`SELECT s._schema FROM 
		data_ d JOIN data_type_ t ON d._type=t._id
		JOIN data_type_schema_ s ON s._data_type=t._id
		WHERE d._id=$1`
	err := s.db.Conn.QueryRow(s.ctx, schema_id_query, data).Scan(&schema_id)
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data schema")
		return nil, err
	}

	var schema_cardinality DataSchemaCardinality
	schema_cardinality_query :=
		`SELECT _cardinality FROM data_schema_ 
		WHERE _id=$1`
	err = s.db.Conn.QueryRow(s.ctx, schema_cardinality_query, schema_id).Scan(&schema_cardinality)
	if err != nil {
		s.logger.With(
			"error", err,
			"schema", schema_id,
		).Error("could not get data schema cardinality")
		return nil, err
	}

	schema_fields_query :=
		`SELECT _label, _dtype, index FROM
		data_schema_field_ WHERE _id=$1`
	rows, _ := s.db.Conn.Query(s.ctx, schema_fields_query, schema_id)
	fields, err := pgx.CollectRows(rows, pgx.RowToStructByName[schemaFieldInfo])
	field_labels := make([]string, len(fields))
	for idx, field := range fields {
		field_labels[idx] = field.Label
	}
	data_query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE _data=$1",
		strings.Join(field_labels, ", "),
		DataStorageTableNameFromSchemaId(schema_id),
	)
	rows, err = s.db.Conn.Query(s.ctx, data_query, data)
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data values")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		s.logger.With(
			"query", data_query,
			"data", data,
		).Error("data values not found")
		return nil, pgx.ErrNoRows
	}
	rx_fields := rows.FieldDescriptions()
	rx_values, err := rows.Values()
	if err != nil {
		s.logger.With(
			"error", err,
			"schema", schema_id,
		).Error("could not get data schema values")
		return nil, err
	}

	values := make([]SchemaFieldValues, len(fields))
	for _, field := range fields {
		rx_idx := slices.IndexFunc(rx_fields, func(desc pgconn.FieldDescription) bool {
			return desc.Name == field.Label
		})
		if rx_idx < 0 {
			s.logger.With(
				"fields", rx_fields,
				"field", field,
			).Error("invalid data schema field")
			panic("invalid data schema field")
		}

		fidx := field.Index
		values[fidx].Label = field.Label
		values[fidx].DType = field.DType
		values[fidx].Cardinality = schema_cardinality
		values[fidx].Values = rx_values[rx_idx]
	}

	return values, nil
}

func (s *DataService) StoredDataToCsv(fields []SchemaFieldValues) (string, error) {
	var records strings.Builder
	writer := csv.NewWriter(&records)

	header := make([]string, len(fields))
	for idx, field := range fields {
		header[idx] = field.Label
	}
	err := writer.Write(header)
	if err != nil {
		return "", err
	}
	writer.Flush()
	if writer.Error() != nil {
		return "", err
	}

	switch fields[0].Cardinality {
	case DataSchemaCardinalityMultiple:
		value_strs := make([][]string, len(fields))
		for idx, field := range fields {
			value_strs[idx] = valuesToStrings(field.DType, field.Values.([]any))
		}
		height := len(value_strs[0])
		record := make([]string, len(fields))
		for ridx := range height {
			for fidx, values := range value_strs {
				record[fidx] = values[ridx]
			}
			err = writer.Write(record)
			if err != nil {
				return "", err
			}
		}
		writer.Flush()
		if writer.Error() != nil {
			return "", err
		}
	case DataSchemaCardinalitySingle:
		record := make([]string, len(fields))
		for idx, field := range fields {
			record[idx] = valueToString(field.DType, field.Values)
		}
		err = writer.WriteAll([][]string{record})
		if err != nil {
			return "", err
		}
	default:
		panic("unexpected DataSchemaCardinality")
	}

	return records.String(), nil
}

func valueToString(dtype ValueType, value any) string {
	fmtstr := dataTypeFormatString(dtype)
	return fmt.Sprintf(fmtstr, castValueToGoType(dtype, value))
}

func valuesToStrings(dtype ValueType, values []any) []string {
	strs := make([]string, len(values))
	fmtstr := dataTypeFormatString(dtype)
	for idx, val := range values {
		strs[idx] = fmt.Sprintf(fmtstr, castValueToGoType(dtype, val))
	}

	return strs
}

func castValueToGoType(dtype ValueType, value any) any {
	switch dtype {
	case ValueTypeBoolean:
		return value.(bool)
	case ValueTypeFloat:
		return value.(float64)
	case ValueTypeInt:
		return int64(value.(float64))
	case ValueTypeUint:
		return int64(value.(float64))
	case ValueTypeString:
		return value.(string)
	case ValueTypeTimestamp:
		return value.(time.Time)
	default:
		panic(fmt.Sprintf("unexpected ValueType: %#v", dtype))
	}
}

func dataTypeFormatString(dtype ValueType) string {
	switch dtype {
	case ValueTypeBoolean:
		return "%t"
	case ValueTypeFloat:
		return "%f"
	case ValueTypeInt:
		return "%d"
	case ValueTypeUint:
		return "%d"
	case ValueTypeString:
		return "\"%s\""
	case ValueTypeTimestamp:
		return "%v"
	default:
		panic(fmt.Sprintf("unexpected ValueType: %#v", dtype))
	}
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
	panic("TODO: SaveSampleDataMultiple")
	// if len(sample_data) == 0 {
	// 	return "", nil
	// }

	// stored_data, err := s.SampleDataStoredById(sample_data)
	// if err != nil {
	// 	return "", err
	// }
	// if len(stored_data) != len(sample_data) {
	// 	s.logger.With("sample data", sample_data, "stored data", stored_data).Error("incompatible number of data found")
	// 	panic("found invalid number of data")
	// }

	// type SampleDataInfo struct {
	// 	SampleData uuid.UUID
	// 	Sample     uuid.UUID
	// 	DataSchema uuid.UUID
	// 	Timestamp  time.Time
	// }
	// data_sample_query := "SELECT _id, _sample, _schema, timestamp FROM sample_data_ WHERE _id=ANY($1)"
	// rows, _ := s.db.Conn.Query(s.ctx, data_sample_query, sample_data)
	// sample_data_info, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleDataInfo, error) {
	// 	var record SampleDataInfo
	// 	err := row.Scan(&record.SampleData, &record.Sample, &record.DataSchema, &record.Timestamp)
	// 	return record, err
	// })
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not retrive data samples")
	// 	return "", err
	// }

	// type SampleInfo struct {
	// 	Id    uuid.UUID
	// 	Label string
	// }
	// var sample_ids []uuid.UUID
	// for _, data_sample := range sample_data_info {
	// 	if !slices.Contains(sample_ids, data_sample.Sample) {
	// 		sample_ids = append(sample_ids, data_sample.Sample)
	// 	}
	// }
	// sample_label_query := "SELECT _sample, label FROM project_sample_membership_ where _project=$1 AND _sample=ANY($2)"
	// rows, err = s.db.Conn.Query(s.ctx, sample_label_query, project, sample_ids)
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not get sample labels")
	// 	return "", err
	// }
	// sample_info, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleInfo, error) {
	// 	var info SampleInfo
	// 	err = rows.Scan(&info.Id, &info.Label)
	// 	return info, err
	// })
	// if err != nil {
	// 	s.logger.With("error", err, "samples", sample_ids).Error("could not get sample info")
	// }

	// type DataSchemaRx struct {
	// 	Id    uuid.UUID
	// 	Label string
	// }
	// data_schema_ids := []uuid.UUID{}
	// for _, data := range sample_data_info {
	// 	if !slices.Contains(data_schema_ids, data.DataSchema) {
	// 		data_schema_ids = append(data_schema_ids, data.DataSchema)
	// 	}
	// }
	// schema_query := "SELECT _id, label FROM data_schema_ WHERE _id=ANY($1)"
	// rows, _ = s.db.Conn.Query(s.ctx, schema_query, data_schema_ids)
	// data_schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchemaRx, error) {
	// 	var record DataSchemaRx
	// 	err := row.Scan(&record.Id, &record.Label)
	// 	return record, err
	// })
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not get project data schemas")
	// 	return "", err
	// }

	// buf := new(bytes.Buffer)
	// archive := zip.NewWriter(buf)
	// for _, stored := range stored_data {
	// 	data_sample_idx := slices.IndexFunc(sample_data_info, func(info SampleDataInfo) bool {
	// 		return info.SampleData == stored.SampleData
	// 	})
	// 	if data_sample_idx < 0 {
	// 		s.logger.With("sample data", stored.SampleData).Error("could not find sample data label record")
	// 		panic("could not find sample data label record")
	// 	}
	// 	data_info := sample_data_info[data_sample_idx]

	// 	sample_info_idx := slices.IndexFunc(sample_info, func(info SampleInfo) bool {
	// 		return info.Id == data_info.Sample
	// 	})
	// 	sample_info := sample_info[sample_info_idx]

	// 	data_schema_idx := slices.IndexFunc(data_schemas, func(record DataSchemaRx) bool {
	// 		return record.Id == data_info.DataSchema
	// 	})
	// 	data_schema := data_schemas[data_schema_idx]

	// 	var file_name string
	// 	var data []byte
	// 	switch stored.Storage {
	// 	case DataStorageExternal:
	// 		file_path := stored.Data.(string)
	// 		base := filepath.Base(file_path)
	// 		ext := filepath.Ext(base)
	// 		fname := base[:-(len(ext) + 1)]
	// 		file_name = fmt.Sprintf(
	// 			"%s.%s.%s",
	// 			fname,
	// 			stored.SampleData.String(),
	// 			ext,
	// 		)
	// 		data, err = s.data_storage_external_get_data(file_path)
	// 		if err != nil {
	// 			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
	// 			return "", err
	// 		}
	// 	case DataStorageInternal:
	// 		file_name = fmt.Sprintf(
	// 			"%s-%s.%s.csv",
	// 			data_info.Timestamp.Format(time.DateOnly),
	// 			data_info.Timestamp.Format(time.TimeOnly),
	// 			stored.SampleData.String(),
	// 		)
	// 		data, err = s.StoredDataToCsv(stored.Data.([]ColumnData))
	// 		if err != nil {
	// 			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
	// 			return "", err
	// 		}
	// 	default:
	// 		panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored.Storage))
	// 	}

	// 	file_path, err := s.save_data_file_path(data_hierarchy, file_name, sample_info.Label, data_schema.Label)
	// 	if err != nil {
	// 		return "", err
	// 	}

	// 	file, err := archive.Create(file_path)
	// 	if err != nil {
	// 		s.logger.With(
	// 			"error", err,
	// 			"sample data", stored.SampleData,
	// 		).Error("could not create archive file")
	// 		return "", err
	// 	}

	// 	_, err = file.Write(data)
	// 	if err != nil {
	// 		s.logger.With(
	// 			"error", err,
	// 			"stored data", stored,
	// 		).Error("could not write data to archive file")
	// 	}
	// }

	// err = archive.Close()
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not close archive")
	// 	return "", nil
	// }

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
	panic("TODO: SaveDataSchemaSampleDataAll")

	// rows, _ := s.db.Conn.Query(s.ctx, sample_query, project, data_schema)
	// samples, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleRx, error) {
	// 	var record SampleRx
	// 	err := row.Scan(&record.Sample, &record.SampleData, &record.Label, &record.Timestamp)
	// 	return record, err
	// })
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not get project data schema samples")
	// 	return "", err
	// }
	// if len(samples) == 0 {
	// 	return "", nil
	// }

	// sample_data_ids := make([]uuid.UUID, len(samples))
	// for idx, data := range samples {
	// 	sample_data_ids[idx] = data.SampleData
	// }
	// stored_data, err := s.DataValuesById(sample_data_ids)
	// if err != nil {
	// 	return "", err
	// }
	// if len(stored_data) != len(sample_data_ids) {
	// 	s.logger.With(
	// 		"sample data", sample_data_ids,
	// 		"stored data", stored_data,
	// 	).Error("incompatible number of data found")
	// 	panic("found invalid number of data")
	// }

	// buf := new(bytes.Buffer)
	// archive := zip.NewWriter(buf)
	// for _, stored := range stored_data {
	// 	sample_idx := slices.IndexFunc(samples, func(record SampleRx) bool {
	// 		return record.SampleData == stored.Data
	// 	})
	// 	if sample_idx < 0 {
	// 		s.logger.With(
	// 			"sample data", stored.Data,
	// 		).Error("could not find sample data label record")
	// 		panic("could not find sample data label record")
	// 	}
	// 	sample_info := samples[sample_idx]

	// 	var file_name string
	// 	var data []byte
	// 	switch stored.Storage {
	// 	case DataStorageExternal:
	// 		file_path := stored.Data(string)
	// 		base := filepath.Base(file_path)
	// 		ext := filepath.Ext(base)
	// 		fname := base[:-(len(ext) + 1)]
	// 		file_name = fmt.Sprintf(
	// 			"%s.%s.%s",
	// 			fname,
	// 			stored.Data.String(),
	// 			ext,
	// 		)
	// 		data, err = s.data_storage_external_get_data(file_path)
	// 		if err != nil {
	// 			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
	// 			return "", err
	// 		}
	// 	case DataStorageInternal:
	// 		file_name = fmt.Sprintf(
	// 			"%s-%s.%s.csv",
	// 			sample_info.Timestamp.Format(time.DateOnly),
	// 			sample_info.Timestamp.Format(time.TimeOnly),
	// 			stored.Data.String(),
	// 		)
	// 		data, err = s.StoredDataToCsv(stored.Data.([]ColumnData))
	// 		if err != nil {
	// 			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
	// 			return "", err
	// 		}
	// 	default:
	// 		panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored.Storage))
	// 	}

	// 	file_path, err := s.save_data_schema_sample_data_file_path(data_hierarchy, file_name, sample_info.Label)
	// 	if err != nil {
	// 		return "", err
	// 	}

	// 	file, err := archive.Create(file_path)
	// 	if err != nil {
	// 		s.logger.With(
	// 			"error", err,
	// 			"data", stored.Data,
	// 		).Error("could not create archive file")
	// 		return "", err
	// 	}

	// 	_, err = file.Write(data)
	// 	if err != nil {
	// 		s.logger.With(
	// 			"error", err,
	// 			"stored data", stored,
	// 		).Error("could not write data to archive file")
	// 	}
	// }

	// err = archive.Close()
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not close archive")
	// 	return "", nil
	// }

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
	panic("TODO: SaveProjectDataAll")
	// type ProjectSampleRx struct {
	// 	Sample uuid.UUID
	// 	Label  string
	// }
	// sample_query := "SELECT _sample, label FROM project_sample_membership_ WHERE _project=$1"
	// rows, _ := s.db.Conn.Query(s.ctx, sample_query, project)
	// project_samples, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ProjectSampleRx, error) {
	// 	var record ProjectSampleRx
	// 	err := row.Scan(&record.Sample, &record.Label)
	// 	return record, err
	// })
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not get project samples")
	// 	return "", err
	// }
	// if len(project_samples) == 0 {
	// 	return "", nil
	// }

	// type SampleDataRx struct {
	// 	Id         uuid.UUID
	// 	Sample     uuid.UUID
	// 	DataSchema uuid.UUID
	// 	Timestamp  time.Time
	// }
	// sample_ids := make([]uuid.UUID, len(project_samples))
	// for idx, sample := range project_samples {
	// 	sample_ids[idx] = sample.Sample
	// }
	// data_query := "SELECT _id, _sample, _schema, timestamp FROM sample_data_ WHERE _sample=ANY($1)"
	// rows, _ = s.db.Conn.Query(s.ctx, data_query, sample_ids)
	// sample_data, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleDataRx, error) {
	// 	var record SampleDataRx
	// 	err := row.Scan(&record.Id, &record.Sample, &record.DataSchema, &record.Timestamp)
	// 	return record, err
	// })
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not get project sample data")
	// 	return "", err
	// }

	// type DataSchemaRx struct {
	// 	Id    uuid.UUID
	// 	Label string
	// }
	// data_schema_ids := []uuid.UUID{}
	// for _, data := range sample_data {
	// 	if !slices.Contains(data_schema_ids, data.DataSchema) {
	// 		data_schema_ids = append(data_schema_ids, data.DataSchema)
	// 	}
	// }
	// schema_query := "SELECT _id, label FROM data_schema_ WHERE _id=ANY($1)"
	// rows, _ = s.db.Conn.Query(s.ctx, schema_query, data_schema_ids)
	// data_schemas, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DataSchemaRx, error) {
	// 	var record DataSchemaRx
	// 	err := row.Scan(&record.Id, &record.Label)
	// 	return record, err
	// })
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not get project data schemas")
	// 	return "", err
	// }

	// sample_data_ids := make([]uuid.UUID, len(sample_data))
	// for idx, data := range sample_data {
	// 	sample_data_ids[idx] = data.Id
	// }
	// stored_data, err := s.DataValuesById(sample_data_ids)
	// if err != nil {
	// 	return "", err
	// }
	// if len(stored_data) != len(sample_data) {
	// 	s.logger.With("sample data", sample_data, "stored data", stored_data).Error("incompatible number of data found")
	// 	panic("found invalid number of data")
	// }

	// buf := new(bytes.Buffer)
	// archive := zip.NewWriter(buf)
	// for _, stored := range stored_data {
	// 	data_sample_idx := slices.IndexFunc(sample_data, func(record SampleDataRx) bool {
	// 		return record.Id == stored.Data
	// 	})
	// 	if data_sample_idx < 0 {
	// 		s.logger.With(
	// 			"sample data", stored.Data,
	// 		).Error("could not find sample data label record")
	// 		panic("could not find sample data label record")
	// 	}
	// 	data_info := sample_data[data_sample_idx]

	// 	project_sample_idx := slices.IndexFunc(project_samples, func(record ProjectSampleRx) bool {
	// 		return record.Sample == data_info.Sample
	// 	})
	// 	project_sample := project_samples[project_sample_idx]

	// 	data_schema_idx := slices.IndexFunc(data_schemas, func(record DataSchemaRx) bool {
	// 		return record.Id == data_info.DataSchema
	// 	})
	// 	data_schema := data_schemas[data_schema_idx]

	// 	var file_name string
	// 	var data []byte
	// 	switch stored.Storage {
	// 	case DataStorageExternal:
	// 		file_path := stored.Data.(string)
	// 		base := filepath.Base(file_path)
	// 		ext := filepath.Ext(base)
	// 		fname := base[:-(len(ext) + 1)]
	// 		file_name = fmt.Sprintf(
	// 			"%s.%s.%s",
	// 			fname,
	// 			stored.Data.String(),
	// 			ext,
	// 		)
	// 		data, err = s.data_storage_external_get_data(file_path)
	// 		if err != nil {
	// 			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
	// 			return "", err
	// 		}
	// 	case DataStorageInternal:
	// 		file_name = fmt.Sprintf(
	// 			"%s-%s.%s.csv",
	// 			data_info.Timestamp.Format(time.DateOnly),
	// 			data_info.Timestamp.Format(time.TimeOnly),
	// 			stored.Data.String(),
	// 		)
	// 		data, err = s.StoredDataToCsv(stored.Data.([]ColumnData))
	// 		if err != nil {
	// 			s.logger.With("stored data", stored_data).Error("could not get stored sample data")
	// 			return "", err
	// 		}
	// 	default:
	// 		panic(fmt.Sprintf("unexpected app.DataStorage: %#v", stored.Storage))
	// 	}

	// 	file_path, err := s.save_data_file_path(hierarchy, file_name, project_sample.Label, data_schema.Label)
	// 	if err != nil {
	// 		return "", err
	// 	}

	// 	file, err := archive.Create(file_path)
	// 	if err != nil {
	// 		s.logger.With(
	// 			"error", err,
	// 			"data", stored.Data,
	// 		).Error("could not create archive file")
	// 		return "", err
	// 	}

	// 	_, err = file.Write(data)
	// 	if err != nil {
	// 		s.logger.With(
	// 			"error", err,
	// 			"stored data", stored,
	// 		).Error("could not write data to archive file")
	// 	}
	// }

	// err = archive.Close()
	// if err != nil {
	// 	s.logger.With("error", err).Error("could not close archive")
	// 	return "", nil
	// }

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

func (s *DataService) DataById(id uuid.UUID) (DataRx, error) {
	query :=
		`SELECT _id, _type, _creator_type, timestamp, visibility
		FROM data_ WHERE _id=$1`
	rows, err := s.db.Conn.Query(s.ctx, query, id)
	if err != nil {
		s.logger.With(
			"error", err,
			"id", id,
		).Error("could not get data")
		return DataRx{}, err
	}
	data, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[DataRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"id", id,
		).Error("could not get data")
		return DataRx{}, err
	}

	return data, nil
}

func (s *DataService) DataByIds(ids []uuid.UUID) ([]DataRx, error) {
	query :=
		`SELECT _id, _type, _creator_type, timestamp, visibility
		FROM data_ WHERE _id=ANY($1)`
	rows, err := s.db.Conn.Query(s.ctx, query, ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"ids", ids,
		).Error("could not get data")
		return nil, err
	}
	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"ids", ids,
		).Error("could not get data")
		return nil, err
	}

	return data, nil
}

type DataUserPermission string

const (
	DataUserPermissionOwner            DataUserPermission = "owner"
	DataUserPermissionRead             DataUserPermission = "read"
	DataUserPermissionReadValues       DataUserPermission = "read_values"
	DataUserPermissionNoteCreate       DataUserPermission = "note_create"
	DataUserPermissionPropertiesModify DataUserPermission = "properties_modify"
)

type DataUserPermissions struct {
	Data        uuid.UUID
	Permissions []DataUserPermission
}

func (s *DataService) DataUserPermission(user uuid.UUID, data uuid.UUID) ([]DataUserPermission, error) {
	type permissionRx struct {
		Data       uuid.UUID          `db:"_data"`
		Permission DataUserPermission `db:"_permission"`
	}

	query :=
		`SELECT _data, _permission FROM data_user_permission_ 
		WHERE _data=$1 AND _user=$2`
	rows, _ := s.db.Conn.Query(s.ctx, query, data, user)
	rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[permissionRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
			"user", user,
		).Error("could not get data user permissions")
		return nil, err
	}

	var permissions []DataUserPermission
	for _, rx := range rxs {
		permissions = append(permissions, rx.Permission)
	}

	return permissions, nil
}

func (s *DataService) DataUserPermissions(user uuid.UUID, data_ids []uuid.UUID) ([]DataUserPermissions, error) {
	type permissionRx struct {
		Data       uuid.UUID          `db:"_data"`
		Permission DataUserPermission `db:"_permission"`
	}

	query :=
		`SELECT _data, _permission FROM data_user_permission_ 
		WHERE _data=ANY($1) AND _user=$2`
	rows, _ := s.db.Conn.Query(s.ctx, query, data_ids, user)
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[permissionRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data_ids,
			"user", user,
		).Error("could not get data user permissions")
		return nil, err
	}

	permissions := make([]DataUserPermissions, len(data_ids))
	for idx, data_id := range data_ids {
		permissions[idx].Data = data_id
	}

	for _, record := range records {
		permissions_idx := slices.IndexFunc(permissions, func(entry DataUserPermissions) bool {
			return entry.Data == record.Data
		})
		if permissions_idx < 0 {
			s.logger.With(
				"data", record.Data,
				"records", permissions,
			).Error("could not find record for data")
			panic("could not find record for data")
		}

		permissions[permissions_idx].Permissions = append(permissions[permissions_idx].Permissions, record.Permission)
	}

	return permissions, nil
}

type ProjectDataWithMembership struct {
	Data              DataRx
	MembershipCreator uuid.UUID
	ProjectLabel      *string
}

func (s *DataService) ProjectDataAll(project uuid.UUID) ([]ProjectDataWithMembership, error) {
	type dataMembership struct {
		Data    uuid.UUID `db:"_data"`
		Creator uuid.UUID `db:"_creator"`
		Label   *string   `db:"label"`
	}

	query :=
		`SELECT _data, _creator, label FROM project_data_membership_
		WHERE _project=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, project)
	memberships, err := pgx.CollectRows(rows, pgx.RowToStructByName[dataMembership])
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
		).Error("could not get project data memberships")
		return nil, err
	}

	data_ids := make([]uuid.UUID, len(memberships))
	for idx, mem := range memberships {
		data_ids[idx] = mem.Data
	}
	data, err := s.DataByIds(data_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
		).Error("could not get project data")
		return nil, err
	}

	info := make([]ProjectDataWithMembership, len(memberships))
	for idx, mem := range memberships {
		data_idx := slices.IndexFunc(data, func(d DataRx) bool {
			return d.Id == mem.Data
		})
		if data_idx < 0 {
			panic("data not found")
		}

		info[idx].Data = data[data_idx]
		info[idx].MembershipCreator = mem.Creator
		info[idx].ProjectLabel = mem.Label
	}

	return info, nil
}

type DataTypeTransformRx struct {
	Id          uuid.UUID `db:"_id"`
	Creator     uuid.UUID `db:"_creator"`
	Source      uuid.UUID `db:"_source"`
	Destination uuid.UUID `db:"_destination"`
	Cmd         uuid.UUID `db:"cmd"`
	Label       string    `db:"label"`
	Description string    `db:"description"`
}

type DataTypeTransformCmdRx struct {
	Id      uuid.UUID `db:"_id"`
	Creator uuid.UUID `db:"_creator"`
	Path    string    `db:"_path"`
	Cmd     string    `db:"_cmd"`
	Args    []string  `db:"_args"`
}

type DataTypeTransform struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Source      uuid.UUID
	Destination uuid.UUID
	Label       string
	Description string
	Cmd         DataTypeTransformCmdRx
}

func (s *DataService) DataTypeTransformsGetAll() ([]DataTypeTransform, error) {
	transform_query :=
		`SELECT _id, _creator, _source, _destination, cmd, label, description
		FROM data_type_transform_`
	rows, _ := s.db.Conn.Query(s.ctx, transform_query)
	transform_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeTransformRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get data type transforms")
		return nil, err
	}

	cmd_ids := make([]uuid.UUID, len(transform_rxs))
	for idx, transform := range transform_rxs {
		cmd_ids[idx] = transform.Cmd
	}
	cmd_query :=
		`SELECT _id, _creator, _path, _cmd, _args FROM data_type_transform_cmd_
		WHERE _id=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, cmd_query, cmd_ids)
	cmd_rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataTypeTransformCmdRx])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get data type transform commands")
		return nil, err
	}

	transforms := make([]DataTypeTransform, len(transform_rxs))
	for idx := range transforms {
		transforms[idx].Id = transform_rxs[idx].Id
		transforms[idx].Creator = transform_rxs[idx].Creator
		transforms[idx].Source = transform_rxs[idx].Source
		transforms[idx].Destination = transform_rxs[idx].Destination
		transforms[idx].Label = transform_rxs[idx].Label
		transforms[idx].Description = transform_rxs[idx].Description

		cmd_idx := slices.IndexFunc(cmd_rxs, func(cmd DataTypeTransformCmdRx) bool {
			return cmd.Id == transform_rxs[idx].Cmd
		})
		if cmd_idx < 0 {
			panic("invalid data type transform command")
		}

		transforms[idx].Cmd = cmd_rxs[cmd_idx]
	}

	return transforms, nil
}

type DataTypeTransformCreate struct {
	Creator     uuid.UUID
	Source      uuid.UUID
	Destination uuid.UUID
	Label       string
	Description string
	Cmd         string
	Args        []string
	Script      *multipart.FileHeader
}

func (s *DataService) DataTypeTransformCreate(transform DataTypeTransformCreate) (uuid.UUID, error) {
	transform_path, err := s.app_service.AppDataDir(AppDataDirTransform)
	if err != nil {
		s.logger.With(
			"error", err,
			"key", AppDataKeyDataPath,
		).Error("could not get app data path")
		return uuid.Nil, err
	}

	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return uuid.Nil, err
	}
	defer tx.Rollback(s.ctx)

	s.logger.Debug("fielname", "filename", transform.Script.Filename)
	filename := fmt.Sprintf("%s.%s", rand.Text(), transform.Script.Filename)
	script_path := filepath.Join(transform_path, filename)

	var cmd_id uuid.UUID
	cmd_query :=
		`INSERT INTO data_type_transform_cmd_ (_creator, _path, _cmd, _args)
		VALUES ($1, $2, $3, $4) RETURNING _id`
	err = tx.QueryRow(
		s.ctx,
		cmd_query,
		transform.Creator,
		script_path,
		transform.Cmd,
		transform.Args,
	).Scan(&cmd_id)
	if err != nil {
		s.logger.With(
			"error", err,
			"transform", transform,
		).Error("could not create data type transform command")
		return uuid.Nil, err
	}

	var transform_id uuid.UUID
	transform_query :=
		`INSERT INTO data_type_transform_ 
		(_creator, _source, _destination, cmd, label, description) 
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING _id`
	err = tx.QueryRow(
		s.ctx,
		transform_query,
		transform.Creator,
		transform.Source,
		transform.Destination,
		cmd_id,
		transform.Label,
		transform.Description,
	).Scan(&transform_id)
	if err != nil {
		s.logger.With(
			"error", err,
			"transform", transform,
		).Error("could not create transform")
		return uuid.Nil, err
	}

	err = SaveFormFile(transform.Script, script_path)
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not create transform script")
		return uuid.Nil, err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not commit transaction")
		return uuid.Nil, err
	}
	return transform_id, nil
}

type DataCreatorType string

const (
	DataCreatorTypeUser      DataCreatorType = "user"
	DataCreatorTypeTransform DataCreatorType = "transform"
)

type Note struct {
	Timestamp  time.Time
	Visibility Visibility
	Content    string
}

type DataIngestionMethod string

const (
	DataIngestionManual DataIngestionMethod = "manual"
	DataIngestionScript DataIngestionMethod = "script"
)

// `Values` is only valid if `IngestionMethod` is `manual`.
// `IngestionScript` and `IngestionScriptSources` are only valid if `IngestionMethod` is `script`.
type DataCreate struct {
	Type                   uuid.UUID
	Creator                DataCreatorUser
	Timestamp              time.Time
	Visibility             Visibility
	Properties             []Property
	Notes                  []Note
	IngestionMethod        DataIngestionMethod
	Values                 map[string]any
	IngestionScript        uuid.UUID
	IngestionScriptSources map[uuid.UUID][]*multipart.FileHeader
}

func (s *DataService) DataCreate(
	data []DataCreate,
	owner uuid.UUID,
) ([]uuid.UUID, error) {
	if len(data) == 0 {
		return []uuid.UUID{}, nil
	}

	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return nil, err
	}
	defer tx.Rollback(s.ctx)

	data_ids, err := s.dataInsert(tx, data)
	if err != nil {
		return nil, err
	}

	err = s.dataCreatePermissions(tx, data_ids, owner)
	if err != nil {
		return nil, err
	}

	err = s.dataCreateCreatorUser(tx, data, data_ids)
	if err != nil {
		return nil, err
	}

	err = s.dataCreateProperties(tx, data, data_ids)
	if err != nil {
		return nil, err
	}

	err = s.dataCreateNotes(tx, data, data_ids)
	if err != nil {
		return nil, err
	}

	for idx, datum := range data {
		switch datum.IngestionMethod {
		case DataIngestionManual:
			var schema_id uuid.UUID
			schema_query := "SELECT _schema FROM data_type_schema_ WHERE _data_type=$1"
			err = s.db.Conn.QueryRow(s.ctx, schema_query, datum.Type).Scan(&schema_id)
			if err != nil {
				s.logger.With(
					"erorr", err,
					"data type", datum.Type,
				).Error("could not get data type schema")
				return nil, err
			}

			schema, err := s.DataSchemaById(schema_id)
			if err != nil {
				s.logger.With(
					"error", err,
					"schema", schema_id,
				).Error("could not get data schema")
				return nil, err
			}

			err = s.dataCreateValidateValuesAsSchema(schema, datum.Values)
			if err != nil {
				return nil, err
			}

			field_labels := make([]string, len(schema.Fields))
			for idx, field := range schema.Fields {
				field_labels[idx] = field.Label
			}
			var values_query strings.Builder
			fmt.Fprintf(
				&values_query,
				`INSERT INTO %s (_data, %s) VALUES ($1`,
				DataStorageTableNameFromSchemaId(schema_id),
				strings.Join(field_labels, ", "),
			)
			args := make([]any, len(schema.Fields)+1)
			args[0] = data_ids[idx]
			for idx, field := range schema.Fields {
				idx_arg := idx + 1
				fmt.Fprintf(&values_query, ", $%d", idx_arg+1)
				args[idx_arg] = datum.Values[field.Label]
			}
			values_query.WriteString(")")
			_, err = tx.Exec(s.ctx, values_query.String(), args...)
			if err != nil {
				s.logger.With(
					"error", err,
					"data", datum,
				).Error("could not store data values")
				return nil, err
			}

		case DataIngestionScript:
			// TODO: Validate sources are valid relative to ingestion script
			var save_err error
			data_id := data_ids[idx]
			filepaths, err := s.dataCreateDataIngestionScriptSources(tx, data_id, datum.IngestionScriptSources)
			for source, files := range datum.IngestionScriptSources {
				source_paths := filepaths[source]
				for idx, file := range files {
					filepath := source_paths[idx]
					err = SaveFormFile(file, filepath)
					if err != nil {
						s.logger.With(
							"error", err,
							"data", data_id,
							"source", source,
						).Error("could not save data ingestion source file")
						save_err = err
						break
					}
				}

				if save_err != nil {
					break
				}
			}

			if save_err != nil {
				for _, files := range filepaths {
					for _, filepath := range files {
						err = os.Remove(filepath)
						if !os.IsNotExist(err) {
							s.logger.With(
								"error", err,
								"file", filepath,
							).Error("could not remove file")
						}
					}
				}

				return nil, save_err
			}
		}
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not create data")
		return nil, err
	}

	return data_ids, nil
}

func (s *DataService) dataCreateValidateValuesAsSchema(schema DataSchema, values map[string]any) error {
	switch schema.Cardinality {
	case DataSchemaCardinalitySingle:
		for _, field := range schema.Fields {
			_, exists := values[field.Label]
			if !exists && field.Required {
				return fmt.Errorf("required field %s missing", field.Label)
			}
		}

		return nil
	case DataSchemaCardinalityMultiple:
		for _, field := range schema.Fields {
			f_values, exists := values[field.Label]
			if !exists {
				if field.Required {
					return fmt.Errorf("required field %s missing", field.Label)
				} else {
					continue
				}
			}

			f_values_arr := f_values.([]any)
			if !field.Nullable {
				if slices.Contains(f_values_arr, nil) {
					return fmt.Errorf("null value found in non-nullable field %s", field.Label)
				}
			}
		}

		return nil
	}

	panic("unreachable")
}

type DataPermissionKey string

const (
	DataPermissionKeyOwner            DataPermissionKey = "owner"
	DataPermissionKeyRead             DataPermissionKey = "read"
	DataPermissionKeyNoteCreate       DataPermissionKey = "note_create"
	DataPermissionKeyPropertiesModify DataPermissionKey = "properties_modify"
)

func (s *DataService) dataInsert(tx pgx.Tx, data []DataCreate) ([]uuid.UUID, error) {
	data_ids := make([]uuid.UUID, len(data))
	data_query :=
		`INSERT INTO data_ (_type, _creator_type, timestamp, visibility) 
		VALUES ($1, $2, $3, $4) RETURNING _id`
	for idx, datum := range data {
		err := tx.QueryRow(
			s.ctx,
			data_query,
			datum.Type,
			datum.Creator.Type(),
			datum.Timestamp,
			datum.Visibility,
		).Scan(&data_ids[idx])
		if err != nil {
			s.logger.With(
				"error", err,
				"data", datum,
			).Error("could not create data")
			return nil, err
		}
	}

	return data_ids, nil
}

func (s *DataService) dataCreatePermissions(
	tx pgx.Tx,
	data_ids []uuid.UUID,
	owner uuid.UUID,
) error {
	const argsOffset = 2

	args := make([]any, len(data_ids)+argsOffset)
	args[0] = owner
	args[1] = DataPermissionKeyOwner
	var query strings.Builder
	query.WriteString("INSERT INTO data_user_permission_ (_data, _user, _permission) VALUES ")
	for idx, id := range data_ids {
		if idx > 0 {
			query.WriteString(", ")
		}

		fmt.Fprintf(&query, "($%d, $1, $2)", idx+argsOffset+1)
		args[idx+argsOffset] = id
	}

	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", query.String(),
			"args", args,
		).Error("could not create data user permissions")
		return err
	}

	return nil
}

func (s *DataService) dataCreateCreatorUser(
	tx pgx.Tx,
	data []DataCreate,
	data_ids []uuid.UUID,
) error {
	const FIELDS = 3

	var query strings.Builder
	args := make([]any, len(data_ids)*FIELDS)
	query.WriteString(
		`INSERT INTO data_creator_user_ (_data, _creator, _origin) VALUES `,
	)
	for idx, datum := range data {
		if idx > 0 {
			query.WriteString(", ")
		}

		idx_data := idx * FIELDS
		idx_creator := idx_data + 1
		idx_origin := idx_creator + 1
		fmt.Fprintf(
			&query,
			"($%d, $%d, $%d)",
			idx_data+1,
			idx_creator+1,
			idx_origin+1,
		)

		args[idx_data] = data_ids[idx]
		args[idx_creator] = datum.Creator.Id
		args[idx_origin] = datum.Creator.Origin
	}
	_, err := tx.Exec(
		s.ctx,
		query.String(),
		args...,
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not create data creator user")
		return err
	}

	return nil
}

func (s *DataService) dataCreateProperties(tx pgx.Tx, data []DataCreate, data_ids []uuid.UUID) error {
	num_properties := 0
	for _, datum := range data {
		num_properties += len(datum.Properties)
	}
	if num_properties == 0 {
		return nil
	}

	const numFields = 4
	args := make([]any, num_properties*numFields)
	var query strings.Builder
	query.WriteString(
		`INSERT INTO data_properties_ (_data, _key, _type, value) VALUES `,
	)
	idx := 0
	for ddx, datum := range data {
		for _, property := range datum.Properties {
			data_idx := idx * numFields
			key_idx := data_idx + 1
			type_idx := key_idx + 1
			value_idx := type_idx + 1

			args[data_idx] = data_ids[ddx]
			args[key_idx] = property.Key
			args[type_idx] = property.Type
			args[value_idx] = property.Value

			if idx > 0 {
				query.WriteString(", ")
			}
			fmt.Fprintf(
				&query,
				"($%d, $%d, $%d, $%d)",
				data_idx+1,
				key_idx+1,
				type_idx+1,
				value_idx+1,
			)

			idx += 1
		}
	}

	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", query.String(),
			"args", args,
		).Error("could not create data properties")
		return err
	}

	return nil
}

func (s *DataService) dataCreateNotes(
	tx pgx.Tx,
	data []DataCreate,
	data_ids []uuid.UUID,
) error {
	num_notes := 0
	for _, datum := range data {
		num_notes += len(datum.Notes)
	}
	if num_notes == 0 {
		return nil
	}

	const numFields = 5
	args := make([]any, num_notes*numFields)
	var query strings.Builder
	query.WriteString(
		`INSERT INTO data_properties_ (_data, _creator, timestamp, visibility, content) VALUES `,
	)
	idx := 0
	for ddx, datum := range data {
		for _, note := range datum.Notes {
			data_idx := idx * numFields
			creator_idx := data_idx + 1
			timestamp_idx := creator_idx + 1
			visibility_idx := timestamp_idx + 1
			content_idx := visibility_idx + 1

			args[data_idx] = data_ids[ddx]
			args[creator_idx] = datum.Creator.Id
			args[timestamp_idx] = note.Timestamp
			args[visibility_idx] = note.Visibility
			args[content_idx] = note.Content

			if idx > 0 {
				query.WriteString(", ")
			}
			fmt.Fprintf(
				&query,
				"($%d, $%d, $%d, $%d, $%d)",
				data_idx+1,
				creator_idx+1,
				timestamp_idx+1,
				visibility_idx+1,
				content_idx+1,
			)

			idx += 1
		}
	}

	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", query.String(),
			"args", args,
		).Error("could not create data notes")
		return err
	}

	return nil
}

// dataCreateDataIngestionScriptSources returns the file paths each file should be saved to.
func (s *DataService) dataCreateDataIngestionScriptSources(
	tx pgx.Tx,
	data_id uuid.UUID,
	sources map[uuid.UUID][]*multipart.FileHeader,
) (map[uuid.UUID][]string, error) {
	const argsOffset = 1
	const argsPerFile = 2

	base_dir, err := s.app_service.AppDataDir(AppDataDirIngestionScriptSource)
	if err != nil {
		s.logger.With("error", err).Error("could not get ingestion script source data directory")
		return nil, err
	}

	num_args := argsOffset + len(sources)
	for _, files := range sources {
		num_args += len(files) * argsPerFile
	}
	args := make([]any, num_args)
	args[0] = data_id
	var query strings.Builder
	query.WriteString(
		"INSERT INTO data_ingestion_script_source_ (_data, _source, _path, _filename) VALUES ",
	)
	rxidx := 0
	argidx := argsOffset
	paths := make(map[uuid.UUID][]string, len(sources))
	for source, files := range sources {
		src_idx := argidx
		argidx += 1
		args[src_idx] = source

		source_paths := make([]string, len(files))
		for f_idx, file := range files {
			if rxidx > 0 {
				query.WriteString(", ")
			}

			path_idx := src_idx + 1
			filename_idx := path_idx + 1
			argidx += argsPerFile

			filename := fmt.Sprintf("%s.%d.%s", data_id, f_idx, file.Filename)
			path := filepath.Join(base_dir, source.String(), filename)
			args[path_idx] = path
			args[filename_idx] = file.Filename

			fmt.Fprintf(
				&query,
				"($1, $%d, $%d, $%d)",
				src_idx+1,
				path_idx+1,
				filename_idx+1,
			)
			source_paths[f_idx] = path

			rxidx += 1
		}

		paths[source] = source_paths
	}

	return paths, nil
}

type DataWithOrigin struct {
	Data   DataRx
	Origin uuid.UUID
}

type OrphanedDataResources struct {
	Data      []DataWithOrigin
	Origins   []DataOriginRx
	DataTypes []DataType
}

func (s *DataService) OrphanedData(user uuid.UUID) (OrphanedDataResources, error) {
	type rx struct {
		Id          uuid.UUID       `db:"id"`
		Type        uuid.UUID       `db:"type"`
		CreatorType DataCreatorType `db:"creator_type"`
		Timestamp   time.Time       `db:"timestamp"`
		Visibility  Visibility      `db:"visibility"`
		Origin      uuid.UUID       `db:"origin"`
	}

	query := fmt.Sprintf(
		`SELECT 
			d._id as id, 
			d._type as type, 
			d._creator_type as creator_type, 
			d.timestamp as timestamp, 
			d.visibility as visibility, 
			c._origin as origin
		FROM data_ d JOIN data_creator_user_ c ON d._id=c._data
		LEFT JOIN project_data_membership_ p ON d._id=p._data
		WHERE d._creator_type='%s' 
		AND c._creator=$1 
		AND p._data IS NULL`,
		DataCreatorTypeUser,
	)
	rows, _ := s.db.Conn.Query(s.ctx, query, user)
	rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[rx])
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user,
		).Error("could not get orphaned data")
		return OrphanedDataResources{}, err
	}

	origin_ids := make([]uuid.UUID, 0, 32)
	type_ids := make([]uuid.UUID, 0, 32)
	for _, r := range rxs {
		if !slices.Contains(origin_ids, r.Origin) {
			origin_ids = append(origin_ids, r.Origin)
		}

		if !slices.Contains(type_ids, r.Type) {
			type_ids = append(type_ids, r.Type)
		}
	}

	origins, err := s.DataOriginsByIds(origin_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"origins", origin_ids,
		).Error("could not get data origins")
		return OrphanedDataResources{}, err
	}

	data_types, err := s.DataTypesById(type_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"data types", type_ids,
		).Error("could not get data types")
		return OrphanedDataResources{}, err
	}

	data := make([]DataWithOrigin, len(rxs))
	for idx, r := range rxs {
		data[idx] = DataWithOrigin{
			Data: DataRx{
				Id:          r.Id,
				Type:        r.Type,
				CreatorType: r.CreatorType,
				Timestamp:   r.Timestamp,
				Visibility:  r.Visibility,
			},
			Origin: r.Origin,
		}
	}

	return OrphanedDataResources{
		Data:      data,
		Origins:   origins,
		DataTypes: data_types,
	}, nil
}

type DataProjectResources struct {
	Project           Project
	MembershipCreator uuid.UUID
	Label             *string
	Tags              []string
	Properties        []Property
	Notes             []ProjectDataNote
}

func (s *DataService) DataProjectsResources(data uuid.UUID) ([]DataProjectResources, error) {
	memberships, err := s.dataProjectMemberships(data)
	if err != nil {
		return nil, err
	}

	project_ids := make([]uuid.UUID, len(memberships))
	for idx, membership := range memberships {
		project_ids[idx] = membership.Project
	}
	projects, err := s.dataProjects(project_ids)
	if err != nil {
		return nil, err
	}

	tags, err := s.dataProjectTags(data)
	if err != nil {
		return nil, err
	}

	properties, err := s.dataProjectProperties(data)
	if err != nil {
		return nil, err
	}

	notes, err := s.dataProjectNotes(data)
	if err != nil {
		return nil, err
	}

	resources := make([]DataProjectResources, len(memberships))
	for idx, membership := range memberships {
		resources[idx].MembershipCreator = membership.Creator
		resources[idx].Label = membership.Label

		idx := slices.IndexFunc(projects, func(res Project) bool {
			return res.Id == membership.Project
		})
		if idx < 0 {
			s.logger.With(
				"project", membership.Project,
			).Error("project not found")
			panic("project not found")
		}
		resources[idx].Project = projects[idx]

		idx = slices.IndexFunc(tags, func(res projectTags) bool {
			return res.Project == membership.Project
		})
		if idx > -1 {
			resources[idx].Tags = tags[idx].Tags
		}

		idx = slices.IndexFunc(properties, func(res projectProperties) bool {
			return res.Project == membership.Project
		})
		if idx > -1 {
			resources[idx].Properties = properties[idx].Properties
		}

		idx = slices.IndexFunc(notes, func(res projectNotes) bool {
			return res.Project == membership.Project
		})
		if idx > -1 {
			resources[idx].Notes = notes[idx].Notes
		}
	}

	return resources, nil
}

type membershipRx struct {
	Project uuid.UUID `db:"_project"`
	Creator uuid.UUID `db:"_creator"`
	Label   *string   `db:"label"`
}

func (s *DataService) dataProjectMemberships(data uuid.UUID) ([]membershipRx, error) {
	query :=
		`SELECT _project, _creator, label FROM project_data_membership_
		WHERE _data=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, data)
	rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[membershipRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not get project data memberships")
		return nil, err
	}

	return rxs, nil
}

func (s *DataService) dataProjects(projects []uuid.UUID) ([]Project, error) {
	query :=
		`SELECT _id, _creator, label, description, visibility
		FROM project_ WHERE _id=ANY($1)`
	rows, _ := s.db.Conn.Query(s.ctx, query, projects)
	rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[Project])
	if err != nil {
		s.logger.With(
			"error", err,
			"projects", projects,
		).Error("could not get projects")
		return nil, err
	}

	return rxs, nil
}

type projectTags struct {
	Project uuid.UUID
	Tags    []string
}

func (s *DataService) dataProjectTags(data uuid.UUID) ([]projectTags, error) {
	type rx struct {
		Project uuid.UUID `db:"_project"`
		Tag     string    `db:"_tag"`
	}

	query := "SELECT _project, _tag FROM project_data_tag_ WHERE _data=$1"
	rows, _ := s.db.Conn.Query(s.ctx, query, data)
	rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[rx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data project tags")
		return nil, err
	}

	var tags []projectTags
	for _, r := range rxs {
		idx := slices.IndexFunc(tags, func(tag projectTags) bool {
			return tag.Project == r.Project
		})
		if idx < 0 {
			tags = append(tags, projectTags{Project: r.Project, Tags: []string{r.Tag}})
		} else {
			tags[idx].Tags = append(tags[idx].Tags, r.Tag)
		}
	}

	return tags, nil
}

type projectProperties struct {
	Project    uuid.UUID
	Properties []Property
}

func (s *DataService) dataProjectProperties(data uuid.UUID) ([]projectProperties, error) {
	type rx struct {
		Project uuid.UUID    `db:"_project"`
		Key     string       `db:"_key"`
		Type    PropertyType `db:"_type"`
		Value   any          `db:"value"`
	}

	query :=
		`SELECT _project, _key, _type, value FROM project_data_property_ 
		WHERE _data=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, data)
	rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[rx])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data project properties")
		return nil, err
	}

	var properties []projectProperties
	for _, r := range rxs {
		prop := Property{
			Key:   r.Key,
			Type:  r.Type,
			Value: r.Value,
		}

		idx := slices.IndexFunc(properties, func(prop projectProperties) bool {
			return prop.Project == r.Project
		})
		if idx < 0 {
			properties = append(
				properties,
				projectProperties{
					Project:    r.Project,
					Properties: []Property{prop},
				},
			)
		} else {
			properties[idx].Properties = append(
				properties[idx].Properties,
				prop,
			)
		}
	}

	return properties, nil
}

type ProjectDataNote struct {
	Id         uuid.UUID  `db:"_id"`
	Project    uuid.UUID  `db:"_project"`
	Data       uuid.UUID  `db:"_data"`
	Creator    uuid.UUID  `db:"_creator"`
	Timestamp  time.Time  `db:"timestamp"`
	Visibility Visibility `db:"visibility"`
	Content    string     `db:"content"`
}

type projectNotes struct {
	Project uuid.UUID
	Notes   []ProjectDataNote
}

func (s *DataService) dataProjectNotes(data uuid.UUID) ([]projectNotes, error) {

	query :=
		`SELECT _id, _project, _data, _creator, timestamp, visibility, content 
		FROM project_data_note_ WHERE _data=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, data)
	rxs, err := pgx.CollectRows(rows, pgx.RowToStructByName[ProjectDataNote])
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data,
		).Error("could not get data project notes")
		return nil, err
	}

	var notes []projectNotes
	for _, r := range rxs {
		idx := slices.IndexFunc(notes, func(note projectNotes) bool {
			return note.Project == r.Project
		})
		if idx < 0 {
			notes = append(
				notes,
				projectNotes{
					Project: r.Project,
					Notes:   []ProjectDataNote{r},
				},
			)
		} else {
			notes[idx].Notes = append(
				notes[idx].Notes,
				r,
			)
		}
	}

	return notes, nil
}
