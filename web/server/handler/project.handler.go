package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syredb/database"
	"syredb/service"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
)

type ProjectHandler struct {
	db              *database.DBConnection
	project_service *service.ProjectService
	sample_service  *service.SampleService
}

func NewProjectHandler(
	db *database.DBConnection,
	project_service *service.ProjectService,
	sample_service *service.SampleService,

) *ProjectHandler {
	return &ProjectHandler{
		db:              db,
		project_service: project_service,
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
	var project service.ProjectCreate
	err := c.Bind(&project)
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

func (h *ProjectHandler) GetProjectResources(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	project_id, err := uuid.Parse(c.QueryParam("project"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"id", c.Param("id"),
		).Error("could not parse project id")
		return c.NoContent(http.StatusBadRequest)
	}

	_, err = h.project_service.ProjectUserPermission(project_id, user_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Logger().With(
				"user", user_id,
				"project", project_id,
			).Error("insufficient permission for project resources")
			return c.NoContent(http.StatusUnauthorized)
		}

		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get user project permissions")
		return c.NoContent(http.StatusInternalServerError)
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

type ProjectSampleCreatePayload struct {
	Project uuid.UUID
	Samples []ProjectSampleCreate
}

type ProjectSampleCreate struct {
	Label      string
	Tags       []string
	Properties []service.Property
	Data       []ProjectSampleDataCreate
	Notes      []service.ProjectSampleNoteCreate
}

type ProjectSampleDataCreate struct {
	Schema     uuid.UUID
	File       uuid.UUID
	Timestamp  time.Time
	Properties []service.ProjectSampleDataPropertyCreate
}

func (h *ProjectHandler) CreateProjectSamples(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	// form, err := c.MultipartForm()
	// if err != nil {
	// 	c.Logger().With("error", err).Error("could not parse form data")
	// 	return c.NoContent(http.StatusInternalServerError)
	// }

	err := error(nil)
	var payload ProjectSampleCreatePayload
	payload.Project, err = uuid.Parse(c.FormValue("project"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"projet", c.FormValue("project"),
		).Error("could not parse projet")
		return c.NoContent(http.StatusBadRequest)
	}

	user_permission, err := h.project_service.ProjectUserPermission(payload.Project, user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", payload.Project,
		).Error("could not get project user permission")
	}
	if user_permission != service.ProjectUserPermissionOwner &&
		user_permission != service.ProjectUserPermissionAdmin &&
		user_permission != service.ProjectUserPermissionReadWrite {
		c.Logger().With(
			"user", user_id,
			"project", payload.Project,
		).Debug("insufficient permissions to create samples for project")
		return c.NoContent(http.StatusUnauthorized)
	}

	err = json.Unmarshal([]byte(c.FormValue("samples")), &payload.Samples)
	if err != nil {
		c.Logger().With(
			"error", err,
			"samples", c.FormValue("samples"),
		).Error("could not parse samples")
		return c.NoContent(http.StatusBadRequest)
	}

	samples := make([]service.ProjectSampleCreate, len(payload.Samples))
	for sidx, sample_info := range payload.Samples {
		sample := service.ProjectSampleCreate{
			Label:      sample_info.Label,
			Tags:       sample_info.Tags,
			Properties: sample_info.Properties,
			Data:       make([]service.ProjectSampleDataCreate, len(sample_info.Data)),
			Notes:      sample_info.Notes,
		}
		for didx, data_info := range sample_info.Data {
			header, err := c.FormFile(fmt.Sprintf("datafiles[%s]", data_info.File))
			if err != nil {
				c.Logger().With(
					"error", err,
					"data file", data_info.File,
				).Error("data file not found")
				return c.NoContent(http.StatusInternalServerError)
			}

			src, err := header.Open()
			if err != nil {
				c.Logger().With(
					"error", err,
					"data file", header.Filename,
				).Error("could not open data file")
				return c.NoContent(http.StatusInternalServerError)
			}
			defer src.Close()

			ext := filepath.Ext(header.Filename)
			filename := strings.TrimSuffix(header.Filename, ext)
			tmpname := fmt.Sprintf("%s.*%s", filename, ext)
			dst, err := os.CreateTemp("", tmpname)
			if err != nil {
				c.Logger().With("error", err).Error("could not create temporary file")
				return c.NoContent(http.StatusInternalServerError)
			}
			defer dst.Close()

			_, err = io.Copy(dst, src)
			if err != nil {
				c.Logger().With("error", err).Error("could not copy data to file")
				return c.NoContent(http.StatusInternalServerError)
			}

			dst.Seek(0, 0)
			file_info := service.FileInfo{
				Name: header.Filename,
				Size: header.Size,
				File: dst,
			}
			data := service.ProjectSampleDataCreate{
				Schema:     data_info.Schema,
				File:       file_info,
				Timestamp:  data_info.Timestamp,
				Properties: data_info.Properties,
			}
			sample.Data[didx] = data
		}
		samples[sidx] = sample
	}

	err = h.project_service.CreateProjectSamples(user_id, payload.Project, samples)
	if err != nil {
		if errors.Is(err, &service.InsufficientPermissionsError{}) {
			c.Logger().With(
				"user", user_id,
				"project", payload.Project,
			).Error("insufficient permission to create project samples")
			return c.NoContent(http.StatusUnauthorized)
		}

		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", payload.Project,
			"samples", payload.Samples,
		).Error("could not create project samples")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *ProjectHandler) GetProjectSampleResources(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	project_id, err := uuid.Parse(c.QueryParam("project"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"project", c.QueryParam("project"),
		).Error("could not parse project id")
		return c.NoContent(http.StatusBadRequest)
	}
	sample_id, err := uuid.Parse(c.QueryParam("sample"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"sample", c.QueryParam("sample"),
		).Error("could not parse sample id")
		return c.NoContent(http.StatusBadRequest)
	}

	_, err = h.project_service.ProjectUserPermission(project_id, user_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Logger().With(
				"user", user_id,
				"project", project_id,
				"sample", sample_id,
			).Error("insufficient permission to get project sample resources (project)")
			return c.NoContent(http.StatusUnauthorized)
		}

		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get user project permissions")
		return c.NoContent(http.StatusInternalServerError)
	}

	resources, err := h.project_service.GetProjectSampleResources(user_id, project_id, sample_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
			"sample", sample_id,
		).Error("could not get project sample resources")
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(resources.SampleUserPermissions) == 0 {
		c.Logger().With(
			"user", user_id,
			"project", project_id,
			"sample", sample_id,
		).Error("insufficient permission to get project sample resources (sample)")
		return c.NoContent(http.StatusUnauthorized)
	}

	return c.JSON(http.StatusOK, resources)
}

func (h *ProjectHandler) UpdateProjectSample(c *echo.Context) error {
	type Payload struct {
		Project uuid.UUID                   `json:"project"`
		Update  service.ProjectSampleUpdate `json:"update"`
	}

	user_id := c.Get(UserIdKey).(uuid.UUID)
	var payload Payload
	err := c.Bind(&payload)
	if err != nil {
		c.Logger().With(
			"error", err,
			"payload", c.Request().Body,
		).Error("could not bind payload")
		return c.NoContent(http.StatusBadRequest)
	}

	resources, err := h.project_service.GetProjectSampleResources(
		user_id,
		payload.Project,
		payload.Update.Id,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", payload.Project,
			"sample", payload.Update.Id,
		).Error("could not get project sample resources")
		return c.NoContent(http.StatusInternalServerError)
	}

	user_permissions_idx := slices.IndexFunc(
		resources.ProjectSampleUserPermissions,
		func(permissions service.ProjectSampleUserPermissions) bool {
			return permissions.User == user_id
		},
	)
	if user_permissions_idx < 0 {
		c.Logger().With(
			"user", user_id,
			"project", payload.Project,
			"sample", payload.Update.Id,
		).Error("insuffcient permissions to update project sample")
		return c.NoContent(http.StatusUnauthorized)
	}
	user_permissions := resources.ProjectSampleUserPermissions[user_permissions_idx].Permissions

	if payload.Update.Label != resources.ProjectMembership.Label &&
		!slices.Contains(user_permissions, service.ProjectSampleUserPermissionModifyLabel) {
		c.Logger().With(
			"user", user_id,
			"project", payload.Project,
			"sample", payload.Update.Id,
		).Error("insuffcient permissions to update project sample label")
		return c.NoContent(http.StatusUnauthorized)
	}

	tags_equal := len(payload.Update.Tags) == len(resources.ProjectTags)
	if tags_equal {
		for _, tag := range payload.Update.Tags {
			if !slices.Contains(resources.ProjectTags, tag) {
				tags_equal = false
				break
			}
		}
	}
	if !tags_equal &&
		!slices.Contains(user_permissions, service.ProjectSampleUserPermissionModifyTags) {
		c.Logger().With(
			"user", user_id,
			"project", payload.Project,
			"sample", payload.Update.Id,
		).Error("insuffcient permissions to update project sample tags")
		return c.NoContent(http.StatusUnauthorized)
	}

	err = h.project_service.UpdateProjectSample(payload.Project, payload.Update)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", payload.Project,
			"update", payload.Update,
		).Error("could not update project sample")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
