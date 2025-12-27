package app

import (
	"context"
	"log/slog"

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
	ctx       context.Context
	logger    *slog.Logger
	db        *DbConnection
	app_state *AppState
}

func NewDataService(
	logger *slog.Logger,
	db *DbConnection,
	app_state *AppState,
) *DataService {
	return &DataService{logger: logger, db: db, app_state: app_state}
}

func (s *DataService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

type ColumnSchema struct {
	Label string   `json:"label"`
	DType DataType `json:"dtype"`
}

const (
	STORAGE_INTERNAL = Storage("internal")
	STORAGE_FILE     = Storage("file")
)

type Storage string

type DataSchema struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Schema      []ColumnSchema
	Storage     Storage
	Label       string
	Description string
}

func (s *DataService) GetDataSchemas() ([]DataSchema, error) {
	s.app_state._lock.RLock()
	user_id := s.app_state.user_id
	s.app_state._lock.RUnlock()
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

type DataSchemaCreate struct {
	Schema      []ColumnSchema
	Storage     Storage
	Label       string
	Description string
}

func (s *DataService) DataSchemaCreate(data_schema DataSchemaCreate) (Ok, error) {
	s.app_state._lock.RLock()
	user_id := s.app_state.user_id
	s.app_state._lock.RUnlock()
	if user_id == uuid.Nil {
		return Ok{}, &UserNotAuthenticatedError{}
	}

	var user_role UserRole
	user_role_query := "SELECT role FROM user_ WHERE _id=$1"
	err := s.db.conn.QueryRow(
		s.ctx,
		user_role_query,
		user_id,
	).Scan(&user_role)
	if err != nil ||
		(user_role != USER_ROLE_OWNER &&
			user_role != USER_ROLE_ADMIN) {
		s.logger.With("user", user_id).Debug(
			"insufficient permissions to create data schema for user",
		)
		return Ok{}, &InsufficientPermissionsError{}
	}

	create_schema_query := "INSERT INTO data_schema_ (_creator, _schema, _storage, label, description) VALUES ($1, $2, $3, $4, $5)"
	_, err = s.db.conn.Exec(
		s.ctx,
		create_schema_query,
		user_id,
		data_schema.Schema,
		data_schema.Storage,
		data_schema.Label,
		data_schema.Description,
	)
	if err != nil {
		s.logger.With("error", err).Error("could not create data schema")
		return Ok{}, err
	}

	return Ok{}, nil
}
