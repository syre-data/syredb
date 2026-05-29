package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"syredb/database"
	"syredb/service"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
)

type ApiClientHandler struct {
	db           *database.DBConnection
	auth_service *service.AuthService
	data_service *service.DataService
}

func NewApiClientHandler(
	db *database.DBConnection,
	auth_service *service.AuthService,
	data_service *service.DataService,
) *ApiClientHandler {
	return &ApiClientHandler{
		db:           db,
		auth_service: auth_service,
		data_service: data_service,
	}
}

func (h *ApiClientHandler) Authenticate(c *echo.Context) error {
	type credentials struct {
		Email      string    `form:"email"`
		Password   string    `form:"password"`
		Expiration time.Time `form:"expiration"`
	}

	var creds credentials
	err := echo.BindBody(c, &creds)
	if err != nil {
		c.Logger().With(
			"error", err,
		).Error("could not bind request data")
		return c.NoContent(http.StatusBadRequest)
	}
	if creds.Email == "" || creds.Password == "" || time.Now().After(creds.Expiration) {
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	var user_id uuid.UUID
	user_id_query := "SELECT _id FROM user_ WHERE email=$1"
	err = h.db.Conn.QueryRow(
		c.Request().Context(),
		user_id_query,
		creds.Email,
	).Scan(&user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"email", creds.Email,
		).Error("could not retrieve user id")
		return c.NoContent(http.StatusUnauthorized)
	}

	var hash string
	auth_query := "SELECT auth FROM user_auth_ WHERE _id=$1"
	err = h.db.Conn.QueryRow(c.Request().Context(), auth_query, user_id).Scan(&hash)
	if err != nil {
		c.Logger().With("error", err).Error("could not retrieve user auth hash")
		return c.NoContent(http.StatusUnauthorized)
	}

	authenticated, err := h.auth_service.ComparePasswordAndHash(creds.Password, hash)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user id", user_id,
		).Error("could not authenticate user")
		return c.NoContent(http.StatusInternalServerError)
	}

	if !authenticated {
		return echo.ErrUnauthorized
	}

	token, err := h.auth_service.CreateSession(user_id, creds.Expiration)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"expires", creds.Expiration,
		).Error("could not create user session")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.String(http.StatusOK, token.String())
}

func (h *ApiClientHandler) Deactivate(c *echo.Context) error {
	token, err := uuid.Parse(c.FormValue("token"))
	if err != nil {
		c.Logger().With(
			"error", err,
		).Info("could not parse request token")
		return c.NoContent(http.StatusBadRequest)
	}

	err = h.auth_service.DeactivateSession(token)
	if err != nil {
		c.Logger().With(
			"error", err,
			"token", token,
		).Error("could not deactivate session token")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *ApiClientHandler) DataType(c *echo.Context) error {
	type schemaField struct {
		Label    string            `json:"label"`
		Dtype    service.ValueType `json:"dtype"`
		Required bool              `json:"required"`
		Nullable bool              `json:"nullable"`
	}

	type dataSchema struct {
		Cardinality service.DataSchemaCardinality `json:"cardinality"`
		Fields      []schemaField                 `json:"fields"`
	}

	type dataSource struct {
		Label       string                        `json:"label"`
		Required    bool                          `json:"required"`
		Cardinality service.DataSourceCardinality `json:"cardinality"`
		ExtFilter   []string                      `json:"ext_filter"`
	}

	data_type_label := c.FormValue("data_type")
	if data_type_label == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	var data_type service.DataType
	data_type_id, err := uuid.Parse(data_type_label)
	if err != nil {
		data_type, err = h.data_service.DataTypeByLabel(data_type_label)
		if err != nil {
			c.Logger().With(
				"error", err,
				"data type", data_type_id,
			).Error("could not get data type")
			return c.NoContent(http.StatusNotFound)
		}
	} else {
		data_type, err = h.data_service.DataTypeById(data_type_id)
		if err != nil {
			c.Logger().With(
				"error", err,
				"data type", data_type_id,
			).Error("could not get data type")
			return c.NoContent(http.StatusNotFound)
		}
	}

	switch data_type.DataStorage() {
	case service.DataStorageExternal:
		dtype := data_type.(service.DataTypeExternal)
		sources := make([]dataSource, len(dtype.Sources))
		for idx, source := range dtype.Sources {
			sources[idx] = dataSource{
				Label:       source.Label,
				Required:    source.Required,
				Cardinality: source.Cardinality,
				ExtFilter:   source.ExtFilter,
			}
		}

		info := make(map[string]any, 3)
		info["storage"] = service.DataStorageExternal
		info["id"] = dtype.Id
		info["sources"] = sources
		return c.JSON(http.StatusOK, info)

	case service.DataStorageInternal:
		dtype := data_type.(service.DataTypeInternal)
		dschema, err := h.data_service.DataSchemaById(dtype.Schema)
		if err != nil {
			c.Logger().With(
				"error", err,
				"data type", dtype.Id,
			).Error("could not get data schema")
			return c.NoContent(http.StatusInternalServerError)
		}

		fields := make([]schemaField, len(dschema.Fields))
		for idx, field := range dschema.Fields {
			fields[idx] = schemaField{
				Label:    field.Label,
				Dtype:    field.DType,
				Required: field.Required,
				Nullable: field.Nullable,
			}
		}
		schema := dataSchema{
			Cardinality: dschema.Cardinality,
			Fields:      fields,
		}

		info := make(map[string]any, 3)
		info["storage"] = service.DataStorageInternal
		info["id"] = dtype.Id
		info["schema"] = schema
		return c.JSON(http.StatusOK, info)

	default:
		panic("unexpected service.DataStorage")
	}
}

type propertyValue struct {
	Key   string               `json:"key"`
	Type  service.PropertyType `json:"dtype"`
	Value any                  `json:"value"`
}
type note struct {
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
}
type schemaField struct {
	Label  string `json:"label"`
	Values any    `json:"values"`
}
type dataSource struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}
type dataCreate struct {
	Id         uuid.UUID           `json:"id"`
	Origin     string              `json:"origin"`
	Visibility service.Visibility  `json:"visibility"`
	Timestamp  time.Time           `json:"timestamp"`
	Properties []propertyValue     `json:"properties"`
	Tags       []string            `json:"tags"`
	Notes      []note              `json:"notes"`
	Storage    service.DataStorage `json:"storage"`
	Fields     []schemaField       `json:"fields"`
	Sources    []dataSource        `json:"sources"`
}

