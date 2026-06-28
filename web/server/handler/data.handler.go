package handler

import (
	"archive/zip"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"syredb/database"
	"syredb/service"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
)

type DataHandler struct {
	db              *database.DBConnection
	data_service    *service.DataService
	app_service     *service.AppService
	user_service    *service.UserService
	project_service *service.ProjectService
}

func NewDataHandler(
	db *database.DBConnection,
	data_service *service.DataService,
	app_service *service.AppService,
	user_service *service.UserService,
	project_service *service.ProjectService,
) *DataHandler {
	return &DataHandler{
		db:              db,
		data_service:    data_service,
		app_service:     app_service,
		user_service:    user_service,
		project_service: project_service,
	}
}

func (h *DataHandler) DataTypeCreate(c *echo.Context) error {
	storage := c.QueryParam("storage")
	switch storage {
	case string(service.DataStorageInternal):
		return h.DataTypeCreateInternal(c)
	case string(service.DataStorageExternal):
		return h.DataTypeCreateExternal(c)
	default:
		return c.NoContent(http.StatusBadRequest)
	}
}

func (h *DataHandler) DataTypeCreateInternal(c *echo.Context) error {
	type dataTypeCreateInternal struct {
		Label       string
		Description string
		Schema      uuid.UUID
	}

	user_id := c.Get(UserIdKey).(uuid.UUID)
	sufficient_permission, err := h.user_service.UserHasPermission(
		user_id,
		service.DbPermissionDataTypeCreate,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not validate user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !sufficient_permission {
		c.Logger().With(
			"user", user_id,
		).Error("insuffiecient permission to create raw data type")
		return c.NoContent(http.StatusUnauthorized)
	}

	var data dataTypeCreateInternal
	err = c.Bind(&data)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not bind data")
		return c.NoContent(http.StatusBadRequest)
	}

	var description *string
	if len(data.Description) > 0 {
		description = &data.Description
	}
	err = h.data_service.DataTypeCreateInternal(user_id, data.Label, description, data.Schema)
	if err != nil {
		c.Logger().With(
			"error", err,
		).Error("could not create data type")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *DataHandler) DataTypeCreateExternal(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	sufficient_permission, err := h.user_service.UserHasPermission(
		user_id,
		service.DbPermissionDataTypeCreate,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not validate user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !sufficient_permission {
		c.Logger().With(
			"user", user_id,
		).Error("insuffiecient permission to create raw data type")
		return c.NoContent(http.StatusUnauthorized)
	}

	label := c.FormValue("label")
	if len(label) == 0 {
		return c.NoContent(http.StatusBadRequest)
	}

	var description *string
	description_str := c.FormValue("description")
	if description_str != "" {
		description = &description_str
	}

	var sources []service.ExternalSourceCreate
	err = json.Unmarshal([]byte(c.FormValue("sources")), &sources)
	if err != nil {
		c.Logger().With(
			"error", err,
		).Error("could not parse data type sources")
		return c.NoContent(http.StatusBadRequest)
	}

	data_schema := uuid.Nil
	data_schema_str := c.FormValue("data_schema")
	if data_schema_str != "" {
		data_schema, err = uuid.Parse(data_schema_str)
		if err != nil {
			c.Logger().With(
				"error", err,
				"data schema", data_schema_str,
			).Error("could not parse data schema to uuid")
			return c.NoContent(http.StatusBadRequest)
		}
	}

	var recipe *multipart.FileHeader
	recipe, _ = c.FormFile("recipe")

	err = h.data_service.DataTypeCreateExternal(user_id, label, description, sources, data_schema, recipe)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not create data type")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *DataHandler) DataTypeGet(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	data_type_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not parse data type id")
		return c.NoContent(http.StatusBadRequest)
	}

	data_type, err := h.data_service.DataTypeById(data_type_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"data type", data_type_id,
		).Error("could not get data type resources")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, data_type)
}

func (h *DataHandler) DataTypeUpdate(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	var update service.DataTypeUpdate
	err := c.Bind(&update)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"payload", c.Request().Body,
		).Error("could not bind update")
		return c.NoContent(http.StatusBadRequest)
	}

	err = h.data_service.DataTypeUpdate(update)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"udpate", update,
		).Error("could not update data type")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *DataHandler) DataTypesGetAll(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	schemas, err := h.data_service.DataTypesAll()
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get data types")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, schemas)

}

