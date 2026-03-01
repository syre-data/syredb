package handler

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
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
	user_service    *service.UserService
	project_service *service.ProjectService
}

func NewDataHandler(
	db *database.DBConnection,
	data_service *service.DataService,
	user_service *service.UserService,
	project_service *service.ProjectService,
) *DataHandler {
	return &DataHandler{
		db:              db,
		data_service:    data_service,
		user_service:    user_service,
		project_service: project_service,
	}
}

func (h *DataHandler) GetDataSchemasAll(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	kind := "all"
	if c.QueryParams().Has("type") {
		kind = c.QueryParam("type")
	}

	switch kind {
	case "all":
		schemas, err := h.data_service.GetDataSchemasAll(user_id)
		if err != nil {
			c.Logger().With(
				"error", err,
				"user", user_id,
			).Error("could not get user data schemas")
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.JSON(http.StatusOK, schemas)
	case "raw":
		schemas, err := h.data_service.GetDataSchemasRaw(user_id)
		if err != nil {
			c.Logger().With(
				"error", err,
				"user", user_id,
			).Error("could not get user raw data schemas")
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.JSON(http.StatusOK, schemas)
	case "transform":
		schemas, err := h.data_service.GetDataSchemasTransform(user_id)
		if err != nil {
			c.Logger().With(
				"error", err,
				"user", user_id,
			).Error("could not get user transform data schemas")
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.JSON(http.StatusOK, schemas)
	default:
		c.Logger().With(
			"kind", kind,
			"user", user_id,
		).Error("invalid data schema type")
		return c.NoContent(http.StatusBadRequest)
	}
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

	if data_schema.Storage == service.DataStorageInternal {
		if len(data_schema.Schema) == 0 {
			c.Logger().With(
				"error", "internal storage data schema must have at least one column",
				"user", user_id,
			).Debug("invalid data schema")
			return c.NoContent(http.StatusBadRequest)
		}
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

func (h *DataHandler) GetDataSchemaResources(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	data_schema_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"data schema id", c.QueryParam("id"),
		).Error("could not parse data schema id")
		return c.NoContent(http.StatusBadRequest)
	}

	resources, err := h.data_service.GetDataSchemaResources(data_schema_id)
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

func (h *DataHandler) DownloadSampleDataSingle(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	sample_data_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"id", c.QueryParam("id"),
		).Error("could not parse id")
		return c.NoContent(http.StatusBadRequest)
	}

	sample_data, err := h.data_service.GetSampleData(sample_data_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.NoContent(http.StatusNotFound)
		}
		return c.NoContent(http.StatusInternalServerError)
	}

	if sample_data.Visibility != service.VisibilityPublic {
		permissions, err := h.data_service.GetSampleDataUserPermission(
			[]uuid.UUID{sample_data_id},
			user_id,
		)
		if err != nil {
			c.Logger().With(
				"error", err,
				"sample data", sample_data_id,
				"user", user_id,
			).Error("could not get sample data user permissions")
			return c.NoContent(http.StatusInternalServerError)
		}
		if permissions[0].SampleData != sample_data_id {
			c.Logger().With(
				"user", user_id,
				"sample data", sample_data_id,
				"permissions", permissions,
			).Error("invalid sample data user permissions")
			panic("invalid sample data user permissions")
		}
		user_permissions := permissions[0].Permissions
		if len(user_permissions) == 0 {
			c.Logger().With(
				"sample data", sample_data_id,
				"user", user_id,
			).Error("insufficient permissions")
			return c.NoContent(http.StatusUnauthorized)
		}
	}

	datas, err := h.data_service.GetSampleDataStoredById([]uuid.UUID{sample_data_id})
	if err != nil {
		c.Logger().With(
			"error", err,
			"sample data", sample_data_id,
		).Error("could not get sample data stored data")
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(datas) != 1 {
		c.Logger().With(
			"sample data", sample_data_id,
		).Error("invalid sample data storage")
		return c.NoContent(http.StatusInternalServerError)
	}
	stored := datas[0]

	var path string
	var filename string
	switch stored.Storage {
	case service.DataStorageExternal:
		info := stored.Data.(service.SampleDataPayloadExternal)
		path = info.Path
		filename = info.Filename
	case service.DataStorageInternal:
		cols := stored.Data.([]service.ColumnData)
		name := fmt.Sprintf("%s.*.csv", sample_data_id)
		tmpfile, err := os.CreateTemp("", name)
		if err != nil {
			c.Logger().With(
				"error", err,
			).Error("could not create temporary data file")
			return c.NoContent(http.StatusInternalServerError)
		}
		defer tmpfile.Close()
		path = tmpfile.Name()

		writer := csv.NewWriter(tmpfile)
		record := make([]string, len(cols))
		for idx := range len(cols[0].Values) {
			for cidx := range len(cols) {
				col := cols[cidx]
				value := col.Values[idx]
				var value_str string
				switch col.DType {
				case service.DataTypeBoolean:
					if value.(bool) {
						value_str = "true"
					} else {
						value_str = "false"
					}
				case service.DataTypeFloat:
					value_str = fmt.Sprint(value.(float64))
				case service.DataTypeInt:
					value_str = fmt.Sprint(value.(int64))
				case service.DataTypeString:
					value_str = value.(string)
				case service.DataTypeTimestamp:
					value_str = value.(time.Time).String()
				case service.DataTypeUint:
					value_str = fmt.Sprint(value.(uint64))
				default:
					panic(fmt.Sprintf("unexpected service.DataType: %#v", col.DType))
				}
				record[cidx] = value_str
			}
			err = writer.Write(record)
			if err != nil {
				c.Logger().With(
					"error", err,
				).Error("could not write sample data to file")
			}
		}
		writer.Flush()

		schema, err := h.data_service.GetDataSchemasById([]uuid.UUID{sample_data.Schema})
		if err != nil {
			c.Logger().With(
				"error", err,
				"schema", sample_data.Schema,
			).Error("could not get data schema")

			filename = fmt.Sprintf("%s.csv", sample_data_id)
		} else {
			filename = fmt.Sprintf("%s.%s.csv", schema[0].Label, sample_data_id)
		}
	}

	return c.Attachment(path, filename)
}

func (h *DataHandler) DownloadSampleDataProject(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	project_id, err := uuid.Parse(c.QueryParam("id"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"id", c.QueryParam("id"),
		).Error("could not parse id")
		return c.NoContent(http.StatusBadRequest)
	}
	hierarchy, err := service.ParseSaveDataHierarchy(c.QueryParam("hierarchy"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"hierarchy", c.QueryParam("hierarchy"),
		).Error("invalid hierarchy parameter")
		return c.NoContent(http.StatusBadRequest)
	}

	sample_data, err := h.data_service.GetProjectSampleData(project_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"project", project_id,
		).Error("could not get project sample data")
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(sample_data) == 0 {
		c.Logger().With(
			"project", project_id,
		).Debug("no sample data found for project")
		return c.NoContent(http.StatusNoContent)
	}

	sample_data_ids := make([]uuid.UUID, len(sample_data))
	for idx, sample_data := range sample_data {
		sample_data_ids[idx] = sample_data.Id
	}

	permissions, err := h.data_service.GetSampleDataUserPermission(sample_data_ids, user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"sample data", sample_data_ids,
			"user", user_id,
		).Error("could not get sample data user permissions")
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(sample_data_ids) != len(permissions) {
		c.Logger().With(
			"sample data", sample_data_ids,
			"permissions", permissions,
		).Error("invalid sample data user permissions")
		panic("invalid sample data user permissions")
	}

	sufficient_permissions := make([]uuid.UUID, 0, len(sample_data_ids))
	for _, user_permissions := range permissions {
		if !slices.Contains(sample_data_ids, user_permissions.SampleData) {
			c.Logger().With(
				"user permissions", user_permissions.SampleData,
				"sample data", sample_data_ids,
			).Error("invalid smaple data user permissions")
			panic("invalid sample data user permissions")
		}

		if len(user_permissions.Permissions) > 0 {
			sufficient_permissions = append(sufficient_permissions, user_permissions.SampleData)
		}
	}
	if len(sufficient_permissions) == 0 {
		return c.NoContent(http.StatusUnauthorized)
	}

	var sample_ids []uuid.UUID
	var data_schema_ids []uuid.UUID
	for _, sample_data_id := range sufficient_permissions {
		sample_data_idx := slices.IndexFunc(sample_data, func(sample_data service.SampleData) bool {
			return sample_data.Id == sample_data_id
		})
		if sample_data_idx < 0 {
			c.Logger().With(
				"sample data", sample_data_id,
				"all sample data", sample_data,
			).Error("invalid sample data")
			panic("invalid sample data")
		}
		data := sample_data[sample_data_idx]

		if !slices.Contains(data_schema_ids, data.Schema) {
			data_schema_ids = append(data_schema_ids, data.Schema)
		}

		if !slices.Contains(sample_ids, data.Sample) {
			sample_ids = append(sample_ids, data.Sample)
		}
	}

	samples, err := h.project_service.GetProjectSampleMembershipsByProject(project_id)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	data_schemas, err := h.data_service.GetDataSchemasById(data_schema_ids)
	if err != nil {
		c.Logger().With(
			"error", err,
			"data schemas", data_schema_ids,
		).Error("could not get data schemas")
		return c.NoContent(http.StatusInternalServerError)
	}

	sample_data_stored, err := h.data_service.GetSampleDataStoredById(sufficient_permissions)
	if err != nil {
		c.Logger().With(
			"error", err,
			"sample data", sufficient_permissions,
		).Error("could not get stored sample data")
		return c.NoContent(http.StatusInternalServerError)
	}

	tmpfile, err := os.CreateTemp("", "")
	if err != nil {
		c.Logger().With(
			"error", err,
		).Error("could not create temporary data file")
		return c.NoContent(http.StatusInternalServerError)
	}
	defer tmpfile.Close()

	archive := zip.NewWriter(tmpfile)
	for _, stored_data := range sample_data_stored {
		sample_data_idx := slices.IndexFunc(sample_data, func(data service.SampleData) bool {
			return stored_data.SampleData == data.Id
		})
		if sample_data_idx < 0 {
			c.Logger().With(
				"sample data", stored_data.SampleData,
				"sample data all", sample_data,
			).Error("invalid sample data")
			panic("invalid sample data")
		}
		data_info := sample_data[sample_data_idx]

		data_schema_idx := slices.IndexFunc(data_schemas, func(schema service.DataSchema) bool {
			return data_info.Schema == schema.Id
		})
		if sample_data_idx < 0 {
			c.Logger().With(
				"data schema", data_info.Schema,
				"data schema all", data_schemas,
			).Error("invalid data schema")
			panic("invalid data schema")
		}
		schema := data_schemas[data_schema_idx]

		sample_idx := slices.IndexFunc(samples, func(sample service.ProjectSampleMembership) bool {
			return data_info.Sample == sample.Sample
		})
		if sample_idx < 0 {
			c.Logger().With(
				"sample", data_info.Sample,
				"samples", samples,
			).Error("invalid sample")
			panic("invalid sample")
		}
		sample := samples[sample_idx]

		var filename strings.Builder
		switch hierarchy {
		case service.SaveDataHierarchyFlat:
			fmt.Fprintf(&filename, "%s.%s.", schema.Label, sample.Label)
			if data_info.Label != nil && *data_info.Label != "" {
				fmt.Fprintf(&filename, "%s.", *data_info.Label)
			}
		case service.SaveDataHierarchyDataSchema:
			panic("todo")
		case service.SaveDataHierarchyDataSchemaSample:
			panic("todo")
		case service.SaveDataHierarchySample:
			panic("todo")
		case service.SaveDataHierarchySampleDataSchema:
			panic("todo")
		default:
			panic(fmt.Sprintf("unexpected service.SaveDataHierarchy: %#v", hierarchy))
		}

		if stored_data.Storage == service.DataStorageInternal {
			filename.WriteString("csv")
		}
		file, err := archive.Create(filename.String())
		if err != nil {
			c.Logger().With(
				"error", err,
				"sample data", stored_data.SampleData,
				"file name", filename,
			).Error("could not create file in archive")
			return c.NoContent(http.StatusInternalServerError)
		}

		switch stored_data.Storage {
		case service.DataStorageExternal:
			panic("todo")
		case service.DataStorageInternal:
			data, err := h.data_service.StoredDataToCsv(stored_data.Data.([]service.ColumnData))
			if err != nil {
				c.Logger().With(
					"error", err,
					"stored data", stored_data,
				).Error("could not write data to csv")
				return c.NoContent(http.StatusInternalServerError)
			}
			_, err = file.Write(data)
			if err != nil {
				c.Logger().With(
					"error", err,
					"stored data", stored_data,
				).Error("could not write data to archive file")
				return c.NoContent(http.StatusInternalServerError)
			}
		default:
			panic(fmt.Sprintf("unexpected service.DataStorage: %#v", stored_data.Storage))
		}
	}

	err = archive.Close()
	if err != nil {
		c.Logger().With("error", err).Error("could not close archive")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.Attachment(tmpfile.Name(), "data.zip")
}

func (h *DataHandler) CreateTransform(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	user_role, err := h.user_service.UserRole(user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user role")
		return c.NoContent(http.StatusInternalServerError)
	}
	if user_role != service.UserRoleAdmin &&
		user_role != service.UserRoleOwner {
		c.Logger().With(
			"user", user_id,
		).Debug("insufficient permissions to create transform")
		return c.NoContent(http.StatusUnauthorized)
	}

	var transform service.TransformCreate
	transform.Input, err = uuid.Parse(c.FormValue("input"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"input", c.FormValue("input"),
		).Error("could not parse transform input")
		return c.NoContent(http.StatusBadRequest)
	}
	transform.Output, err = uuid.Parse(c.FormValue("output"))
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"output", c.FormValue("output"),
		).Error("could not parse transform output")
		return c.NoContent(http.StatusBadRequest)
	}
	transform.Script, err = c.FormFile("script")
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"script", c.FormValue("script"),
		).Error("could not parse transform script")
		return c.NoContent(http.StatusBadRequest)
	}
	transform.Label = c.FormValue("label")
	transform.Description = c.FormValue("description")

	_, err = h.data_service.CreateTransform(user_id, transform)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not create transform")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
