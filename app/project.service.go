package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	PROJECT_USER_PERMISSION_READ       = ProjectUserPermission("read")
	PROJECT_USER_PERMISSION_READ_WRITE = ProjectUserPermission("read_write")
	PROJECT_USER_PERMISSION_ADMIN      = ProjectUserPermission("admin")
	PROJECT_USER_PERMISSION_OWNER      = ProjectUserPermission("owner")
)

type ProjectUserPermission string

const (
	PROJECT_PUBLIC  = ProjectVisibility("public")
	PROJECT_PRIVATE = ProjectVisibility("private")
)

type ProjectVisibility string

type ProjectService struct {
	ctx          context.Context
	logger       *slog.Logger
	db           *DbConnection
	app_state    *AppState
	user_service *UserService
	data_service *DataService
}

func NewProjectService(
	logger *slog.Logger,
	db *DbConnection,
	app_state *AppState,
	user_service *UserService,
	data_service *DataService,
) *ProjectService {
	return &ProjectService{
		logger:       logger,
		db:           db,
		app_state:    app_state,
		user_service: user_service,
		data_service: data_service,
	}
}

func (s *ProjectService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

func (s *ProjectService) user_id() uuid.UUID {
	s.app_state._lock.RLock()
	defer s.app_state._lock.RUnlock()
	return s.app_state.user_id
}

type Project struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Label       string
	Description string
	Visibility  ProjectVisibility
}

func (s *ProjectService) GetUserProjects() ([]Project, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return nil, &UserNotAuthenticatedError{}
	}

	user_project_query := "SELECT _id, _creator, label, description, visibility FROM project_ WHERE _creator=$1 ORDER BY _id"
	rows, _ := s.db.conn.Query(s.ctx, user_project_query, user_id)
	user_projects, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Project, error) {
		var project Project
		err := row.Scan(&project.Id, &project.Creator, &project.Label, &project.Description, &project.Visibility)
		return project, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not collect user projects")
		return nil, err
	}

	return user_projects, nil
}

type ProjectCreate struct {
	Label       string
	Description string
	Visibility  ProjectVisibility
}

func (s *ProjectService) CreateProject(project ProjectCreate) (uuid.UUID, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return uuid.Nil, &UserNotAuthenticatedError{}
	}

	tx, err := s.db.conn.Begin(s.ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(s.ctx)

	var project_id uuid.UUID
	create_project_query :=
		`INSERT INTO project_ (_creator, label, description, visibility) 
	VALUES ($1, $2, $3, $4) 
	RETURNING _id`
	err = tx.QueryRow(s.ctx, create_project_query, user_id, project.Label, project.Description, project.Visibility).Scan(&project_id)
	if err != nil {
		s.logger.With("error", err).Error("could not create project")
		return uuid.Nil, err
	}

	set_user_permission_query :=
		`INSERT INTO project_user_permission_ (_project, _user, permission) 
	VALUES ($1, $2, $3)`
	_, err = tx.Exec(s.ctx, set_user_permission_query, project_id, user_id, PROJECT_USER_PERMISSION_OWNER)
	if err != nil {
		s.logger.With("error", err).Error("could not create user project permission")
		return uuid.Nil, err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not create project")
		return uuid.Nil, err
	}

	return project_id, nil
}

type PropertyType string

const (
	PROPERTY_TYPE_STRING    = PropertyType("string")
	PROPERTY_TYPE_BOOL      = PropertyType("bool")
	PROPERTY_TYPE_UINT      = PropertyType("uint")
	PROPERTY_TYPE_INT       = PropertyType("int")
	PROPERTY_TYPE_FLOAT     = PropertyType("float")
	PROPERTY_TYPE_QUANTITY  = PropertyType("quantity")
	PROPERTY_TYPE_TIMESTAMP = PropertyType("timestamp")
)

type Property struct {
	Key   string
	Type  PropertyType
	Value any // TODO: Match value with type
}

type ProjectSampleNote struct {
	Id        uuid.UUID
	Sample    uuid.UUID
	Project   uuid.UUID
	Creator   uuid.UUID
	Timestamp time.Time
	Content   string
}

type ProjectSample struct {
	Id                uuid.UUID
	Creator           uuid.UUID
	MembershipCreator uuid.UUID
	MembershipCreated time.Time
	Label             string
	Tags              []string
	Properties        []Property
	NoteCount         uint
}

type SampleData struct {
	Id        uuid.UUID
	Sample    uuid.UUID
	Schema    uuid.UUID
	Creator   uuid.UUID
	Timestamp time.Time
}

type ProjectSampleGroup struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Label       string
	Description string
	Properties  []Property
	Samples     []uuid.UUID
}

type SampleGroupRelation struct {
	Parent uuid.UUID
	Child  uuid.UUID
}

type ProjectResources struct {
	Project               Project
	ProjectTags           []string
	Samples               []ProjectSample
	SampleData            []SampleData
	DataSchemas           []DataSchema
	SampleGroups          []ProjectSampleGroup
	SampleGroupRelations  []SampleGroupRelation
	ProjectNoteCount      uint
	ProjectUserPermission ProjectUserPermission
}