func (h *ApiClientHandler) DataCreate(c *echo.Context) error {
	token := c.Get(ApiClientTokenKey).(string)
	user := c.Get(UserIdKey).(uuid.UUID)
	data_str := c.FormValue("data")
	if data_str == "" {
		c.Logger().Debug("data not present")
		return c.NoContent(http.StatusBadRequest)
	}
	var data dataCreate
	err := json.Unmarshal([]byte(data_str), &data)
	if err != nil {
		c.Logger().With(
			"error", err,
			"token", token,
			"data", data_str,
		).Error("could not unmarshal data")
		return c.NoContent(http.StatusBadRequest)
	}

	origin_id, err := uuid.Parse(data.Origin)
	if err != nil {
		origin, err := h.data_service.DataOriginByLabel(data.Origin)
		if err != nil {
			c.Logger().With(
				"error", err,
				"origin", data.Origin,
			)

			if errors.Is(err, pgx.ErrNoRows) {
				return c.NoContent(http.StatusBadRequest)
			} else {
				return c.NoContent(http.StatusInternalServerError)
			}
		}

		origin_id = origin.Id
	}

	switch data.Storage {
	case service.DataStorageExternal:
		return h.dataCreateExternal(c, user, origin_id, data)
	case service.DataStorageInternal:
		return h.dataCreateInternal(c, user, origin_id, data)
	default:
		c.Logger().With("data", data).Error("invalid data storage")
		return c.NoContent(http.StatusBadRequest)
	}
}

func (h *ApiClientHandler) dataCreateExternal(
	c *echo.Context,
	user uuid.UUID,
	origin uuid.UUID,
	data dataCreate,
) error {
	panic("TODO: dataCreateExternal")
}

func (h *ApiClientHandler) dataCreateInternal(
	c *echo.Context,
	user uuid.UUID,
	origin uuid.UUID,
	data dataCreate,
) error {
	properties := make([]service.Property, len(data.Properties))
	for idx, prop := range data.Properties {
		properties[idx] = service.Property{
			Key:   prop.Key,
			Type:  prop.Type,
			Value: prop.Value,
		}
	}

	notes := make([]service.Note, len(data.Notes))
	for idx, note := range data.Notes {
		notes[idx] = service.Note{
			Timestamp:  note.Timestamp,
			Visibility: service.VisibilityPrivate,
			Content:    note.Content,
		}
	}

	values := make(map[string]any, len(data.Fields))
	for _, field := range data.Fields {
		values[field.Label] = field.Values
	}

	create := service.DataCreate{
		Type:                   data.Id,
		Origin:                 origin,
		CreatorType:            service.DataCreatorTypeUser,
		Timestamp:              data.Timestamp,
		Visibility:             data.Visibility,
		Properties:             properties,
		Notes:                  notes,
		IngestionMethod:        service.DataIngestionManual,
		Values:                 values,
		IngestionScript:        uuid.Nil,
		IngestionScriptSources: nil,
	}
	_, err := h.data_service.DataCreate([]service.DataCreate{create}, user, user)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user,
		).Error("could not create data")
		c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