func (h *DataHandler) DataSchemasGetAll(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	schemas, err := h.data_service.DataSchemasGetAll()
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get data schemas")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, schemas)

}

func (h *DataHandler) DataSchemaCreate(c *echo.Context) error {
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

	has_permission, err := h.user_service.UserHasPermission(
		user_id,
		service.DbPermissionDataSchemaCreate,
	)
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

	err = h.data_service.DataSchemaCreate(user_id, data_schema)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not create data schema")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *DataHandler) DataSchemaResources(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	data_schema_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"data schema id", c.QueryParam("id"),
		).Error("could not parse data schema id")
		return c.NoContent(http.StatusBadRequest)
	}

	resources, err := h.data_service.DataSchemaGetResources(data_schema_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"data schema", data_schema_id,
		).Error("could not get data schema resources")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, resources)
}

func (h *DataHandler) DataSchemaUpdate(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)

	var update service.DataSchemaUpdate
	err := c.Bind(&update)
	if err != nil {
		c.Logger().With(
			"error", err,
			"request", c.Request().Body,
		).Error("could not bind request to update")
		return c.NoContent(http.StatusBadRequest)
	}

	err = h.data_service.DataSchemaUpdate(update)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"update", update,
		).Error("could not update data schema")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *DataHandler) DownloadDataValuesSingle(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	data_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"id", c.QueryParam("id"),
		).Warn("could not parse id")
		return c.NoContent(http.StatusBadRequest)
	}
	var project_id uuid.UUID
	project_id_str := c.QueryParam("project")
	if project_id_str != "" {
		project_id, err = uuid.Parse(project_id_str)
		if err != nil {
			c.Logger().With(
				"error", err,
				"project", project_id_str,
			).Warn("could not parse project id")
			return c.NoContent(http.StatusBadRequest)
		}
	}

	data_rx, err := h.data_service.DataById(data_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.NoContent(http.StatusNotFound)
		}
		return c.NoContent(http.StatusInternalServerError)
	}

	if data_rx.Visibility != service.VisibilityPublic {
		permissions, err := h.data_service.DataUserPermissions(
			user_id,
			[]uuid.UUID{data_id},
		)
		if err != nil {
			c.Logger().With(
				"error", err,
				"data", data_id,
				"user", user_id,
			).Error("could not get data user permissions")
			return c.NoContent(http.StatusInternalServerError)
		}
		if permissions[0].Data != data_id {
			c.Logger().With(
				"user", user_id,
				"data", data_id,
				"permissions", permissions,
			).Error("invalid data user permissions")
			panic("invalid data user permissions")
		}
		user_permissions := permissions[0].Permissions
		if len(user_permissions) == 0 {
			c.Logger().With(
				"data", data_id,
				"user", user_id,
			).Warn("insufficient permissions")
			return c.NoContent(http.StatusUnauthorized)
		}
	}

	var data_type_label string
	data_type, err := h.data_service.DataTypeById(data_rx.Type)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_rx,
		).Error("could not get data type")
	} else {
		switch data_type.DataStorage() {
		case service.DataStorageExternal:
			data_type_ext := data_type.(service.DataTypeExternal)
			data_type_label = data_type_ext.Label
		case service.DataStorageInternal:
			data_type_int := data_type.(service.DataTypeInternal)
			data_type_label = data_type_int.Label
		default:
			panic("unexpected service.DataStorage")
		}
	}

	values, err := h.data_service.DataValuesById(data_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_id,
		).Error("could not get data values")
		return c.NoContent(http.StatusInternalServerError)
	}

	switch values.Storage {
	case service.DataStorageExternal:
		ext_values := values.Values.([]service.DataSource)
		return h.downloadDataValuesSingleExternal(c, data_id, ext_values)
	case service.DataStorageInternal:
		var filename string
		if project_id == uuid.Nil {
			if data_type_label != "" {
				filename = fmt.Sprintf(
					"%s.%s.csv",
					sanitizeStringForFilename(data_type_label),
					formatTimeForFilename(data_rx.Timestamp),
				)
			} else {
				filename = fmt.Sprintf("%s.csv", data_id)
			}
		} else {
			project, project_err := h.project_service.ProjectById(project_id)
			if project_err != nil {
				c.Logger().With(
					"error", project_err,
					"project", project_id,
					"data", data_id,
				).Error("could not get project data membership")
			}

			membership, membership_err := h.project_service.DataMembership(project_id, data_id)
			if membership_err != nil {
				c.Logger().With(
					"error", membership_err,
					"project", project_id,
					"data", data_id,
				).Error("could not get project data membership")

				filename = fmt.Sprintf("%s.csv", data_id)
			}

			if project_err == nil && membership_err == nil {
				filename = fmt.Sprintf(
					"%s.%s.csv",
					sanitizeStringForFilename(project.Label),
					sanitizeStringForFilename(*membership.Label),
				)
			} else if membership_err != nil && membership.Label != nil {
				filename = fmt.Sprintf(
					"%s.csv",
					sanitizeStringForFilename(*membership.Label),
				)
			} else if data_type_label != "" {
				filename = fmt.Sprintf(
					"%s.%s.csv",
					sanitizeStringForFilename(data_type_label),
					formatTimeForFilename(data_rx.Timestamp),
				)
			} else {
				filename = fmt.Sprintf("%s.csv", data_id)
			}
		}

		int_values := values.Values.([]service.SchemaFieldValues)
		return h.downloadDataValuesSingleInternal(c, data_id, filename, int_values)
	default:
		panic(fmt.Sprintf("unexpected service.DataStorage: %#v", values.Storage))
	}
}