func (s *ProjectService) GetProjectResources(project_id uuid.UUID) (ProjectResources, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return ProjectResources{}, &UserNotAuthenticatedError{}
	}

	var project_resources ProjectResources
	project_query := "SELECT _id, _creator, label, description, visibility FROM project_ WHERE _id=$1"
	err := s.db.conn.QueryRow(
		s.ctx,
		project_query,
		project_id,
	).Scan(
		&project_resources.Project.Id,
		&project_resources.Project.Creator,
		&project_resources.Project.Label,
		&project_resources.Project.Description,
		&project_resources.Project.Visibility,
	)
	if err != nil {
		s.logger.With("error", err).Error("could not get project")
		return ProjectResources{}, err
	}

	project_user_permission_query := "SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	err = s.db.conn.QueryRow(s.ctx, project_user_permission_query, project_id, user_id).Scan(&project_resources.ProjectUserPermission)
	if err != nil {
		s.logger.With("error", err).Error("could not get project user permission")
		return ProjectResources{}, err
	}

	project_tags_query := "SELECT _tag FROM project_tag_ WHERE _project=$1"
	project_tag_rows, _ := s.db.conn.Query(s.ctx, project_tags_query, project_id)
	project_resources.ProjectTags, err = pgx.CollectRows(project_tag_rows, func(row pgx.CollectableRow) (string, error) {
		var tag string
		err := row.Scan(&tag)
		return tag, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not get project tags")
	}

	project_note_count_query := "SELECT COUNT(*) FROM project_note_ WHERE _project=$1"
	err = s.db.conn.QueryRow(s.ctx, project_note_count_query, project_id).Scan(&project_resources.ProjectNoteCount)
	if err != nil {
		s.logger.With("error", err).Error("could not get project note count")
	}

	project_sample_membership_query := `
		SELECT _sample, _creator, _timestamp, label 
		FROM project_sample_membership_ 
		WHERE _project=$1
		ORDER BY label
	`
	project_sample_membership_rows, _ := s.db.conn.Query(
		s.ctx,
		project_sample_membership_query,
		project_id,
	)
	project_resources.Samples, err = pgx.CollectRows(project_sample_membership_rows, func(row pgx.CollectableRow) (ProjectSample, error) {
		var sample ProjectSample
		err := row.Scan(
			&sample.Id,
			&sample.MembershipCreator,
			&sample.MembershipCreated,
			&sample.Label,
		)
		return sample, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not get project sample memberships")
		project_resources.Samples = []ProjectSample{}
	}

	var samples_err error
	sample_tags_query := "SELECT _tag FROM project_sample_tag_ WHERE _project=$1 AND _sample=$2"
	sample_properties_query := "SELECT _key, _type, value FROM sample_property_ WHERE _sample=$1"
	sample_note_count_query := "SELECT COUNT(*) FROM project_sample_note_ WHERE _sample=$1"
	for i := range project_resources.Samples {
		sample_id := project_resources.Samples[i].Id
		sample_tag_rows, _ := s.db.conn.Query(s.ctx, sample_tags_query, project_id, sample_id)
		project_resources.Samples[i].Tags, err = pgx.CollectRows(sample_tag_rows, func(row pgx.CollectableRow) (string, error) {
			var tag string
			err := row.Scan(&tag)
			return tag, err
		})
		if err != nil {
			s.logger.With("project", project_id, "sample", sample_id, "error", err).Error("could not get project sample tags in project")
			samples_err = err
		}

		sample_properties_rows, _ := s.db.conn.Query(s.ctx, sample_properties_query, sample_id)
		project_resources.Samples[i].Properties, err = pgx.CollectRows(sample_properties_rows, func(row pgx.CollectableRow) (Property, error) {
			var property Property
			err := row.Scan(&property.Key, &property.Type, &property.Value)
			return property, err
		})
		if err != nil {
			s.logger.With("sample", sample_id, "error", err).Error("could not get sample properties")
		}

		err = s.db.conn.QueryRow(s.ctx, sample_note_count_query, sample_id).Scan(&project_resources.Samples[i].NoteCount)
		if err != nil {
			s.logger.With("sample", sample_id, "error", err).Error("could not get sample properties")
		}

	}
	if samples_err != nil {
		return ProjectResources{}, samples_err
	}

	sample_ids := make([]string, len(project_resources.Samples))
	for idx, sample := range project_resources.Samples {
		sample_ids[idx] = fmt.Sprintf("'%s'", sample.Id)
	}
	sample_data_query := fmt.Sprintf(
		"SELECT _id, _sample, _schema, _creator, timestamp FROM sample_data_ WHERE _sample IN (%s)",
		strings.Join(sample_ids, ", "),
	)
	sample_data_rows, _ := s.db.conn.Query(s.ctx, sample_data_query)
	project_resources.SampleData, err = pgx.CollectRows(sample_data_rows, func(row pgx.CollectableRow) (SampleData, error) {
		var sample_data SampleData
		err := row.Scan(&sample_data.Id, &sample_data.Sample, &sample_data.Schema, &sample_data.Creator, &sample_data.Timestamp)
		return sample_data, err
	})
	if err != nil {
		s.logger.With("error", err, "query", sample_data_query).Error("could not get sample data")
		return ProjectResources{}, err
	}

	data_schema_ids := []uuid.UUID{}
	for _, sample_data := range project_resources.SampleData {
		if !slices.Contains(data_schema_ids, sample_data.Schema) {
			data_schema_ids = append(data_schema_ids, sample_data.Schema)
		}
	}
	project_resources.DataSchemas, err = s.data_service.GetDataSchemasById(data_schema_ids)
	if err != nil {
		return ProjectResources{}, err
	}

	return project_resources, nil
}

type ProjectWithUserPermission struct {
	Id             uuid.UUID
	Creator        uuid.UUID
	Label          string
	Description    string
	Visibility     ProjectVisibility
	UserPermission ProjectUserPermission
}

func (s *ProjectService) GetProjectWithUserPermission(project_id uuid.UUID) (ProjectWithUserPermission, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return ProjectWithUserPermission{}, &UserNotAuthenticatedError{}
	}

	var project ProjectWithUserPermission
	project_query := "SELECT _id, _creator, label, description, visibility FROM project_ WHERE _id=$1"
	err := s.db.conn.QueryRow(
		s.ctx,
		project_query,
		project_id,
	).Scan(
		&project.Id,
		&project.Creator,
		&project.Label,
		&project.Description,
		&project.Visibility,
	)
	if err != nil {
		s.logger.With("error", err).Error("could not get project")
		return ProjectWithUserPermission{}, err
	}

	project_user_permission_query := "SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	err = s.db.conn.QueryRow(s.ctx, project_user_permission_query, project_id, user_id).Scan(&project.UserPermission)
	if err != nil {
		s.logger.With("error", err).Error("could not get project user permission")
		return ProjectWithUserPermission{}, err
	}

	return project, nil
}

