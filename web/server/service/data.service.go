package service

import (
	"context"
	"log/slog"
	"syredb/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DataType string

const (
	DATA_TYPE_STRING    DataType = "string"
	DATA_TYPE_INT       DataType = "int"
	DATA_TYPE_UINT      DataType = "uint"
	DATA_TYPE_FLOAT     DataType = "float"
	DATA_TYPE_BOOLEAN   DataType = "boolean"
	DATA_TYPE_TIMESTAMP DataType = "timestamp"
)

type SaveDataHierarchy string

const (
	SAVE_DATA_HIERARCHY_DATA_SCHEMA SaveDataHierarchy = "data_schema"
	SAVE_DATA_HIERARCHY_SAMPLE      SaveDataHierarchy = "sample"
)

type DataService struct {
	ctx    context.Context
	logger *slog.Logger
	db     *database.DbConnection
}

func NewDataService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DbConnection,
) *DataService {
	return &DataService{
		ctx:    ctx,
		logger: logger,
		db:     db,
	}
}

type ColumnSchema struct {
	Label string   `json:"label"`
	DType DataType `json:"dtype"`
}

type DataStorage string

const (
	DATA_STORAGE_INTERNAL DataStorage = "internal"
	DATA_STORAGE_EXTERNAL DataStorage = "external"
)

type DataSchema struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Schema      []ColumnSchema
	Storage     DataStorage
	Label       string
	Description string
}

func (s *DataService) GetDataSchemasAll(user_id uuid.UUID) ([]DataSchema, error) {
	data_schema_query :=
		`SELECT (_id, _creator, _schema, _storage, label, description)
		FROM data_schema_ ORDER BY _id DESC`
	rows, err := s.db.Conn.Query(s.ctx, data_schema_query)
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