func (h *DataHandler) downloadDataValuesSingleExternal(
	c *echo.Context,
	data_id uuid.UUID,
	values []service.DataSource,
) error {
	panic("TODO: download externally stored data")
}

func (h *DataHandler) downloadDataValuesSingleInternal(
	c *echo.Context,
	data_id uuid.UUID,
	filename string,
	fields []service.SchemaFieldValues,
) error {
	if len(fields) == 0 {
		c.Logger().With(
			"data", data_id,
		).Error("invalid data schema")
		panic("invalid data schema")
	}

	data, err := h.data_service.StoredDataToCsv(fields)
	if err != nil {
		c.Logger().With(
			"error", err,
		).Error("could not transform values to csv")
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set(
		echo.HeaderContentDisposition,
		fmt.Sprintf(`attachment; filename="%s"`, filename),
	)

	return c.Blob(
		http.StatusOK,
		"text/csv; charset=utf-8",
		[]byte(data),
	)
}

func (h *DataHandler) DownloadProjectDataValuesAll(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	project_id, err := uuid.Parse(c.QueryParam("project"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"project", c.QueryParam("project"),
		).Debug("could not parse project id")
		return c.NoContent(http.StatusBadRequest)
	}

	tmpfile, err := os.CreateTemp("", "")
	if err != nil {
		c.Logger().With(
			"error", err,
		).Error("could not create temporary data file")
		return c.NoContent(http.StatusInternalServerError)
	}
	defer tmpfile.Close()

	var project_label string
	project, err := h.project_service.ProjectById(project_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"project", project_id,
		).Error("could not get project")
	} else {
		project_label = project.Label
	}

	data, err := h.data_service.ProjectDataAll(project_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"project", project_id,
		).Error("could not get user data")
		return c.NoContent(http.StatusInternalServerError)
	}
	data_ids := make([]uuid.UUID, len(data))
	for idx, d := range data {
		data_ids[idx] = d.Data.Id
	}

	data_permissions, err := h.data_service.DataUserPermissions(user_id, data_ids)
	if err != nil {
		c.Logger().With(
			"error", err,
			"project", project_id,
			"data", data_ids,
		).Error("could not get data user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}

	filtered := make([]service.ProjectDataWithMembership, 0, len(data))
	for _, d := range data {
		if d.Data.Visibility == service.VisibilityPublic {
			filtered = append(filtered, d)
		} else {
			perm_idx := slices.IndexFunc(data_permissions, func(perm service.DataUserPermissions) bool {
				return perm.Data == d.Data.Id
			})
			if perm_idx < 0 {
				c.Logger().With(
					"data", d.Data.Id,
					"user", user_id,
				).Debug("data user permission not found")
				continue
			}

			has_permission := false
			for _, perm := range data_permissions[perm_idx].Permissions {
				if perm == service.DataUserPermissionOwner ||
					perm == service.DataUserPermissionReadValues {
					has_permission = true
					break
				}
			}

			if has_permission {
				filtered = append(filtered, d)
			}
		}
	}

	filtered_ids := make([]uuid.UUID, len(filtered))
	for idx, d := range filtered {
		filtered_ids[idx] = d.Data.Id
	}
	values, err := h.data_service.DataValuesByIds(filtered_ids)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", filtered_ids,
		).Error("could not get data values")
		return c.NoContent(http.StatusInternalServerError)
	}
	data_type_ids := make([]uuid.UUID, 0, len(filtered))
	for _, d := range filtered {
		if !slices.Contains(data_type_ids, d.Data.Type) {
			data_type_ids = append(data_type_ids, d.Data.Type)
		}
	}
	data_types, err := h.data_service.DataTypesById(data_type_ids)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data types", data_type_ids,
		).Error("could not get data types")
	}

	archive := zip.NewWriter(tmpfile)
	defer archive.Close()
	for _, vals := range values {
		data_idx := slices.IndexFunc(filtered, func(d service.ProjectDataWithMembership) bool {
			return d.Data.Id == vals.Data
		})
		if data_idx < 0 {
			c.Logger().With(
				"data", vals.Data,
			).Error("could not find data")
			panic("could not find data")
		}
		vdata := filtered[data_idx]

		type_idx := slices.IndexFunc(data_types, func(t service.DataType) bool {
			switch t.DataStorage() {
			case service.DataStorageExternal:
				text := t.(service.DataTypeExternal)
				return text.Id == vdata.Data.Type
			case service.DataStorageInternal:
				tint := t.(service.DataTypeInternal)
				return tint.Id == vdata.Data.Type
			default:
				panic("unexpected service.DataStorage")
			}
		})
		if type_idx < 0 {
			c.Logger().With(
				"data", vdata.Data.Id,
				"type", vdata.Data.Type,
			).Error("could not find data type")
			panic("could not find data type")
		}

		switch vals.Storage {
		case service.DataStorageExternal:
			panic("todo")
		case service.DataStorageInternal:
			data_type := data_types[type_idx].(service.DataTypeInternal)
			fields := vals.Values.([]service.SchemaFieldValues)
			csv, err := h.data_service.StoredDataToCsv(fields)
			if err != nil {
				c.Logger().With(
					"error", err,
					"data", vals.Data,
				).Error("could not convert data values to csv")
				return c.NoContent(http.StatusInternalServerError)
			}

			filename := fmt.Sprintf(
				"%s.%s.csv",
				sanitizeStringForFilename(data_type.Label),
				formatTimeForFilename(vdata.Data.Timestamp),
			)
			if vdata.ProjectLabel != nil && *vdata.ProjectLabel != "" {
				filename = fmt.Sprintf(
					"%s.csv",
					sanitizeStringForFilename(*vdata.ProjectLabel),
				)
			}

			f, err := archive.Create(filename)
			if err != nil {
				c.Logger().With(
					"error", err,
					"data", vals.Data,
				).Error("zip archive could not create file")
				return c.NoContent(http.StatusInternalServerError)
			}
			_, err = f.Write([]byte(csv))
			if err != nil {
				c.Logger().With(
					"error", err,
					"data", vals.Data,
				).Error("could not write data to archive file")
				return c.NoContent(http.StatusInternalServerError)
			}
		default:
			panic(fmt.Sprintf("unexpected service.DataStorage: %#v", vals.Storage))
		}
	}

	archive_name := "data.zip"
	if project_label != "" {
		archive_name = fmt.Sprintf("%s.data.zip", project_label)
	}

	err = archive.Close()
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not close archive")
		return c.NoContent(http.StatusInternalServerError)
	}

	stat, err := tmpfile.Stat()
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not stat archive")
		return c.NoContent(http.StatusInternalServerError)
	}

	buf := make([]byte, stat.Size())
	_, err = tmpfile.ReadAt(buf, 0)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not read archive file")
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set(
		echo.HeaderContentDisposition,
		fmt.Sprintf(`attachment; filename="%s"`, archive_name),
	)
	return c.Blob(
		http.StatusOK,
		"text/csv; charset=utf-8",
		buf,
	)
}