type ProjectSampleDataCreate struct {
	Schema     uuid.UUID
	FilePath   string
	Timestamp  time.Time
	Properties []ProjectSampleDataPropertyCreate
}

type ProjectSampleDataPropertyCreate struct {
	Key   string
	Type  PropertyType
	Value any // TODO: match type
}

type sampleDataParsed struct {
	SampleIndex int
	DataIndex   int
	Timestamp   time.Time
	Payload     any
}

type ProjectSampleNoteCreate struct {
	Timestamp time.Time
	Content   string
}

type ProjectSampleCreate struct {
	Label      string
	Tags       []string
	Properties []Property
	Data       []ProjectSampleDataCreate
	Notes      []ProjectSampleNoteCreate
}

func (s *ProjectService) CreateProjectSamples(project uuid.UUID, samples []ProjectSampleCreate) (Ok, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return Ok{}, &UserNotAuthenticatedError{}
	}

	user_permission, err := s.project_user_permission(project, user_id)
	if err != nil ||
		(user_permission != PROJECT_USER_PERMISSION_OWNER &&
			user_permission != PROJECT_USER_PERMISSION_ADMIN &&
			user_permission != PROJECT_USER_PERMISSION_READ_WRITE) {
		s.logger.With("project", project, "user", user_id).Debug(
			"insufficient permissions to create samples for user in project",
		)
		return Ok{}, &InsufficientPermissionsError{}
	}

	if len(samples) == 0 {
		return Ok{}, nil
	}

	data_schemas, err := s.create_project_samples_sample_data_schemas(samples)
	if err != nil {
		return Ok{}, err
	}

	schema_data, err := s.create_project_samples_parse_sample_data_to_schema(samples, data_schemas)
	if err != nil {
		return Ok{}, err
	}

	tx, err := s.db.conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return Ok{}, err
	}
	defer tx.Rollback(s.ctx)

	sample_ids, err := s.create_project_samples_create_samples(tx, samples, user_id)
	if err != nil {
		return Ok{}, err
	}

	err = s.create_project_samples_create_sample_memberships(tx, samples, sample_ids, project, user_id)
	if err != nil {
		return Ok{}, err
	}

	err = s.create_project_samples_create_sample_tags(tx, samples, sample_ids, project)
	if err != nil {
		return Ok{}, err
	}

	err = s.create_project_samples_create_sample_properties(tx, samples, sample_ids)
	if err != nil {
		return Ok{}, err
	}

	sample_data_ids, err := s.create_project_samples_create_sample_data(tx, schema_data, sample_ids, user_id)
	if err != nil {
		return Ok{}, err
	}

	err = s.create_project_samples_create_sample_data_properties(tx, sample_data_ids, samples)
	if err != nil {
		return Ok{}, err
	}

	err = s.create_project_samples_store_sample_data(tx, schema_data, sample_data_ids, data_schemas)
	if err != nil {
		return Ok{}, err
	}

	err = s.create_project_samples_create_sample_notes(tx, samples, sample_ids, project, user_id)
	if err != nil {
		return Ok{}, err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		return Ok{}, err
	}

	return Ok{}, nil
}

func (s *ProjectService) project_user_permission(project uuid.UUID, user uuid.UUID) (ProjectUserPermission, error) {
	user_permission_query := "SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	var user_permission ProjectUserPermission
	err := s.db.conn.QueryRow(
		s.ctx,
		user_permission_query,
		project.String(),
		user.String(),
	).Scan(&user_permission)
	if err != nil {
		return ProjectUserPermission(""), err
	}

	return user_permission, nil
}

