package handler

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
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
	sufficient_permission, err := h.user_service.UserHasPermission(user_id, service.DbPermissionDataTypeCreate)
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
	sufficient_permission, err := h.user_service.UserHasPermission(user_id, service.DbPermissionDataTypeCreate)
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

	has_permission, err := h.user_service.UserHasPermission(user_id, service.DbPermissionDataSchemaCreate)
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

func (h *DataHandler) IngestionScriptsGet(c *echo.Context) error {
	data_type := c.QueryParam("data_type")
	if data_type == "" {
		scripts, err := h.data_service.IngestionScriptsGetAll()
		if err != nil {
			c.Logger().With("error", err).Error("could not get ingestion scripts")
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.JSON(http.StatusOK, scripts)
	} else {
		data_type_id, err := uuid.Parse(data_type)
		if err != nil {
			c.Logger().With(
				"error", err,
				"data type", data_type,
			).Error("could not parse data type")
			return c.NoContent(http.StatusBadRequest)
		}

		scripts, err := h.data_service.IngestionScriptsGetForDataType(data_type_id)
		if err != nil {
			c.Logger().With("error", err).Error("could not get ingestion scripts")
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.JSON(http.StatusOK, scripts)

	}
}

type IngestionScriptCreateData struct {
	Type        uuid.UUID
	Label       string
	Description string
	Cmd         string
	Args        []string
	Sources     []service.ExternalSourceCreate
}

func (h *DataHandler) IngestionScriptCreate(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	has_permission, err := h.user_service.UserHasPermission(user_id, service.DbPermissionIngestionScriptCreate)
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
		).Debug("insufficient permissions to create ingestion script")
		return c.NoContent(http.StatusUnauthorized)
	}

	var data IngestionScriptCreateData
	err = json.Unmarshal([]byte(c.FormValue("data")), &data)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", c.FormValue("data"),
		).Error("could not parse data")
		return c.NoContent(http.StatusBadRequest)
	}

	file, err := c.FormFile("script")
	if err != nil {
		c.Logger().With("user", user_id).Error("invalid ingestion script file")
		return c.NoContent(http.StatusBadRequest)
	}

	root_dir, err := h.app_service.AppDataDir(service.AppDataDirIngestionScript)
	if err != nil {
		c.Logger().Error("could not get app data dir")
		return c.NoContent(http.StatusInternalServerError)
	}

	filename := fmt.Sprintf("%s.%s", rand.Text(), file.Filename)
	cmd, err := ingestion_script_command_from_file_ext(filepath.Ext(filename))
	if err != nil {
		c.Logger().With("error", err, "file name", filename).Error("invalid filename extension")
		return c.NoContent(http.StatusBadRequest)
	}

	path := filepath.Join(root_dir, filename)
	script := service.IngestionScriptCreate{
		Type:        data.Type,
		Creator:     user_id,
		Label:       data.Label,
		Description: data.Description,
		Path:        path,
		Cmd:         cmd,
		Args:        []string{},
		Sources:     data.Sources,
	}
	h.data_service.IngestionScriptCreate(script, file)

	return nil

}

func ingestion_script_command_from_file_ext(ext string) (string, error) {
	switch ext {
	case ".py":
		return "python", nil
	default:
		return "", errors.New("unknown file type")
	}
}

func (h *DataHandler) DownloadDataValuesSingle(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	data_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"id", c.QueryParam("id"),
		).Warn("could not parse id")
		return c.NoContent(http.StatusBadRequest)
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
		// TODO: Get data label
		filename := fmt.Sprintf("%s.csv", data_id)
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