func sanitizeStringForFilename(s string) string {
	invalid := regexp.MustCompile(`[^\w\d\s\.\-_]`)
	sanitized := invalid.ReplaceAllString(s, "_")
	return string(sanitized)
}

func formatTimeForFilename(t time.Time) string {
	return fmt.Sprintf(
		"%d-%02d-%02d-%02d-%02d-%02d",
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
	)
}

func (h *DataHandler) DataTypeTransformsGetAll(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	transforms, err := h.data_service.DataTypeTransformsGetAll()
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get data type transforms")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, transforms)
}

func (h *DataHandler) DataTypeTransformCreate(c *echo.Context) error {
	type transformData struct {
		Source      uuid.UUID             `form:"source"`
		Destination uuid.UUID             `form:"destination"`
		Script      *multipart.FileHeader `form:"script"`
		Label       string                `form:"label"`
		Description string                `form:"description"`
	}

	user_id := c.Get(UserIdKey).(uuid.UUID)
	has_permission, err := h.user_service.UserHasPermission(
		user_id,
		service.DbPermissionTransformCreate,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user role")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !has_permission {
		c.Logger().With(
			"user", user_id,
		).Debug("insufficient permissions to create transform")
		return c.NoContent(http.StatusUnauthorized)
	}

	var data transformData
	err = c.Bind(&data)
	if err != nil {
		c.Logger().With(
			"user", user_id,
		).Error("could not bind data type transform")
		return c.NoContent(http.StatusBadRequest)
	}

	cmd, err := data_type_transform_command_from_file_ext(filepath.Ext(data.Script.Filename))
	if err != nil {
		c.Logger().With(
			"user", user_id,
			"file name", data.Script.Filename,
		).Error("could not get data type transform command")
		return c.NoContent(http.StatusBadRequest)
	}

	transform := service.DataTypeTransformCreate{
		Creator:     user_id,
		Source:      data.Source,
		Destination: data.Destination,
		Label:       data.Label,
		Description: data.Description,
		Cmd:         cmd,
		Args:        []string{},
		Script:      data.Script,
	}
	_, err = h.data_service.DataTypeTransformCreate(transform)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not create transform")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func data_type_transform_command_from_file_ext(ext string) (string, error) {
	switch ext {
	case ".py":
		return "python", nil
	default:
		return "", errors.New("unknown file type")
	}
}