func (s *ProjectService) create_project_samples_sample_data_schemas(
	samples []ProjectSampleCreate,
) ([]DataSchema, error) {
	schema_ids := []uuid.UUID{}
	for _, sample := range samples {
		for _, data := range sample.Data {
			present := slices.Index(schema_ids, data.Schema) > -1
			if !present {
				schema_ids = append(schema_ids, data.Schema)
			}
		}
	}

	data_schemas, err := s.data_service.GetDataSchemasById(schema_ids)
	if err != nil {
		s.logger.With("error", err).Error("could not get data schemas")
		return nil, err
	}

	return data_schemas, nil
}

func (s *ProjectService) create_project_samples_parse_sample_data_to_schema(
	samples []ProjectSampleCreate,
	data_schemas []DataSchema,
) (map[uuid.UUID][]sampleDataParsed, error) {
	schema_id_counts := make(map[uuid.UUID]int)
	for _, sample := range samples {
		for _, data := range sample.Data {
			count, present := schema_id_counts[data.Schema]
			if !present {
				schema_id_counts[data.Schema] = 1
			} else {
				schema_id_counts[data.Schema] = count + 1
			}
		}
	}

	errs := ParseSampleDataErrors{}
	schema_data_idx := make(map[uuid.UUID]int, len(data_schemas))
	schema_data := make(map[uuid.UUID][]sampleDataParsed, len(data_schemas))
	for schema_id, count := range schema_id_counts {
		schema_data[schema_id] = make([]sampleDataParsed, count)
	}
	for sample_idx, sample := range samples {
		for data_idx, data := range sample.Data {
			file, err := os.Open(data.FilePath)
			if err != nil {
				s.logger.With("error", err, "file", data.FilePath).Error("could not open data file")
				errs.errors = append(errs.errors, err)
				continue
			}

			data_schema_idx := slices.IndexFunc(data_schemas, func(schema DataSchema) bool {
				return schema.Id == data.Schema
			})
			data_schema := data_schemas[data_schema_idx]
			ext := filepath.Ext(data.FilePath)
			data_parsed, err := parse_data_file_to_schema(ext, file, data_schema)
			if err != nil {
				s.logger.With("error", err, "sample", sample.Label, "data_idx", data_idx).Error("invalid sample data")
				errs.errors = append(errs.errors, err)
				continue
			}

			var payload any
			switch data_schema.Storage {
			case DATA_STORAGE_INTERNAL:
				payload = data_parsed
			case DATA_STORAGE_FILE:
				payload = data.FilePath
			default:
				s.logger.With("data_schema", data.Schema, "storage", data_schema.Storage).Error("invalid storage type")
				panic("invalid storage type")
			}

			sample_data_parsed := sampleDataParsed{
				SampleIndex: sample_idx,
				DataIndex:   data_idx,
				Timestamp:   data.Timestamp,
				Payload:     payload,
			}
			schema_idx := schema_data_idx[data.Schema]
			schema_data[data.Schema][schema_idx] = sample_data_parsed
			schema_data_idx[data.Schema] += 1
		}
	}

	if len(errs.errors) > 0 {
		s.logger.With("errors", errs.errors).Error("invalid data")
		return nil, &errs
	}

	return schema_data, nil
}

func (s *ProjectService) create_project_samples_create_samples(tx pgx.Tx, samples []ProjectSampleCreate, user_id uuid.UUID) ([]uuid.UUID, error) {
	var sample_create_query strings.Builder
	sample_create_query.WriteString("INSERT INTO sample_ (_creator) VALUES ")
	for idx := range samples {
		if idx > 0 {
			fmt.Fprintf(&sample_create_query, ", ")
		}

		fmt.Fprintf(&sample_create_query, "('%s')", user_id)
	}
	sample_create_query.WriteString(" RETURNING _id")
	rows, err := tx.Query(s.ctx, sample_create_query.String())
	if err != nil {
		s.logger.With("error", err).Error("could not create samples")
		return []uuid.UUID{}, err
	}

	sample_ids, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (uuid.UUID, error) {
		var id uuid.UUID
		err := row.Scan(&id)
		return id, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not collect user projects")
		return []uuid.UUID{}, err
	}

	return sample_ids, nil
}

func (s *ProjectService) create_project_samples_create_sample_memberships(
	tx pgx.Tx,
	samples []ProjectSampleCreate,
	sample_ids []uuid.UUID,
	project uuid.UUID,
	user_id uuid.UUID,
) error {
	labels := make([]any, len(samples))
	var project_membership_query strings.Builder
	project_membership_query.WriteString(
		"INSERT INTO project_sample_membership_ (_project, _sample, _creator, label) VALUES ",
	)
	for idx, sample_id := range sample_ids {
		if idx > 0 {
			fmt.Fprintf(&project_membership_query, ", ")
		}

		fmt.Fprintf(
			&project_membership_query,
			"('%s', '%s', '%s', $%d)",
			project,
			sample_id,
			user_id,
			idx+1,
		)
		labels[idx] = samples[idx].Label
	}
	_, err := tx.Exec(s.ctx, project_membership_query.String(), labels...)
	if err != nil {
		s.logger.With("error", err).Error("could not create project sample memberships")
		return err
	}

	return nil
}

