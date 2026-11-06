package handler

import (
	"net/http"
	"net/url"
	"syredb/database"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type ProjectHandler struct {
	db              *database.DBConnection
	project_service *service.ProjectService
	user_service    *service.UserService
}

func NewProjectHandler(
	db *database.DBConnection,
	project_service *service.ProjectService,
	user_service *service.UserService,

) *ProjectHandler {
	return &ProjectHandler{
		db:              db,
		project_service: project_service,
		user_service:    user_service,
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
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not retrieve user projects",
			Payload:   nil,
		})
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
		).Info("could not parse project id")
		return c.JSON(http.StatusUnprocessableEntity, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_INVALID_UUID,
			Message:   "Invalid id query",
			Payload:   nil,
		})
	}

	project, valid, err := h.project_service.GetProjectWithUserPermission(user_id, project_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get project")
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not get project",
			Payload:   nil,
		})
	}
	if !valid {
		c.Logger().With(
			"user", user_id,
			"project", project_id,
		).Info("project not found")
		return c.NoContent(http.StatusNotFound)
	}
	if len(project.Permissions) == 0 {
		c.Logger().With(
			"user", user_id,
			"project", project_id,
		).Info("insufficient user permission")
		return c.NoContent(http.StatusForbidden)
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
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not create project",
			Payload:   nil,
		})
	}
	if !has_permission {
		c.Logger().With(
			"user", user_id,
		).Info("insufficient permissions to create data schema")
		return c.NoContent(http.StatusForbidden)
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
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not create project",
			Payload:   nil,
		})
	}

	res := ResourceCreated{Id: id}
	loc := url.URL{Path: "/api/project"}
	loc.RawQuery = res.ToUrlQuery().Encode()
	c.Response().Header().Set(echo.HeaderLocation, loc.String())
	return c.JSON(http.StatusCreated, res)
}

func (h *ProjectHandler) ProjectResources(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	project_id, err := uuid.Parse(c.QueryParam("project"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"id", c.Param("id"),
		).Info("could not parse project id")
		return c.JSON(http.StatusUnprocessableEntity, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_INVALID_UUID,
			Message:   "Invalid project query",
			Payload:   nil,
		})
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
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not get project resources",
			Payload:   nil,
		})
	}
	if !sufficient_permission {
		c.Logger().With(
			"user", user_id,
			"project", project_id,
		).Info("insufficient permission for project resources")
		return c.NoContent(http.StatusForbidden)
	}

	resources, err := h.project_service.GetProjectResources(user_id, project_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get project resources")
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not get project resources",
			Payload:   nil,
		})
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
		return c.JSON(http.StatusBadRequest, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_INVALID_PAYLOAD,
			Message:   "Invalid request payload",
			Payload:   nil,
		})
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
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not add data",
			Payload:   nil,
		})
	}
	if !has_permission {
		c.Logger().With(
			"user", user_id,
		).Info("insufficient permission")
		return c.NoContent(http.StatusForbidden)
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
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not add data",
			Payload:   nil,
		})
	}

	return c.NoContent(http.StatusOK)
}