// # Notes
// + `Values` is only valid for internal storage.
// `Sources` is only valid for external storage.
// If source is single cardinality value is `string`,
// If multiple cardinality it is `[]string`
// with each string being the name of the file.
type DataIngest struct {
	Type       uuid.UUID
	Creator    uuid.UUID
	Origin     uuid.UUID
	Timestamp  time.Time
	Visibility service.Visibility
	Properties []service.Property
	Notes      []service.Note
	Values     map[string]any
	Sources    map[uuid.UUID]any
}

type DataIngestSourceInfo struct {
	Cardinality service.DataSourceCardinality
	Files       any
	Paths       any
}

func (h *DataHandler) DataIngest(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)

	sufficient_permission, err := h.user_service.UserHasPermission(
		user_id,
		service.DbPermissionDataCreate,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !sufficient_permission {
		c.Logger().With(
			"user", user_id,
		).Debug("insufficient permission to create data")
		return c.NoContent(http.StatusUnauthorized)
	}

	project_id := uuid.Nil
	project_id_str := c.QueryParam("project")
	if project_id_str != "" {
		project_id, err = uuid.Parse(project_id_str)
		if err != nil {
			c.Logger().With(
				"error", err,
				"id", c.Param("id"),
			).Debug("could not parse project id")
			return c.NoContent(http.StatusBadRequest)
		}

		sufficient_permission_project, err := h.project_service.UserHasProjectPermission(
			service.ProjectPermissionDataCreate,
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
		if !sufficient_permission_project {
			c.Logger().With(
				"user", user_id,
				"project", project_id,
			).Debug("insufficient permission to create project data")
			return c.NoContent(http.StatusUnauthorized)
		}
	}

	origin_web_client, err := h.data_service.DataOriginByLabel(service.DATA_ORIGIN_WEB_CLIENT_LABEL)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get web client data origin")
		return c.NoContent(http.StatusInternalServerError)
	}

	var info []DataIngest
	err = json.Unmarshal([]byte(c.FormValue("data")), &info)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", c.FormValue("data"),
		).Debug("could not parse data")
		return c.NoContent(http.StatusBadRequest)
	}

	var project_labels []string
	if project_id != uuid.Nil {
		err = json.Unmarshal([]byte(c.FormValue("project_labels")), &project_labels)
		if err != nil {
			c.Logger().With(
				"error", err,
				"data", c.FormValue("data"),
			).Debug("could not parse data")
			return c.NoContent(http.StatusBadRequest)
		}
		if len(project_labels) != len(info) {
			c.Logger().With(
				"user", user_id,
				"data", info,
				"project labels", project_labels,
			).Debug("invalid project labels")
			return c.NoContent(http.StatusBadRequest)
		}
	}

	data_type_ids := make([]uuid.UUID, 0, len(info))
	for _, datum := range info {
		if !slices.Contains(data_type_ids, datum.Type) {
			data_type_ids = append(data_type_ids, datum.Type)
		}
	}
	data_types, err := h.data_service.DataTypesById(data_type_ids)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data types", data_type_ids,
		).Error("could not get data types")
		return c.NoContent(http.StatusInternalServerError)
	}
	data_types_by_idx := make([]int, len(info))
	for idx, datum := range info {
		data_type_idx := slices.IndexFunc(data_types, func(dtype service.DataType) bool {
			switch dtype.DataStorage() {
			case service.DataStorageExternal:
				dt := dtype.(service.DataTypeExternal)
				return dt.Id == datum.Type
			case service.DataStorageInternal:
				dt := dtype.(service.DataTypeInternal)
				return dt.Id == datum.Type
			default:
				panic("unexpected service.DataStorage")
			}
		})
		if data_type_idx < 0 {
			panic(fmt.Sprintf("invalid data type `%s`", datum.Type))
		}
	}

	source_base_path, err := h.app_service.AppDataDir(service.AppDataDirDataSource)
	if err != nil {
		c.Logger().With(
			"error", err,
		).Warn("could not get data source path")
		return c.NoContent(http.StatusInternalServerError)
	}

	data := make([]service.DataCreate, 0, len(info))
	for idx, datum := range info {
		data[idx].Type = datum.Type
		data[idx].Creator = service.DataCreatorUser{
			Id:     user_id,
			Origin: origin_web_client.Id,
		}
		data[idx].Timestamp = datum.Timestamp
		data[idx].Visibility = datum.Visibility
		data[idx].Properties = datum.Properties
		data[idx].Notes = datum.Notes

		switch datum.IngestionMethod {
		case service.DataIngestionManual:
			data[idx].Values = service.DataCreateValues{
				Values: datum.Values,
			}

		case service.DataIngestionScript:
			ingestion_script_idx := slices.IndexFunc(ingestion_scripts, func(s service.IngestionScript) bool {
				return s.Id == datum.IngestionScript
			})
			if ingestion_script_idx < 0 {
				panic(fmt.Sprintf("invalid ingestion script `%s`", datum.IngestionScript))
			}
			ingestion_script := ingestion_scripts[ingestion_script_idx]

			sources := make(map[uuid.UUID]any, len(datum.IngestionScriptSources))
			for source, name := range datum.IngestionScriptSources {
				file, err := c.FormFile(name)
				if err != nil {
					c.Logger().With(
						"error", err,
						"source", source,
						"file", name,
					).Error("could not get form file")
					return c.NoContent(http.StatusBadRequest)
				}

				sid, err := uuid.Parse(source)
				if err != nil {
					c.Logger().With(
						"error", err,
						"source", source,
					).Error("could not parse data source id")
					return c.NoContent(http.StatusBadRequest)
				}

				source_idx := slices.IndexFunc(ingestion_script.Sources, func(s service.IngestionScriptSourceRx) bool {
					return s.Id == sid
				})
				if source_idx < 0 {
					panic(fmt.Sprintf("invalid ingestion script source `%s`", sid))
				}
				source := ingestion_script.Sources[source_idx]

				f_sources, exists := sources[sid]
				switch source.Cardinality {
				case service.DataSourceCardinalityMultiple:
					if exists {
						sources[sid] = append(f_sources.([]*multipart.FileHeader), file)
					} else {
						sources[sid] = []*multipart.FileHeader{file}
					}
				case service.DataSourceCardinalitySingle:
					if exists {
						c.Logger().With(
							"script", ingestion_script.Id,
							"source", sid,
						).Warn("multiple sources assigned to cardinality single")
						return c.NoContent(http.StatusBadRequest)
					}

					sources[sid] = file
				default:
					panic(fmt.Sprintf("unexpected service.DataSourceCardinality: %#v", source.Cardinality))
				}
			}

			data[idx].Values = service.DataCreateValues{
				IngestionScript:  datum.IngestionScript,
				IngestionSources: sources,
			}
		}
	}

	dataScriptIngestion := make([]service.DataCreateValues, 0, len(info))
	for _, datum := range data {
		if datum.Values.IngestionScript != uuid.Nil {
			dataScriptIngestion = append(dataScriptIngestion, datum.Values)
		}
	}

	err = h.dataIngestValidateIngestionScriptSources(dataScriptIngestion, ingestion_scripts)
	if err != nil {
		c.Logger().With(
			"error", err,
		).Error("could not validate ingestion script sources")
		return c.NoContent(http.StatusBadRequest)
	}

	data_ids, err := h.data_service.DataCreate(data, user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"data", info,
		).Error("could not create data")
		return c.NoContent(http.StatusInternalServerError)
	}

	if project_id != uuid.Nil {
		memberships := make([]service.ProjectDataMembershipRx, len(data_ids))
		for idx, data_id := range data_ids {
			var label *string
			if project_labels[idx] != "" {
				label = &project_labels[idx]
			}
			memberships[idx] = service.ProjectDataMembershipRx{
				Project: project_id,
				Data:    data_id,
				Creator: user_id,
				Label:   label,
			}
		}
		err = h.project_service.DataMembershipsCreate(memberships)
		if err != nil {
			c.Logger().With(
				"error", err,
				"project", project_id,
				"data", data_ids,
			).Error("could not create project data memberships")
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusOK)
}