func (s *ProjectService) create_project_samples_create_sample_tags(
	tx pgx.Tx,
	samples []ProjectSampleCreate,
	sample_ids []uuid.UUID,
	project uuid.UUID,
) error {
	num_tags := 0
	for _, sample := range samples {
		num_tags += len(sample.Tags)
	}
	if num_tags == 0 {
		return nil
	}

	tags := make([]any, num_tags)
	tidx := 0
	var sample_tags_query strings.Builder
	sample_tags_query.WriteString("INSERT INTO project_sample_tag_ (_sample, _project, _tag) VALUES ")
	for idx, sample_id := range sample_ids {
		sample_tags_set := []string{}
		for _, tag := range samples[idx].Tags {
			if slices.Contains(sample_tags_set, tag) {
				continue
			}
			sample_tags_set = append(sample_tags_set, tag)

			if tidx > 0 {
				fmt.Fprintf(&sample_tags_query, ", ")
			}
			fmt.Fprintf(
				&sample_tags_query,
				"('%s', '%s', $%d)",
				sample_id,
				project,
				tidx+1,
			)
			tags[tidx] = tag
			tidx += 1
		}
	}

	_, err := tx.Exec(s.ctx, sample_tags_query.String(), tags...)
	if err != nil {
		s.logger.With("error", err).Error("could not create project sample tags")
		return err
	}

	return nil
}

func (s *ProjectService) create_project_samples_create_sample_properties(
	tx pgx.Tx,
	samples []ProjectSampleCreate,
	sample_ids []uuid.UUID,
) error {
	const NUM_PROPERTY_VALUES = 3
	num_properties := 0
	for _, sample := range samples {
		num_properties += len(sample.Properties)
	}
	if num_properties == 0 {
		return nil
	}

	property_values := make([]any, num_properties*NUM_PROPERTY_VALUES)
	pidx := 0
	var sample_properties_query strings.Builder
	sample_properties_query.WriteString("INSERT INTO sample_property_ (_sample, _key, _type, value) VALUES ")
	for idx, sample_id := range sample_ids {
		for _, property := range samples[idx].Properties {
			if pidx > 0 {
				fmt.Fprint(&sample_properties_query, ", ")
			}

			key_idx := pidx
			type_idx := key_idx + 1
			value_idx := type_idx + 1
			fmt.Fprintf(
				&sample_properties_query,
				"('%s', $%d, $%d, $%d)",
				sample_id,
				key_idx+1,
				type_idx+1,
				value_idx+1,
			)

			property_value, err := json.Marshal(property.Value)
			if err != nil {
				s.logger.With("error", err, "key", property.Key, "value", property.Value).Error(
					"could not serialize property",
				)
				return err
			}

			property_values[key_idx] = property.Key
			property_values[type_idx] = property.Type
			property_values[value_idx] = property_value
			pidx += NUM_PROPERTY_VALUES
		}
	}
	_, err := tx.Exec(s.ctx, sample_properties_query.String(), property_values...)
	if err != nil {
		s.logger.With("error", err).Error("could not create sample properties")
		return err
	}

	return nil
}

type sampleDataIdx struct {
	SampleIndex int
	DataIndex   int
}

func (s *ProjectService) create_project_samples_create_sample_data(
	tx pgx.Tx,
	schema_data map[uuid.UUID][]sampleDataParsed,
	sample_ids []uuid.UUID,
	user_id uuid.UUID,
) (map[sampleDataIdx]uuid.UUID, error) {
	const QUERY_ARGS_SCHEMA_ID_OFFSET = 1
	var QUERY_ARGS_SAMPLE_ID_OFFSET = QUERY_ARGS_SCHEMA_ID_OFFSET + len(schema_data)
	var QUERY_ARGS_SAMPLE_DATA_OFFSET = QUERY_ARGS_SAMPLE_ID_OFFSET + len(sample_ids)

	sample_data_count := 0
	for _, data := range schema_data {
		sample_data_count += len(data)
	}
	if sample_data_count == 0 {
		return map[sampleDataIdx]uuid.UUID{}, nil
	}

	args_size := 1 + len(schema_data) + len(sample_ids) + sample_data_count
	args := make([]any, args_size)
	args[0] = user_id
	idx := QUERY_ARGS_SCHEMA_ID_OFFSET
	for schema_id := range schema_data {
		args[idx] = schema_id
		idx += 1
	}
	for idx, sample_id := range sample_ids {
		args[idx+QUERY_ARGS_SAMPLE_ID_OFFSET] = sample_id
	}

	var sample_data_create_query strings.Builder
	sample_data_create_query.WriteString(
		"INSERT INTO sample_data_ (_sample, _schema, _creator, timestamp) VALUES ",
	)

	sample_data_id_idx := make(map[sampleDataIdx]int, sample_data_count)
	sample_data_arg_idx := 0
	for schema_id, sample_data := range schema_data {
		schema_arg_idx := slices.IndexFunc(args, func(value any) bool {
			return value == schema_id
		})

		for _, sample_data_parsed := range sample_data {
			if sample_data_arg_idx > 0 {
				sample_data_create_query.WriteString(", ")
			}

			sample_id_arg_idx := sample_data_parsed.SampleIndex + QUERY_ARGS_SAMPLE_ID_OFFSET
			timestamp_arg_idx := sample_data_arg_idx + QUERY_ARGS_SAMPLE_DATA_OFFSET
			args[timestamp_arg_idx] = sample_data_parsed.Timestamp
			fmt.Fprintf(
				&sample_data_create_query,
				"($%d, $%d, $1, $%d)",
				sample_id_arg_idx+1,
				schema_arg_idx+1,
				timestamp_arg_idx+1,
			)

			data_key := sampleDataIdx{
				SampleIndex: sample_data_parsed.SampleIndex,
				DataIndex:   sample_data_parsed.DataIndex,
			}
			sample_data_id_idx[data_key] = sample_data_arg_idx

			sample_data_arg_idx += 1
		}
	}
	sample_data_create_query.WriteString(" RETURNING _id")

	rows, err := tx.Query(s.ctx, sample_data_create_query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", sample_data_create_query.String(),
			"args", args,
		).Error("could not create samples")
		return nil, err
	}

	ids, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (uuid.UUID, error) {
		var id uuid.UUID
		err := row.Scan(&id)
		return id, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not collect user projects")
		return nil, err
	}

	sample_data_ids := make(map[sampleDataIdx]uuid.UUID, sample_data_count)
	for key, id_idx := range sample_data_id_idx {
		sample_data_ids[key] = ids[id_idx]
	}
	return sample_data_ids, nil
}

