package handler

import (
	"net/http"
	"syredb/database"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type DataHandler struct {
	db           *database.DbConnection
	data_service *service.DataService
}

func NewDataHandler(
	db *database.DbConnection,
	data_service *service.DataService,
) *DataHandler {
	return &DataHandler{db: db, data_service: data_service}
}

func (h *DataHandler) GetDataSchemasAll(c *echo.Context) error {
	user_id := c.Get(USER_ID_KEY).(uuid.UUID)
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
