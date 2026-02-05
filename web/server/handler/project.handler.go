package handler

import (
	"net/http"
	"syredb/database"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type ProjectHandler struct {
	db              *database.DbConnection
	project_service *service.ProjectService
}

func NewProjectHandler(
	db *database.DbConnection,
	project_service *service.ProjectService,
) *ProjectHandler {
	return &ProjectHandler{db: db, project_service: project_service}
}

func (h *ProjectHandler) GetUserProjects(c *echo.Context) error {
	user_id := c.Get(USER_ID_KEY).(uuid.UUID)
	projects, err := h.project_service.GetUserProjects(user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user projects")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, projects)
}
