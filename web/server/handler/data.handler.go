package handler

import (
	"net/http"
	"syredb/database"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type DataHandler struct {
	db           *database.DBConnection
	data_service *service.DataService
	user_service *service.UserService
}

func NewDataHandler(
	db *database.DBConnection,
	data_service *service.DataService,
	user_service *service.UserService,
) *DataHandler {
	return &DataHandler{
		db:           db,
		data_service: data_service,
		user_service: user_service,
	}
}

func (h *DataHandler) GetDataSchemasAll(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	projects, err := h.data_service.GetDataSchemasAll(user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user projects")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, projects)
}

func (h *DataHandler) CreateDataSchema(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	var data_schema service.DataSchemaCreate
	err := c.Bind(&data_schema)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"payload", c.Request().Body,
		).Error("could not bind data")
	}

	user_role, err := h.user_service.UserRole(user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user role")
	}
	if user_role != service.UserRoleAdmin &&
		user_role != service.UserRoleOwner {
		c.Logger().With(
			"user", user_id,
		).Debug("insufficient permissions to create data schema")
		return c.NoContent(http.StatusUnauthorized)
	}

	err = h.data_service.DataSchemaCreate(user_id, data_schema)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user projects")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