func (h *DataHandler) DownloadRawDataProject(c *echo.Context) error {
	panic("todo")
	// user_id := c.Get(UserIdKey).(uuid.UUID)
	// project_id, err := uuid.Parse(c.QueryParam("id"))
	// if err != nil {
	// 	c.Logger().With(
	// 		"error", err,
	// 		"id", c.QueryParam("id"),
	// 	).Error("could not parse id")
	// 	return c.NoContent(http.StatusBadRequest)
	// }
	// hierarchy, err := service.ParseSaveDataHierarchy(c.QueryParam("hierarchy"))
	// if err != nil {
	// 	c.Logger().With(
	// 		"error", err,
	// 		"hierarchy", c.QueryParam("hierarchy"),
	// 	).Error("invalid hierarchy parameter")
	// 	return c.NoContent(http.StatusBadRequest)
	// }

	// sample_data, err := h.data_service.ProjectRawDataAll(project_id)
	// if err != nil {
	// 	c.Logger().With(
	// 		"error", err,
	// 		"project", project_id,
	// 	).Error("could not get project sample data")
	// 	return c.NoContent(http.StatusInternalServerError)
	// }
	// if len(sample_data) == 0 {
	// 	c.Logger().With(
	// 		"project", project_id,
	// 	).Debug("no sample data found for project")
	// 	return c.NoContent(http.StatusNoContent)
	// }

	// sample_data_ids := make([]uuid.UUID, len(sample_data))
	// for idx, sample_data := range sample_data {
	// 	sample_data_ids[idx] = sample_data.Id
	// }

	// permissions, err := h.data_service.GetSampleDataUserPermission(sample_data_ids, user_id)
	// if err != nil {
	// 	c.Logger().With(
	// 		"error", err,
	// 		"sample data", sample_data_ids,
	// 		"user", user_id,
	// 	).Error("could not get sample data user permissions")
	// 	return c.NoContent(http.StatusInternalServerError)
	// }
	// if len(sample_data_ids) != len(permissions) {
	// 	c.Logger().With(
	// 		"sample data", sample_data_ids,
	// 		"permissions", permissions,
	// 	).Error("invalid sample data user permissions")
	// 	panic("invalid sample data user permissions")
	// }

	// sufficient_permissions := make([]uuid.UUID, 0, len(sample_data_ids))
	// for _, user_permissions := range permissions {
	// 	if !slices.Contains(sample_data_ids, user_permissions.SampleData) {
	// 		c.Logger().With(
	// 			"user permissions", user_permissions.SampleData,
	// 			"sample data", sample_data_ids,
	// 		).Error("invalid smaple data user permissions")
	// 		panic("invalid sample data user permissions")
	// 	}

	// 	if len(user_permissions.Permissions) > 0 {
	// 		sufficient_permissions = append(sufficient_permissions, user_permissions.SampleData)
	// 	}
	// }
	// if len(sufficient_permissions) == 0 {
	// 	return c.NoContent(http.StatusUnauthorized)
	// }

	// var sample_ids []uuid.UUID
	// var data_schema_ids []uuid.UUID
	// for _, sample_data_id := range sufficient_permissions {
	// 	raw_data_idx := slices.IndexFunc(sample_data, func(sample_data service.SampleData) bool {
	// 		return sample_data.Id == sample_data_id
	// 	})
	// 	if raw_data_idx < 0 {
	// 		c.Logger().With(
	// 			"sample data", sample_data_id,
	// 			"all sample data", sample_data,
	// 		).Error("invalid sample data")
	// 		panic("invalid sample data")
	// 	}
	// 	data := sample_data[raw_data_idx]

	// 	if !slices.Contains(data_schema_ids, data.Schema) {
	// 		data_schema_ids = append(data_schema_ids, data.Schema)
	// 	}

	// 	if !slices.Contains(sample_ids, data.Sample) {
	// 		sample_ids = append(sample_ids, data.Sample)
	// 	}
	// }

	// samples, err := h.project_service.GetProjectSampleMembershipsByProject(project_id)
	// if err != nil {
	// 	return c.NoContent(http.StatusInternalServerError)
	// }

	// data_schemas, err := h.data_service.GetDataSchemasById(data_schema_ids)
	// if err != nil {
	// 	c.Logger().With(
	// 		"error", err,
	// 		"data schemas", data_schema_ids,
	// 	).Error("could not get data schemas")
	// 	return c.NoContent(http.StatusInternalServerError)
	// }

	// sample_data_stored, err := h.data_service.GetSampleDataStoredById(sufficient_permissions)
	// if err != nil {
	// 	c.Logger().With(
	// 		"error", err,
	// 		"sample data", sufficient_permissions,
	// 	).Error("could not get stored sample data")
	// 	return c.NoContent(http.StatusInternalServerError)
	// }

	// tmpfile, err := os.CreateTemp("", "")
	// if err != nil {
	// 	c.Logger().With(
	// 		"error", err,
	// 	).Error("could not create temporary data file")
	// 	return c.NoContent(http.StatusInternalServerError)
	// }
	// defer tmpfile.Close()

	// archive := zip.NewWriter(tmpfile)
	// for _, stored_data := range sample_data_stored {
	// 	sample_data_idx := slices.IndexFunc(sample_data, func(data service.SampleData) bool {
	// 		return stored_data.SampleData == data.Id
	// 	})
	// 	if sample_data_idx < 0 {
	// 		c.Logger().With(
	// 			"sample data", stored_data.SampleData,
	// 			"sample data all", sample_data,
	// 		).Error("invalid sample data")
	// 		panic("invalid sample data")
	// 	}
	// 	data_info := sample_data[sample_data_idx]

	// 	data_schema_idx := slices.IndexFunc(data_schemas, func(schema service.DataSchema) bool {
	// 		return data_info.Schema == schema.Id
	// 	})
	// 	if sample_data_idx < 0 {
	// 		c.Logger().With(
	// 			"data schema", data_info.Schema,
	// 			"data schema all", data_schemas,
	// 		).Error("invalid data schema")
	// 		panic("invalid data schema")
	// 	}
	// 	schema := data_schemas[data_schema_idx]

	// 	sample_idx := slices.IndexFunc(samples, func(sample service.ProjectSampleMembership) bool {
	// 		return data_info.Sample == sample.Sample
	// 	})
	// 	if sample_idx < 0 {
	// 		c.Logger().With(
	// 			"sample", data_info.Sample,
	// 			"samples", samples,
	// 		).Error("invalid sample")
	// 		panic("invalid sample")
	// 	}
	// 	sample := samples[sample_idx]

	// 	var filename strings.Builder
	// 	switch hierarchy {
	// 	case service.SaveDataHierarchyFlat:
	// 		fmt.Fprintf(&filename, "%s.%s.", schema.Label, sample.Label)
	// 		if data_info.Label != nil && *data_info.Label != "" {
	// 			fmt.Fprintf(&filename, "%s.", *data_info.Label)
	// 		}
	// 	case service.SaveDataHierarchyDataSchema:
	// 		panic("todo")
	// 	case service.SaveDataHierarchyDataSchemaSample:
	// 		panic("todo")
	// 	case service.SaveDataHierarchySample:
	// 		panic("todo")
	// 	case service.SaveDataHierarchySampleDataSchema:
	// 		panic("todo")
	// 	default:
	// 		panic(fmt.Sprintf("unexpected service.SaveDataHierarchy: %#v", hierarchy))
	// 	}

	// 	if stored_data.Storage == service.DataStorageInternal {
	// 		filename.WriteString("csv")
	// 	}
	// 	file, err := archive.Create(filename.String())
	// 	if err != nil {
	// 		c.Logger().With(
	// 			"error", err,
	// 			"sample data", stored_data.SampleData,
	// 			"file name", filename,
	// 		).Error("could not create file in archive")
	// 		return c.NoContent(http.StatusInternalServerError)
	// 	}

	// 	switch stored_data.Storage {
	// 	case service.DataStorageExternal:
	// 		panic("todo")
	// 	case service.DataStorageInternal:
	// 		data, err := h.data_service.StoredDataToCsv(stored_data.Data.([]service.ColumnData))
	// 		if err != nil {
	// 			c.Logger().With(
	// 				"error", err,
	// 				"stored data", stored_data,
	// 			).Error("could not write data to csv")
	// 			return c.NoContent(http.StatusInternalServerError)
	// 		}
	// 		_, err = file.Write(data)
	// 		if err != nil {
	// 			c.Logger().With(
	// 				"error", err,
	// 				"stored data", stored_data,
	// 			).Error("could not write data to archive file")
	// 			return c.NoContent(http.StatusInternalServerError)
	// 		}
	// 	default:
	// 		panic(fmt.Sprintf("unexpected service.DataStorage: %#v", stored_data.Storage))
	// 	}
	// }

	// err = archive.Close()
	// if err != nil {
	// 	c.Logger().With("error", err).Error("could not close archive")
	// 	return c.NoContent(http.StatusInternalServerError)
	// }

	// return c.Attachment(tmpfile.Name(), "data.zip")
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
	has_permission, err := h.user_service.UserHasPermission(user_id, service.DbPermissionTransformCreate)
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

type DataCreate struct {
	Type                   uuid.UUID
	Creator                uuid.UUID
	Origin                 uuid.UUID
	Timestamp              time.Time
	Visibility             service.Visibility
	Properties             []service.Property
	Notes                  []service.Note
	IngestionMethod        service.DataIngestionMethod
	Values                 map[string]any
	IngestionScript        uuid.UUID
	IngestionScriptSources map[string]string
}

func (h *DataHandler) DataCreate(c *echo.Context) error {
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
		).Error("insufficient permission to create data")
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
			).Error("could not parse project id")
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
			).Error("insufficient permission to create project data")
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

	var info []DataCreate
	err = json.Unmarshal([]byte(c.FormValue("data")), &info)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data", c.FormValue("data"),
		).Error("could not parse data")
		return c.NoContent(http.StatusBadRequest)
	}

	data := make([]service.DataCreate, len(info))
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
		data[idx].IngestionMethod = datum.IngestionMethod

		switch datum.IngestionMethod {
		case service.DataIngestionManual:
			data[idx].Values = datum.Values

		case service.DataIngestionScript:
			data[idx].IngestionScript = datum.IngestionScript
			data[idx].IngestionScriptSources = make(map[uuid.UUID][]*multipart.FileHeader, len(datum.IngestionScriptSources))
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

				data[idx].IngestionScriptSources[sid] = []*multipart.FileHeader{file}
			}
		}
	}

	dataScriptIngestion := make([]service.DataCreate, 0, len(data))
	for _, datum := range data {
		if datum.IngestionMethod == service.DataIngestionScript {
			dataScriptIngestion = append(dataScriptIngestion, datum)
		}
	}
	err = h.dataCreateValidateIngestionScriptSources(dataScriptIngestion)
	if err != nil {
		c.Logger().With(
			"error", err,
		).Error("could not get data ingestion script")
		return c.NoContent(http.StatusInternalServerError)
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
		var project_labels []string
		err = json.Unmarshal([]byte(c.FormValue("project_labels")), &project_labels)
		if err != nil {
			c.Logger().With(
				"error", err,
				"data", c.FormValue("data"),
			).Error("could not parse data")
			return c.NoContent(http.StatusBadRequest)
		}

		memberships := make([]service.ProjectDataMembershipRx, len(data_ids))
		for idx, data_id := range data_ids {
			memberships[idx] = service.ProjectDataMembershipRx{
				Project: project_id,
				Data:    data_id,
				Creator: user_id,
				Label:   nil,
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

func (h *DataHandler) dataCreateValidateIngestionScriptSources(data []service.DataCreate) error {
	ingestion_scripts := []service.IngestionScript{}
	for _, datum := range data {
		script_idx := slices.IndexFunc(ingestion_scripts, func(script service.IngestionScript) bool {
			return script.Id == datum.IngestionScript
		})
		if script_idx > -1 {
			continue
		}

		script, err := h.data_service.IngestionScriptGet(datum.IngestionScript)
		if err != nil {
			return fmt.Errorf("could not get ingestion script %s: %w", datum.IngestionScript, err)
		}

		ingestion_scripts = append(ingestion_scripts, script)
	}

	for _, datum := range data {
		script_idx := slices.IndexFunc(ingestion_scripts, func(script service.IngestionScript) bool {
			return script.Id == datum.IngestionScript
		})
		script := ingestion_scripts[script_idx]

		for _, src := range script.Sources {
			files, exists := datum.IngestionScriptSources[src.Id]
			if src.Required {
				if !exists {
					return errors.New("data is missing required ingestion script source")
				}
			}
			if src.Cardinality == service.DataSourceCardinalitySingle {
				if len(files) > 1 {
					return errors.New("data has multiple sources for single cardinality source")
				}
			}
		}
	}

	return nil
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

	has_permission := slices.Contains(user_data_permissions, service.DataUserPermissionOwner) ||
		slices.Contains(user_data_permissions, service.DataUserPermissionRead)

	if !has_permission {
		has_read_all, err := h.user_service.UserHasPermission(user, service.DbPermissionDataReadAll)
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
		ProjectResources: project_resources,
		Users:            users,
	}
	return c.JSON(http.StatusOK, resources)
}