func removeFiles(paths []string, logger *slog.Logger) {
	for _, path := range paths {
		err := os.Remove(path)
		if err != nil {
			logger.With(
				"error", err,
				"file", path,
			).Error("could not remove file")
		}
	}
}

func (h *DataHandler) OrphanedData(c *echo.Context) error {
	user := c.Get(UserIdKey).(uuid.UUID)

	data, err := h.data_service.OrphanedData(user)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user,
		).Error("could not get orphaned data")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, data)
}

type DataResources struct {
	Data             service.DataRx
	DataType         service.DataType
	Properties       []service.Property
	Notes            []service.Note
	ProjectResources []service.DataProjectResources
	Users            []service.User
}

func (h *DataHandler) DataGet(c *echo.Context) error {
	user := c.Get(UserIdKey).(uuid.UUID)
	data_str := c.QueryParam("id")
	data_id, err := uuid.Parse(data_str)
	if err != nil {
		c.Logger().With(
			"user", user,
			"data", data_str,
		).Error("could not parse id as uuid")
		return c.NoContent(http.StatusBadRequest)
	}

	data_rx, err := h.data_service.DataById(data_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_id,
		).Error("could not get data")
		return c.NoContent(http.StatusInternalServerError)
	}

	user_data_permissions, err := h.data_service.DataUserPermission(user, data_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_id,
		).Error("could not get data user permission")
		return c.NoContent(http.StatusInternalServerError)
	}

	has_permission := slices.Contains(
		user_data_permissions,
		service.DataUserPermissionOwner,
	) || slices.Contains(
		user_data_permissions,
		service.DataUserPermissionRead,
	)

	if !has_permission {
		has_read_all, err := h.user_service.UserHasPermission(
			user,
			service.DbPermissionDataReadAll,
		)
		if err != nil {
			c.Logger().With(
				"error", err,
				"user", user,
			).Error("could not get user permissions")
			return c.NoContent(http.StatusInternalServerError)
		}

		has_permission = has_read_all
	}

	if !has_permission {
		c.Logger().With(
			"user", user,
			"data", data_id,
		).Debug("insufficient permission")
		return c.NoContent(http.StatusUnauthorized)
	}

	data_type, err := h.data_service.DataTypeById(data_rx.Type)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_rx,
			"user", user,
		).Error("could not get data type")
		return c.NoContent(http.StatusInternalServerError)
	}

	properties, err := h.data_service.DataProperties(data_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_id,
			"user", user,
		).Error("could not get data properties")
		return c.NoContent(http.StatusInternalServerError)
	}

	notes, err := h.data_service.DataNotes(data_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_id,
			"user", user,
		).Error("could not get data notes")
		return c.NoContent(http.StatusInternalServerError)
	}

	project_resources, err := h.data_service.DataProjectsResources(data_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_id,
			"user", user,
		).Error("could not get data project resources")
		return c.NoContent(http.StatusInternalServerError)
	}

	user_is_db_owner, err := h.user_service.IsDbOwner(user)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_id,
			"user", user,
		).Error("could not get user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !user_is_db_owner {
		project_permission_ids := make([]uuid.UUID, 0, len(project_resources))
		for _, project := range project_resources {
			if project.Project.Visibility != service.VisibilityPublic {
				project_permission_ids = append(project_permission_ids, project.Project.Id)
			}
		}

		perms, err := h.project_service.ProjectsUserPermissions(project_permission_ids, user)
		if err != nil {
			c.Logger().With(
				"error", err,
				"projects", project_permission_ids,
				"user", user,
			).Error("could not get project user permissions")
			return c.NoContent(http.StatusInternalServerError)
		}

		filtered := make([]service.DataProjectResources, 0, len(project_resources))
		for _, resources := range project_resources {
			pperms := perms[resources.Project.Id]
			if slices.Contains(pperms, service.ProjectPermissionOwner) ||
				slices.Contains(pperms, service.ProjectPermissionRead) {
				filtered = append(filtered, resources)
			}
		}
		project_resources = filtered
	}

	var user_ids []uuid.UUID
	for _, resource := range project_resources {
		if !slices.Contains(user_ids, resource.MembershipCreator) {
			user_ids = append(user_ids, resource.MembershipCreator)
		}

		if !slices.Contains(user_ids, resource.Project.Creator) {
			user_ids = append(user_ids, resource.MembershipCreator)
		}

		for _, note := range resource.Notes {
			if !slices.Contains(user_ids, note.Creator) {
				user_ids = append(user_ids, note.Creator)
			}
		}
	}
	users, err := h.user_service.UsersById(user_ids)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", data_id,
			"user", user,
		).Error("could not get users")
		return c.NoContent(http.StatusInternalServerError)
	}

	resources := DataResources{
		Data:             data_rx,
		DataType:         data_type,
		Properties:       properties,
		Notes:            notes,
		ProjectResources: project_resources,
		Users:            users,
	}
	return c.JSON(http.StatusOK, resources)
}

