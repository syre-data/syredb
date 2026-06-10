package handler

import (
	"errors"
	"net/http"
	"syredb/database"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
)

type ProjectHandler struct {
	db              *database.DBConnection
	project_service *service.ProjectService
	user_service    *service.UserService
	sample_service  *service.SampleService
}

func NewProjectHandler(
	db *database.DBConnection,
	project_service *service.ProjectService,
	user_service *service.UserService,
	sample_service *service.SampleService,

) *ProjectHandler {
	return &ProjectHandler{
		db:              db,
		project_service: project_service,
		user_service:    user_service,
		sample_service:  sample_service,
	}
}

func (h *ProjectHandler) GetUserProjects(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
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

func (h *ProjectHandler) GetProjectWithUserPermission(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	project_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"id", c.Param("id"),
		).Error("could not parse project id")
		return c.NoContent(http.StatusBadRequest)
	}

	project, err := h.project_service.GetProjectWithUserPermission(user_id, project_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Logger().With(
				"user", user_id,
				"project", project_id,
			).Error("insufficient user permission")
			return c.NoContent(http.StatusUnauthorized)
		} else {

			c.Logger().With(
				"error", err,
				"user", user_id,
				"project", project_id,
			).Error("could not get project")
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, project)
}

func (h *ProjectHandler) CreateProject(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	has_permission, err := h.user_service.UserHasPermission(user_id, service.DbPermissionProjectCreate)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user permission")
	}
	if !has_permission {
		c.Logger().With(
			"user", user_id,
		).Debug("insufficient permissions to create data schema")
		return c.NoContent(http.StatusUnauthorized)
	}

	var project service.ProjectCreate
	err = c.Bind(&project)
	if err != nil {
		c.Logger().With(
			"error", err,
			"payload", c.Request().Body,
		).Error("could not bind payload")
	}

	id, err := h.project_service.CreateProject(user_id, project)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project,
			"request body", c.Request().Body,
		).Error("could not create project")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, id)
}

func (h *ProjectHandler) ProjectResources(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	project_id, err := uuid.Parse(c.QueryParam("project"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"id", c.Param("id"),
		).Error("could not parse project id")
		return c.NoContent(http.StatusBadRequest)
	}

	sufficient_permission, err := h.project_service.UserHasProjectPermission(
		service.ProjectPermissionRead,
		user_id,
		project_id,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get user project permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !sufficient_permission {
		c.Logger().With(
			"user", user_id,
			"project", project_id,
		).Error("insufficient permission for project resources")
		return c.NoContent(http.StatusUnauthorized)
	}

	resources, err := h.project_service.GetProjectResources(user_id, project_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get project resources")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, resources)
}

func (h *ProjectHandler) AddData(c *echo.Context) error {
	type payloadData struct {
		Project uuid.UUID `form:"project"`
		Data    uuid.UUID `form:"data"`
		Label   *string   `form:"label"`
	}

	user_id := c.Get(UserIdKey).(uuid.UUID)
	var payload payloadData
	err := c.Bind(&payload)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not bind payload")
		return c.NoContent(http.StatusBadRequest)
	}

	has_permission, err := h.project_service.UserHasProjectPermission(
		service.ProjectPermissionDataCreate,
		user_id,
		payload.Project,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", payload.Project,
		).Error("could not get project user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !has_permission {
		return c.NoContent(http.StatusUnauthorized)
	}

	err = h.project_service.DataMembershipCreate(
		payload.Project,
		payload.Data,
		user_id,
		payload.Label,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"data", payload,
		).Error("could not create project data membership")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