func (s *ProjectService) create_project_samples_store_sample_data(
	tx pgx.Tx,
	schema_data map[uuid.UUID][]sampleDataParsed,
	sample_data_ids map[sampleDataIdx]uuid.UUID,
	data_schemas []DataSchema,
) error {
	for schema_id, parsed_data := range schema_data {
		data_schema_idx := slices.IndexFunc(data_schemas, func(schema DataSchema) bool {
			return schema.Id == schema_id
		})
		if data_schema_idx < 0 {
			s.logger.With("data schema", schema_id).Error("invalid data schema")
			panic("invalid data schema")
		}
		data_schema := data_schemas[data_schema_idx]

		var err error
		switch data_schema.Storage {
		case DATA_STORAGE_INTERNAL:
			err = s.create_project_samples_store_sample_data_internal(
				tx,
				data_schema,
				parsed_data,
				sample_data_ids,
			)
		case DATA_STORAGE_FILE:
			err = s.create_project_samples_store_sample_data_file(
				tx,
				data_schema,
				parsed_data,
				sample_data_ids,
			)
		default:
			panic("unexpected app.Storage")
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ProjectService) create_project_samples_store_sample_data_internal(
	tx pgx.Tx,
	data_schema DataSchema,
	parsed_data []sampleDataParsed,
	sample_data_ids map[sampleDataIdx]uuid.UUID,
) error {
	col_labels := make([]string, len(data_schema.Schema))
	for idx, col := range data_schema.Schema {
		col_labels[idx] = col.Label
	}

	var store_data_query strings.Builder
	fmt.Fprintf(
		&store_data_query,
		"INSERT INTO %s (_sample_data, %s) VALUES ",
		sample_data_table_name_from_schema_id(data_schema.Id),
		strings.Join(col_labels, ", "),
	)

	args_per_sample_data := len(data_schema.Schema) + 1
	args := make([]any, len(parsed_data)*args_per_sample_data)
	for data_idx, data := range parsed_data {
		sample_data_id_key := sampleDataIdx{SampleIndex: data.SampleIndex, DataIndex: data.DataIndex}
		sample_data_id := sample_data_ids[sample_data_id_key]
		payload := data.Payload.([]ColumnData)
		if len(payload) != len(data_schema.Schema) {
			panic("invalid payload")
		}

		args_offset := data_idx * args_per_sample_data
		args[args_offset] = sample_data_id
		for _, col_data := range payload {
			col_idx := slices.IndexFunc(col_labels, func(label string) bool {
				return label == col_data.Label
			})
			if col_idx < 0 {
				s.logger.With("column", col_data.Label).Error("invalid column label")
				panic("invalid column label")
			}

			data_arg_idx := args_offset + col_idx + 1
			args[data_arg_idx] = col_data.Data
		}

		if data_idx > 0 {
			store_data_query.WriteString(", ")
		}
		args_idx := make([]string, len(payload)+1)
		for idx := 0; idx < len(payload)+1; idx++ {
			args_idx[idx] = fmt.Sprintf("$%d", idx+args_offset+1)
		}
		fmt.Fprintf(
			&store_data_query,
			"(%s)",
			strings.Join(args_idx, ", "),
		)
	}

	_, err := tx.Exec(s.ctx, store_data_query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", store_data_query.String(),
			"args", args,
		).Error("could not insert data")
		return err
	}

	return nil
}

func (s *ProjectService) create_project_samples_store_sample_data_file(
	tx pgx.Tx,
	data_schema DataSchema,
	parsed_data []sampleDataParsed,
	sample_data_ids map[sampleDataIdx]uuid.UUID,
) error {
	const ARGS_PER_SAMPLE_DATA = 2
	var store_data_query strings.Builder
	fmt.Fprintf(
		&store_data_query,
		"INSERT INTO %s (_sample_data, %s) VALUES ",
		sample_data_table_name_from_schema_id(data_schema.Id),
		SAMPLE_DATA_STORAGE_TABLE_FILE_COL_LABEL,
	)

	args := make([]any, len(parsed_data)*ARGS_PER_SAMPLE_DATA)
	for data_idx, data := range parsed_data {
		sample_data_id_key := sampleDataIdx{SampleIndex: data.SampleIndex, DataIndex: data.DataIndex}
		sample_data_id := sample_data_ids[sample_data_id_key]
		payload := data.Payload.(string)

		args_offset := data_idx * ARGS_PER_SAMPLE_DATA
		args[args_offset] = sample_data_id
		args[args_offset+1] = payload

		if data_idx > 0 {
			store_data_query.WriteString(", ")
		}
		fmt.Fprintf(
			&store_data_query,
			"($%d, $%d)",
			args_offset+1,
			args_offset+2,
		)
	}

	_, err := tx.Exec(s.ctx, store_data_query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", store_data_query.String(),
			"args", args,
		).Error("could not insert data")
		return err
	}

	return nil
}