func (h *DataHandler) DataOrigins(c *echo.Context) error {
	origins, err := h.data_service.DataOriginsAll()
	if err != nil {
		user_id := c.Get(UserIdKey).(uuid.UUID)
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get data origins")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, origins)
}

func (h *DataHandler) DataOriginCreate(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	sufficient_permission, err := h.user_service.UserHasPermission(
		user_id,
		service.DbPermissionDataOriginCreate,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not validate user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !sufficient_permission {
		c.Logger().With(
			"user", user_id,
		).Error("insuffiecient permission to create data origin")
		return c.NoContent(http.StatusUnauthorized)
	}

	var origin service.DataOriginCreate
	err = c.Bind(&origin)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"data origin", c.Request().Body,
		).Error("invalid data origin")
		return c.NoContent(http.StatusBadRequest)
	}

	err = h.data_service.DataOriginCreate(origin)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"data origin", origin,
		).Error("could not create data origin")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *DataHandler) DataOriginGet(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	origin_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"data origin id", c.Param("id"),
		).Error("could not get data origin id")
		return c.NoContent(http.StatusBadRequest)
	}

	origins, err := h.data_service.DataOriginById(origin_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"data origin", origin_id,
		).Error("could not get data origin")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, origins)
}

func (h *DataHandler) DataOriginUpdate(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	sufficient_permission, err := h.user_service.UserHasPermission(
		user_id,
		service.DbPermissionDataOriginModify,
	)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not validate user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !sufficient_permission {
		c.Logger().With(
			"user", user_id,
		).Error("insuffiecient permission to create data origin")
		return c.NoContent(http.StatusUnauthorized)
	}

	var update service.DataOriginRx
	err = c.Bind(&update)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"update", update,
		).Error("invalid data origin update")
		return c.NoContent(http.StatusBadRequest)
	}

	err = h.data_service.DataOriginUpdate(update)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"update", update,
		).Error("could not get data origin")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