func (s *ProjectService) create_project_samples_create_sample_data_properties(
	tx pgx.Tx,
	sample_data_ids map[sampleDataIdx]uuid.UUID,
	samples []ProjectSampleCreate,
) error {
	const NUM_VALUES_PER_PROPERTY = 4

	properties_count := 0
	for _, sample := range samples {
		for _, data := range sample.Data {
			properties_count += len(data.Properties)
		}
	}
	if properties_count == 0 {
		return nil
	}

	var query strings.Builder
	args := make([]any, properties_count*NUM_VALUES_PER_PROPERTY)
	args_idx := 0

	query.WriteString(
		"INSERT INTO sample_data_property_ (_sample_data, _key, _type, value) VALUES ",
	)
	for sample_idx, sample := range samples {
		for data_idx, data := range sample.Data {
			sample_data_idx := sampleDataIdx{
				SampleIndex: sample_idx,
				DataIndex:   data_idx,
			}
			sample_data_id, present := sample_data_ids[sample_data_idx]
			if !present {
				s.logger.With(
					"sample data ids", sample_data_ids,
					"sample index", sample_idx,
					"data index", data_idx,
				).Error("invalid sample data index")
				panic("invalid sample data index")
			}

			for _, property := range data.Properties {
				id_arg_idx := args_idx
				key_arg_idx := args_idx + 1
				type_arg_idx := args_idx + 2
				value_arg_idx := args_idx + 3

				if args_idx > 0 {
					query.WriteString(", ")
				}

				value, err := json.Marshal(property.Value)
				if err != nil {
					s.logger.With(
						"error", err,
						"value", property.Value,
					).Error("could not serialize value")
					panic(err)
				}

				args[id_arg_idx] = sample_data_id
				args[key_arg_idx] = property.Key
				args[type_arg_idx] = property.Type
				args[value_arg_idx] = string(value)

				fmt.Fprintf(
					&query,
					"($%d, $%d, $%d, $%d)",
					id_arg_idx+1,
					key_arg_idx+1,
					type_arg_idx+1,
					value_arg_idx+1,
				)

				args_idx += NUM_VALUES_PER_PROPERTY
			}
		}
	}

	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", query.String(),
			"args", args,
		).Error("could not create sample data properties")
		return err
	}

	return nil
}

func (s *ProjectService) create_project_samples_create_sample_notes(
	tx pgx.Tx,
	samples []ProjectSampleCreate,
	sample_ids []uuid.UUID,
	project uuid.UUID,
	user_id uuid.UUID,
) error {
	const NUM_NOTE_VALUES = 2
	num_notes := 0
	for _, sample := range samples {
		num_notes += len(sample.Notes)
	}
	if num_notes == 0 {
		return nil
	}

	args := make([]any, num_notes*NUM_NOTE_VALUES)
	arg_idx := 0
	var sample_notes_query strings.Builder
	sample_notes_query.WriteString(
		`INSERT INTO project_sample_note_ 
			(_project, _sample, _creator, timestamp, content) 
			VALUES `,
	)
	for sample_idx, sample := range samples {
		for _, note := range sample.Notes {
			if arg_idx > 0 {
				sample_notes_query.WriteString(", ")
			}

			timestamp_idx := arg_idx
			note_idx := timestamp_idx + 1
			fmt.Fprintf(
				&sample_notes_query,
				"('%s', '%s', '%s', $%d, $%d)",
				project,
				sample_ids[sample_idx],
				user_id,
				timestamp_idx+1,
				note_idx+1,
			)

			args[timestamp_idx] = note.Timestamp
			args[note_idx] = note.Content
			arg_idx += NUM_NOTE_VALUES
		}
	}
	_, err := tx.Exec(s.ctx, sample_notes_query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", sample_notes_query,
			"args", args,
		).Error("could not create sample notes")
		return err
	}

	return nil
}

type ParseSampleDataErrors struct {
	errors []error
}

func (e *ParseSampleDataErrors) Error() string {
	msgs := make([]string, len(e.errors))
	for idx, err := range e.errors {
		msgs[idx] = err.Error()
	}

	return fmt.Sprintf("{%s}", strings.Join(msgs, "; "))
}

type ProjectSampleResources struct {
	Id           uuid.UUID
	Creator      uuid.UUID
	Properties   []Property
	ProjectTags  []string
	ProjectNotes []ProjectSampleNote
	Data         []SampleData
	DataSchemas  []DataSchema
	Users        []User
}

func (s *ProjectService) GetProjectSampleResources(project_id uuid.UUID, sample_id uuid.UUID) (ProjectSampleResources, error) {
	var err error
	resources := ProjectSampleResources{
		Id: sample_id,
	}

	resources.Creator, err = s.get_project_sample_resources_sample_creator(resources.Id)
	if err != nil {
		return ProjectSampleResources{}, nil
	}

	resources.Properties, err = s.get_project_sample_resources_sample_properties(resources.Id)
	if err != nil {
		return ProjectSampleResources{}, nil
	}

	resources.ProjectTags, err = s.get_project_sample_resources_sample_tags(project_id, resources.Id)
	if err != nil {
		return ProjectSampleResources{}, nil
	}

	resources.ProjectNotes, err = s.get_project_sample_resources_sample_notes(project_id, resources.Id)
	if err != nil {
		return ProjectSampleResources{}, nil
	}

	resources.Data, err = s.get_project_sample_resources_sample_data(resources.Id)
	if err != nil {
		return ProjectSampleResources{}, nil
	}

	data_schema_ids := []uuid.UUID{}
	for _, data := range resources.Data {
		data_schema_ids = append(data_schema_ids, data.Schema)
	}
	resources.DataSchemas, err = s.data_service.GetDataSchemasById(data_schema_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"data schemas", data_schema_ids,
		).Error("could not get data schemas for sample data")
		return ProjectSampleResources{}, err
	}

	user_ids := []uuid.UUID{resources.Creator}
	for _, note := range resources.ProjectNotes {
		user_ids = append(user_ids, note.Creator)
	}
	for _, data := range resources.Data {
		user_ids = append(user_ids, data.Creator)
	}
	resources.Users, err = s.user_service.GetUsersById(user_ids)
	if err != nil {
		return ProjectSampleResources{}, err
	}

	return resources, nil
}

func (s *ProjectService) get_project_sample_resources_sample_creator(sample_id uuid.UUID) (uuid.UUID, error) {
	var sample_creator_id uuid.UUID
	creator_query := "SELECT _creator FROM sample_ WHERE _id=$1"
	err := s.db.conn.QueryRow(s.ctx, creator_query, sample_id).Scan(&sample_creator_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.With("sample id", sample_id).Debug("sample not found")
			return uuid.Nil, &RecordNotFoundError{}
		} else {
			s.logger.With("error", err, "query", creator_query).Error("could not get sample creator")
			return uuid.Nil, err
		}
	}

	return sample_creator_id, nil

	// var user User
	// user_query := "SELECT _id, account_status, email, name, role FROM user_ WHERE _id=$1"
	// err = s.db.conn.QueryRow(s.ctx, user_query, sample_creator_id).Scan(&user)
	// if err!= nil {
	// 	s.logger.With("error", err, "query", user_query).Error("could not get user")
	// 	return User{}, err
	// }

	// return user, nil
}

func (s *ProjectService) get_project_sample_resources_sample_properties(sample_id uuid.UUID) ([]Property, error) {
	properties_query := "SELECT _key, _type, value FROM sample_property_ WHERE _sample=$1"
	rows, _ := s.db.conn.Query(s.ctx, properties_query, sample_id)
	properties, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Property, error) {
		var property Property
		err := row.Scan(&property.Key, &property.Type, &property.Value)
		return property, err
	})
	if err != nil {
		s.logger.With("error", err, "query", properties_query).Error("could not get sample properties")
		return nil, err
	}

	return properties, nil
}

func (s *ProjectService) get_project_sample_resources_sample_tags(project_id uuid.UUID, sample_id uuid.UUID) ([]string, error) {
	tags_query := "SELECT _tag FROM project_sample_tag_ WHERE _project=$1 AND _sample=$2"
	rows, _ := s.db.conn.Query(s.ctx, tags_query, project_id, sample_id)
	tags, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var tag string
		err := row.Scan(&tag)
		return tag, err
	})
	if err != nil {
		s.logger.With("error", err, "query", tags_query).Error("could not get project sample tags")
		return nil, err
	}

	return tags, nil
}

func (s *ProjectService) get_project_sample_resources_sample_notes(project_id uuid.UUID, sample_id uuid.UUID) ([]ProjectSampleNote, error) {
	notes_query :=
		`SELECT _id, _sample, _project, _creator, timestamp, content
	FROM project_sample_note_ WHERE _project=$1 AND _sample=$2`
	rows, _ := s.db.conn.Query(s.ctx, notes_query, project_id, sample_id)
	notes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ProjectSampleNote, error) {
		var note ProjectSampleNote
		err := row.Scan(
			&note.Id,
			&note.Sample,
			&note.Project,
			&note.Creator,
			&note.Timestamp,
			&note.Content,
		)
		return note, err
	})
	if err != nil {
		s.logger.With("error", err, "query", notes_query).Error("could not get project sampel notes")
		return nil, err
	}

	return notes, nil
}

func (s *ProjectService) get_project_sample_resources_sample_data(sample_id uuid.UUID) ([]SampleData, error) {
	data_query := "SELECT _id, _sample, _schema, _creator, timestamp FROM sample_data_ WHERE _sample=$1"
	rows, _ := s.db.conn.Query(s.ctx, data_query, sample_id)
	data, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SampleData, error) {
		var data SampleData
		err := row.Scan(&data.Id, &data.Sample, &data.Schema, &data.Creator, &data.Timestamp)
		return data, err
	})
	if err != nil {
		s.logger.With("error", err, "query", data_query).Error("could not get sample data")
		return nil, err
	}

	return data, nil
}
