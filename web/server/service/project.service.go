package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syredb/database"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProjectUserPermission string

const (
	ProjectUserPermissionRead      ProjectUserPermission = "read"
	ProjectUserPermissionReadWrite ProjectUserPermission = "read_write"
	ProjectUserPermissionAdmin     ProjectUserPermission = "admin"
	ProjectUserPermissionOwner     ProjectUserPermission = "owner"
)

type ProjectSampleUserPermission string

const (
	ProjectSampleUserPermissionModifyLabel      ProjectSampleUserPermission = "modify_label"
	ProjectSampleUserPermissionModifyTags       ProjectSampleUserPermission = "modify_tags"
	ProjectSampleUserPermissionModifyProperties ProjectSampleUserPermission = "modify_properties"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type ProjectService struct {
	ctx          context.Context
	logger       *slog.Logger
	db           *database.DBConnection
	user_service *UserService
	data_service *DataService
}

func NewProjectService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DBConnection,
	user_service *UserService,
	data_service *DataService,
) *ProjectService {
	return &ProjectService{
		ctx:          ctx,
		logger:       logger,
		db:           db,
		user_service: user_service,
		data_service: data_service,
	}
}

type Project struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Label       string
	Description string
	Visibility  Visibility
}

func (s *ProjectService) GetUserProjects(user uuid.UUID) ([]Project, error) {
	if user == uuid.Nil {
		panic("invalid user id")
	}

	user_project_query :=
		`SELECT _id, _creator, label, description, visibility FROM project_ 
		WHERE _creator=$1 ORDER BY _id`
	rows, _ := s.db.Conn.Query(s.ctx, user_project_query, user)
	user_projects, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Project, error) {
		var project Project
		err := row.Scan(
			&project.Id,
			&project.Creator,
			&project.Label,
			&project.Description,
			&project.Visibility,
		)
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
	Visibility  Visibility
}

func (s *ProjectService) CreateProject(
	user_id uuid.UUID,
	project ProjectCreate,
) (uuid.UUID, error) {
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(s.ctx)

	var project_id uuid.UUID
	create_project_query :=
		`INSERT INTO project_ (_creator, label, description, visibility) 
		VALUES ($1, $2, $3, $4) 
		RETURNING _id`
	err = tx.QueryRow(
		s.ctx,
		create_project_query,
		user_id,
		project.Label,
		project.Description,
		project.Visibility,
	).Scan(&project_id)
	if err != nil {
		s.logger.With("error", err).Error("could not create project")
		return uuid.Nil, err
	}

	set_user_permission_query :=
		`INSERT INTO project_user_permission_ (_project, _user, permission) 
		VALUES ($1, $2, $3)`
	_, err = tx.Exec(
		s.ctx,
		set_user_permission_query,
		project_id,
		user_id,
		ProjectUserPermissionOwner,
	)
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

func (s *ProjectService) ProjectUserPermission(
	project uuid.UUID,
	user uuid.UUID,
) (ProjectUserPermission, error) {
	var permission ProjectUserPermission
	query := "SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	err := s.db.Conn.QueryRow(
		s.ctx,
		query,
		project,
		user,
	).Scan(&permission)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.With(
				"error", err,
				"user", user,
				"project", project,
			).Error("could not retrieve project user permission")
		}
		return "", err
	}

	return permission, nil
}

type PropertyType string

const (
	PropertyTypeString    PropertyType = "string"
	PropertyTypeBool      PropertyType = "bool"
	PropertyTypeUint      PropertyType = "uint"
	PropertyTypeInt       PropertyType = "int"
	PropertyTypeFloat     PropertyType = "float"
	PropertyTypeQuantity  PropertyType = "quantity"
	PropertyTypeTimestamp PropertyType = "timestamp"
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
	Id         uuid.UUID
	Sample     uuid.UUID
	Schema     uuid.UUID
	Creator    uuid.UUID
	Timestamp  time.Time
	Visibility Visibility
	Label      *string
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

func (s *ProjectService) GetProjectResources(
	user_id uuid.UUID,
	project_id uuid.UUID,
) (ProjectResources, error) {
	var project_resources ProjectResources
	project_query :=
		"SELECT _id, _creator, label, description, visibility FROM project_ WHERE _id=$1"
	err := s.db.Conn.QueryRow(
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
		s.logger.With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get project")
		return ProjectResources{}, err
	}

	project_user_permission_query :=
		`SELECT permission FROM project_user_permission_ 
		WHERE _project=$1 AND _user=$2`
	err = s.db.Conn.QueryRow(
		s.ctx,
		project_user_permission_query,
		project_id,
		user_id,
	).Scan(&project_resources.ProjectUserPermission)
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get project user permission")
		return ProjectResources{}, err
	}

	project_tags_query := "SELECT _tag FROM project_tag_ WHERE _project=$1"
	project_tag_rows, _ := s.db.Conn.Query(s.ctx, project_tags_query, project_id)
	project_resources.ProjectTags, err = pgx.CollectRows(
		project_tag_rows,
		func(row pgx.CollectableRow) (string, error) {
			var tag string
			err := row.Scan(&tag)
			return tag, err
		},
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", project_tags_query,
			"project", project_id,
		).Error("could not get project tags")
	}

	project_note_count_query := "SELECT COUNT(*) FROM project_note_ WHERE _project=$1"
	err = s.db.Conn.QueryRow(
		s.ctx,
		project_note_count_query,
		project_id,
	).Scan(&project_resources.ProjectNoteCount)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", project_note_count_query,
			"project", project_id,
		).Error("could not get project note count")
	}

	project_sample_membership_query := `
		SELECT _sample, _creator, _timestamp, label 
		FROM project_sample_membership_ 
		WHERE _project=$1
		ORDER BY label
	`
	project_sample_membership_rows, _ := s.db.Conn.Query(
		s.ctx,
		project_sample_membership_query,
		project_id,
	)
	project_resources.Samples, err = pgx.CollectRows(
		project_sample_membership_rows,
		func(row pgx.CollectableRow) (ProjectSample, error) {
			var sample ProjectSample
			err := row.Scan(
				&sample.Id,
				&sample.MembershipCreator,
				&sample.MembershipCreated,
				&sample.Label,
			)
			return sample, err
		},
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", project_sample_membership_query,
			"project", project_id,
		).Error("could not get project sample memberships")
		project_resources.Samples = []ProjectSample{}
	}

	sample_ids := make([]uuid.UUID, len(project_resources.Samples))
	for idx, sample := range project_resources.Samples {
		sample_ids[idx] = sample.Id
	}

	sample_info, err := s.get_project_resources_sample_info(user_id, project_id, sample_ids)
	if err != nil {
		return ProjectResources{}, err
	}
	for _, info := range sample_info {
		idx := slices.IndexFunc(project_resources.Samples, func(sample ProjectSample) bool {
			return sample.Id == info.Id
		})
		if idx < 0 {
			s.logger.With("sample", info.Id).Error("could not find sample")
			panic("could not find sample")
		}

		project_resources.Samples[idx].Tags = info.Tags
		project_resources.Samples[idx].Properties = info.Properties
		project_resources.Samples[idx].NoteCount = info.NoteCount
	}

	sample_data_query :=
		`SELECT _id, _sample, _schema, _creator, timestamp FROM sample_data_ 
		WHERE _sample=ANY($1)`
	sample_data_rows, _ := s.db.Conn.Query(s.ctx, sample_data_query, sample_ids)
	project_resources.SampleData, err = pgx.CollectRows(
		sample_data_rows,
		func(row pgx.CollectableRow) (SampleData, error) {
			var sample_data SampleData
			err := row.Scan(
				&sample_data.Id,
				&sample_data.Sample,
				&sample_data.Schema,
				&sample_data.Creator,
				&sample_data.Timestamp,
			)
			return sample_data, err
		},
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", sample_data_query,
			"samples", sample_ids,
		).Error("could not get sample data")
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

type ProjectSampleInfo struct {
	Id         uuid.UUID
	Tags       []string
	Properties []Property
	NoteCount  uint
}

func (s *ProjectService) get_project_resources_sample_info(
	user_id uuid.UUID,
	project_id uuid.UUID,
	sample_ids []uuid.UUID,
) ([]ProjectSampleInfo, error) {
	info := make([]ProjectSampleInfo, len(sample_ids))
	for idx, sample_id := range sample_ids {
		info[idx].Id = sample_id
	}

	tags_query :=
		`SELECT _sample, _tag FROM project_sample_tag_
		WHERE _project=$1 AND _sample=ANY($2)
		GROUP BY _sample`
	rows, _ := s.db.Conn.Query(s.ctx, tags_query, sample_ids)
	for rows.Next() {
		var sample_id uuid.UUID
		var tags []string
		err := rows.Scan(&sample_id, &tags)
		if err != nil {
			s.logger.With("error", err).Error("could not get sample tags")
			return nil, err
		}

		sample_info_idx := slices.IndexFunc(info, func(sample_info ProjectSampleInfo) bool {
			return sample_info.Id == sample_id
		})
		if sample_info_idx < 0 {
			s.logger.With("sample", sample_id).Error("could not find sample")
			panic("could not find sample")
		}

		info[sample_info_idx].Tags = tags
	}

	properties_query :=
		`SELECT _sample, _key, _type, value FROM sample_property_
		WHERE _sample=ANY($1)
		GROUP BY _sample`
	rows, _ = s.db.Conn.Query(s.ctx, properties_query, sample_ids)
	for rows.Next() {
		var sample_id uuid.UUID
		var properties []Property
		err := rows.Scan(&sample_id, &properties)
		if err != nil {
			s.logger.With(
				"error", err,
				"query", properties_query,
				"samples", sample_ids,
			).Error("could not get sample properties")
			return nil, err
		}

		sample_info_idx := slices.IndexFunc(info, func(sample_info ProjectSampleInfo) bool {
			return sample_info.Id == sample_id
		})
		if sample_info_idx < 0 {
			s.logger.With("sample", sample_id).Error("could not find sample")
			panic("could not find sample")
		}

		info[sample_info_idx].Properties = properties
	}

	note_count_query :=
		`SELECT _sample, COUNT(*) FROM sample_note_ 
		WHERE _sample=ANY($1) AND (_creator=$2 OR visibility='public')
		GROUP BY _sample`
	rows, _ = s.db.Conn.Query(s.ctx, note_count_query, sample_ids, user_id)
	for rows.Next() {
		var sample_id uuid.UUID
		var count uint
		err := rows.Scan(&sample_id, &count)
		if err != nil {
			s.logger.With(
				"error", err,
				"query", note_count_query,
				"samples", sample_ids,
				"user", user_id,
			).Error("could not get sample note count")
			return nil, err
		}

		sample_info_idx := slices.IndexFunc(info, func(sample_info ProjectSampleInfo) bool {
			return sample_info.Id == sample_id
		})
		if sample_info_idx < 0 {
			s.logger.With("sample", sample_id).Error("could not find sample")
			panic("could not find sample")
		}

		info[sample_info_idx].NoteCount = count
	}

	project_note_count_query :=
		`SELECT _sample, COUNT(*) FROM project_sample_note_ 
		WHERE _project=$1 AND _sample=ANY($2) AND (_creator=$2 OR visibility='public')
		GROUP BY _sample`
	rows, _ = s.db.Conn.Query(s.ctx, project_note_count_query, project_id, sample_ids, user_id)
	for rows.Next() {
		var sample_id uuid.UUID
		var count uint
		err := rows.Scan(&sample_id, &count)
		if err != nil {
			s.logger.With(
				"error", err,
				"query", project_note_count_query,
				"project", project_id,
				"samples", sample_ids,
				"user", user_id,
			).Error("could not get project sample note count")
			return nil, err
		}

		sample_info_idx := slices.IndexFunc(info, func(sample_info ProjectSampleInfo) bool {
			return sample_info.Id == sample_id
		})
		if sample_info_idx < 0 {
			s.logger.With("sample", sample_id).Error("could not find sample")
			panic("could not find sample")
		}

		info[sample_info_idx].NoteCount += count
	}

	return info, nil
}

type ProjectWithUserPermission struct {
	Id             uuid.UUID
	Creator        uuid.UUID
	Label          string
	Description    string
	Visibility     Visibility
	UserPermission ProjectUserPermission
}

func (s *ProjectService) GetProjectWithUserPermission(
	user_id uuid.UUID,
	project_id uuid.UUID,
) (ProjectWithUserPermission, error) {
	var project ProjectWithUserPermission
	project_query :=
		"SELECT _id, _creator, label, description, visibility FROM project_ WHERE _id=$1"
	err := s.db.Conn.QueryRow(
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

	project_user_permission_query :=
		`SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2`
	err = s.db.Conn.QueryRow(
		s.ctx,
		project_user_permission_query,
		project_id,
		user_id,
	).Scan(&project.UserPermission)
	if err != nil {
		s.logger.With(
			"error", err,
			"query", project_user_permission_query,
			"project", project_id,
			"user", user_id,
		).Error("could not get project user permission")
		return ProjectWithUserPermission{}, err
	}

	return project, nil
}

type FileInfo struct {
	Name string
	Size int64
	File *os.File
}

type ProjectSampleDataCreate struct {
	Schema     uuid.UUID
	File       FileInfo
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

type SampleDataPayloadExternal struct {
	Path     string
	Filename string
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

func (s *ProjectService) CreateProjectSamples(
	user_id uuid.UUID,
	project uuid.UUID,
	samples []ProjectSampleCreate,
) error {
	if len(samples) == 0 {
		return nil
	}

	data_schemas, err := s.create_project_samples_sample_data_schemas(samples)
	if err != nil {
		return err
	}

	schema_data, err := s.create_project_samples_parse_sample_data_to_schema(
		samples,
		data_schemas,
	)
	if err != nil {
		return err
	}

	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return err
	}
	defer tx.Rollback(s.ctx)

	sample_ids, err := s.create_project_samples_create_samples(tx, samples, user_id)
	if err != nil {
		return err
	}

	err = s.create_project_samples_create_sample_memberships(
		tx,
		samples,
		sample_ids,
		project,
		user_id,
	)
	if err != nil {
		return err
	}

	err = s.create_project_samples_create_sample_tags(tx, samples, sample_ids, project)
	if err != nil {
		return err
	}

	err = s.create_project_samples_create_sample_properties(tx, samples, sample_ids)
	if err != nil {
		return err
	}

	sample_data_ids, err := s.create_project_samples_create_sample_data(
		tx,
		schema_data,
		sample_ids,
		user_id,
	)
	if err != nil {
		return err
	}

	err = s.create_project_samples_create_sample_data_properties(
		tx,
		sample_data_ids,
		samples,
	)
	if err != nil {
		return err
	}

	err = s.create_project_samples_store_sample_data(
		tx,
		schema_data,
		sample_data_ids,
		data_schemas,
	)
	if err != nil {
		return err
	}

	err = s.create_project_samples_sample_data_user_permisson_as_owner(
		tx,
		sample_data_ids,
		user_id,
	)
	if err != nil {
		return err
	}

	err = s.create_project_samples_create_sample_notes(
		tx,
		samples,
		sample_ids,
		project,
		user_id,
	)
	if err != nil {
		return err
	}

	err = s.create_project_samples_create_sample_user_permissions(tx, sample_ids, user_id)
	if err != nil {
		return err
	}

	err = s.create_project_samples_create_project_sample_user_permissions(tx, sample_ids, user_id, project)
	if err != nil {
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		return err
	}

	return nil
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
			data_schema_idx := slices.IndexFunc(data_schemas, func(schema DataSchema) bool {
				return schema.Id == data.Schema
			})
			data_schema := data_schemas[data_schema_idx]
			ext := filepath.Ext(data.File.Name)
			data_parsed, err := parse_data_file_to_schema(ext, data.File.File, data_schema)
			if err != nil {
				s.logger.With(
					"error", err,
					"sample", sample.Label,
					"data_idx", data_idx,
				).Error("invalid sample data")
				errs.errors = append(errs.errors, err)
				continue
			}

			var payload any
			switch data_schema.Storage {
			case DataStorageInternal:
				payload = data_parsed
			case DataStorageExternal:
				payload = SampleDataPayloadExternal{
					Path:     "TODO",
					Filename: data.File.Name,
				}
			default:
				s.logger.With(
					"data_schema", data.Schema,
					"storage", data_schema.Storage,
				).Error("invalid storage type")
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

func (s *ProjectService) create_project_samples_create_samples(
	tx pgx.Tx,
	samples []ProjectSampleCreate,
	user_id uuid.UUID,
) ([]uuid.UUID, error) {
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
	sample_tags_query.WriteString(
		"INSERT INTO project_sample_tag_ (_sample, _project, _tag) VALUES ",
	)
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
	sample_properties_query.WriteString(
		"INSERT INTO sample_property_ (_sample, _key, _type, value) VALUES ",
	)
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
				s.logger.With(
					"error", err,
					"key", property.Key,
					"value", property.Value,
				).Error("could not serialize property")
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
		case DataStorageInternal:
			err = s.create_project_samples_store_sample_data_internal(
				tx,
				data_schema,
				parsed_data,
				sample_data_ids,
			)
		case DataStorageExternal:
			err = s.create_project_samples_store_sample_data_external(
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

func (s *ProjectService) create_project_samples_sample_data_user_permisson_as_owner(
	tx pgx.Tx,
	sample_data_ids map[sampleDataIdx]uuid.UUID,
	user uuid.UUID,
) error {
	if len(sample_data_ids) == 0 {
		return nil
	}

	args := make([]any, len(sample_data_ids)+2)
	args[0] = user
	args[1] = SampleDataUserPermissionOwner
	arg_idx := 2
	var query strings.Builder
	query.WriteString("INSERT INTO sample_data_user_permission_ (_sample_data, _user, _permission) VALUES ")
	for _, sample_data := range sample_data_ids {
		args[arg_idx] = sample_data
		fmt.Fprintf(&query, "($%d, $1, $2)", arg_idx+1)
		arg_idx += 1
	}

	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"sample data", sample_data_ids,
			"user", user,
		).Error("could not insert sample data user permission")
		return err
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
		data_storage_table_name_from_schema_id(data_schema.Id),
		strings.Join(col_labels, ", "),
	)

	args_per_sample_data := len(data_schema.Schema) + 1
	args := make([]any, len(parsed_data)*args_per_sample_data)
	for data_idx, data := range parsed_data {
		sample_data_id_key := sampleDataIdx{
			SampleIndex: data.SampleIndex,
			DataIndex:   data.DataIndex,
		}
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
			args[data_arg_idx] = col_data.Values
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

func (s *ProjectService) create_project_samples_store_sample_data_external(
	tx pgx.Tx,
	data_schema DataSchema,
	parsed_data []sampleDataParsed,
	sample_data_ids map[sampleDataIdx]uuid.UUID,
) error {
	const ARGS_PER_SAMPLE_DATA = 3
	var store_data_query strings.Builder
	fmt.Fprintf(
		&store_data_query,
		"INSERT INTO %s (_sample_data, %s, %s) VALUES ",
		data_storage_table_name_from_schema_id(data_schema.Id),
		DATA_STORAGE_TABLE_EXTERNAL_COL_PATH_LABEL,
		DATA_STORAGE_TABLE_EXTERNAL_COL_FILENAME_LABEL,
	)

	args := make([]any, len(parsed_data)*ARGS_PER_SAMPLE_DATA)
	for data_idx, data := range parsed_data {
		sample_data_id_key := sampleDataIdx{
			SampleIndex: data.SampleIndex,
			DataIndex:   data.DataIndex,
		}
		sample_data_id := sample_data_ids[sample_data_id_key]
		payload := data.Payload.(SampleDataPayloadExternal)

		args_offset := data_idx * ARGS_PER_SAMPLE_DATA
		arg_idx_id := args_offset
		arg_idx_path := arg_idx_id + 1
		arg_idx_filename := arg_idx_path + 1
		args[arg_idx_id] = sample_data_id
		args[arg_idx_path] = payload.Path
		args[arg_idx_filename] = payload.Filename

		if data_idx > 0 {
			store_data_query.WriteString(", ")
		}
		fmt.Fprintf(
			&store_data_query,
			"($%d, $%d, $%d)",
			arg_idx_id+1,
			arg_idx_path+1,
			arg_idx_filename+1,
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

func (s *ProjectService) create_project_samples_create_sample_user_permissions(
	tx pgx.Tx,
	sample_ids []uuid.UUID,
	user_id uuid.UUID,
) error {
	args := make([]any, len(sample_ids)+2)
	args[0] = user_id
	args[1] = SampleUserPermissionOwner

	var query strings.Builder
	query.WriteString(
		"INSERT INTO sample_user_permission_ (_sample, _user, _permission) VALUES ",
	)
	for idx, sample_id := range sample_ids {
		if idx > 0 {
			query.WriteString(", ")
		}

		arg_idx := idx + 2
		fmt.Fprintf(&query, "($%d, $1, $2)", arg_idx+1)
		args[arg_idx] = sample_id
	}
	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"samples", sample_ids,
			"user", user_id,
		).Error("could not create sample user permissions")
		return err
	}

	return nil
}

func (s *ProjectService) create_project_samples_create_project_sample_user_permissions(
	tx pgx.Tx,
	sample_ids []uuid.UUID,
	user_id uuid.UUID,
	project_id uuid.UUID,
) error {
	const PROJECT_SAMPLE_USER_PERMISSION_COUNT = 3
	args := make([]any, len(sample_ids)+PROJECT_SAMPLE_USER_PERMISSION_COUNT+2)
	args[0] = project_id
	args[1] = user_id
	args[2] = ProjectSampleUserPermissionModifyLabel
	args[3] = ProjectSampleUserPermissionModifyTags
	args[4] = ProjectSampleUserPermissionModifyProperties

	var query strings.Builder
	query.WriteString(
		"INSERT INTO project_sample_user_permission_ (_project, _sample, _user, _permission) VALUES ",
	)
	for idx, sample_id := range sample_ids {
		sample_idx := idx + PROJECT_SAMPLE_USER_PERMISSION_COUNT + 2
		args[sample_idx] = sample_id
		for perm_idx := range PROJECT_SAMPLE_USER_PERMISSION_COUNT {
			if idx > 0 || perm_idx > 0 {
				query.WriteString(", ")
			}
			fmt.Fprintf(&query, "($1, $%d, $2, $%d)", sample_idx+1, perm_idx+3)
		}
	}
	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project_id,
			"samples", sample_ids,
			"user", user_id,
			"q", query.String(),
		).Error("could not create project sample user permissions")
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

type ProjectSampleMembershipAsResource struct {
	Creator   User
	Timestamp time.Time
	Label     string
}

type SampleUserPermissions struct {
	User        uuid.UUID
	Permissions []SampleUserPermission
}

type ProjectSampleUserPermissions struct {
	User        uuid.UUID
	Permissions []ProjectSampleUserPermission
}

type ProjectSampleResources struct {
	Id                           uuid.UUID
	Creator                      uuid.UUID
	Properties                   []Property
	ProjectMembership            ProjectSampleMembershipAsResource
	ProjectTags                  []string
	ProjectNotes                 []ProjectSampleNote
	Data                         []SampleData
	DataSchemas                  []DataSchema
	Users                        []User
	SampleUserPermissions        []SampleUserPermissions
	ProjectSampleUserPermissions []ProjectSampleUserPermissions
}

func (s *ProjectService) GetProjectSampleResources(
	user_id, project_id uuid.UUID,
	sample_id uuid.UUID,
) (ProjectSampleResources, error) {
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

	resources.ProjectMembership, err = s.get_project_sample_resources_project_membership(
		user_id,
		project_id,
		resources.Id,
	)
	if err != nil {
		return ProjectSampleResources{}, nil
	}

	resources.ProjectTags, err = s.get_project_sample_tags(
		project_id,
		resources.Id,
	)
	if err != nil {
		return ProjectSampleResources{}, nil
	}

	resources.ProjectNotes, err = s.get_project_sample_resources_sample_notes(
		project_id,
		resources.Id,
	)
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

	resources.SampleUserPermissions, err = s.get_project_sample_resources_sample_user_permissions(
		sample_id,
	)
	if err != nil {
		return ProjectSampleResources{}, err
	}

	resources.ProjectSampleUserPermissions, err = s.get_project_sample_resources_project_sample_user_permissions(
		project_id,
		sample_id,
	)
	if err != nil {
		return ProjectSampleResources{}, err
	}

	return resources, nil
}

func (s *ProjectService) get_project_sample_resources_sample_creator(
	sample_id uuid.UUID,
) (uuid.UUID, error) {
	var sample_creator_id uuid.UUID
	creator_query := "SELECT _creator FROM sample_ WHERE _id=$1"
	err := s.db.Conn.QueryRow(s.ctx, creator_query, sample_id).Scan(&sample_creator_id)
	if err != nil {
		s.logger.With(
			"error", err,
			"sample", sample_id,
		).Error("could not get sample creator")
		return uuid.Nil, err
	}

	return sample_creator_id, nil
}

func (s *ProjectService) get_project_sample_resources_sample_properties(
	sample_id uuid.UUID,
) ([]Property, error) {
	properties_query := "SELECT _key, _type, value FROM sample_property_ WHERE _sample=$1"
	rows, _ := s.db.Conn.Query(s.ctx, properties_query, sample_id)
	properties, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Property, error) {
		var property Property
		err := row.Scan(&property.Key, &property.Type, &property.Value)
		return property, err
	})
	if err != nil {
		s.logger.With(
			"error", err,
			"query", properties_query,
		).Error("could not get sample properties")
		return nil, err
	}

	return properties, nil
}

func (s *ProjectService) get_project_sample_resources_project_membership(
	user_id uuid.UUID,
	project_id uuid.UUID,
	sample_id uuid.UUID,
) (ProjectSampleMembershipAsResource, error) {
	var membership ProjectSampleMembershipAsResource
	var creator_id uuid.UUID
	membership_query :=
		`SELECT _creator, _timestamp, label FROM project_sample_membership_
		WHERE _project=$1 AND _sample=$2`
	err := s.db.Conn.QueryRow(
		s.ctx,
		membership_query,
		project_id,
		sample_id,
	).Scan(&creator_id, &membership.Timestamp, &membership.Label)
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project_id,
			"sample", sample_id,
		).Error("could not get project sample membership")
		return ProjectSampleMembershipAsResource{}, err
	}

	membership.Creator, err = s.user_service.GetUserById(user_id)
	if err != nil {
		return ProjectSampleMembershipAsResource{}, err
	}

	return membership, nil
}

func (s *ProjectService) get_project_sample_tags(
	project uuid.UUID,
	sample uuid.UUID,
) ([]string, error) {
	tags_query := "SELECT _tag FROM project_sample_tag_ WHERE _project=$1 AND _sample=$2"
	rows, _ := s.db.Conn.Query(s.ctx, tags_query, project, sample)
	tags, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var tag string
		err := row.Scan(&tag)
		return tag, err
	})
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
			"sample", sample,
		).Error("could not get project sample tags")
		return nil, err
	}

	return tags, nil
}

func (s *ProjectService) get_project_sample_resources_sample_notes(
	project_id uuid.UUID,
	sample_id uuid.UUID,
) ([]ProjectSampleNote, error) {
	notes_query :=
		`SELECT _id, _sample, _project, _creator, timestamp, content
		FROM project_sample_note_ WHERE _project=$1 AND _sample=$2`
	rows, _ := s.db.Conn.Query(s.ctx, notes_query, project_id, sample_id)
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
		s.logger.With(
			"error", err,
			"query", notes_query,
		).Error("could not get project sampel notes")
		return nil, err
	}

	return notes, nil
}

func (s *ProjectService) get_project_sample_resources_sample_data(
	sample_id uuid.UUID,
) ([]SampleData, error) {
	data_query := "SELECT _id, _sample, _schema, _creator, timestamp FROM sample_data_ WHERE _sample=$1"
	rows, _ := s.db.Conn.Query(s.ctx, data_query, sample_id)
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

func (s *ProjectService) get_project_sample_resources_sample_user_permissions(
	sample_id uuid.UUID,
) ([]SampleUserPermissions, error) {
	permission_query :=
		`SELECT _user, _permission FROM sample_user_permission_ 
		WHERE _sample=$1`
	rows, _ := s.db.Conn.Query(s.ctx, permission_query, sample_id)
	var permissions []SampleUserPermissions
	for rows.Next() {
		var user uuid.UUID
		var permission SampleUserPermission
		err := rows.Scan(&user, &permission)
		if err != nil {
			s.logger.With("error", err).Error("could not get sample user permissions")
			return nil, err
		}

		idx := slices.IndexFunc(permissions, func(user_perm SampleUserPermissions) bool {
			return user_perm.User == user
		})
		if idx < 0 {
			permissions = append(permissions, SampleUserPermissions{User: user, Permissions: []SampleUserPermission{permission}})
		} else {
			permissions[idx].Permissions = append(permissions[idx].Permissions, permission)
		}

	}

	return permissions, nil
}

func (s *ProjectService) get_project_sample_resources_project_sample_user_permissions(
	project_id uuid.UUID,
	sample_id uuid.UUID,
) ([]ProjectSampleUserPermissions, error) {
	query :=
		`SELECT _user, _permission FROM project_sample_user_permission_ 
		WHERE _project=$1 AND _sample=$2`
	rows, _ := s.db.Conn.Query(
		s.ctx,
		query,
		project_id,
		sample_id,
	)

	var permissions []ProjectSampleUserPermissions
	for rows.Next() {
		var user uuid.UUID
		var permission ProjectSampleUserPermission
		err := rows.Scan(&user, &permission)
		if err != nil {
			s.logger.With("error", err).Error("could not get sample user permissions")
			return nil, err
		}

		idx := slices.IndexFunc(permissions, func(user_perm ProjectSampleUserPermissions) bool {
			return user_perm.User == user
		})
		if idx < 0 {
			permissions = append(
				permissions,
				ProjectSampleUserPermissions{User: user, Permissions: []ProjectSampleUserPermission{permission}},
			)
		} else {
			permissions[idx].Permissions = append(permissions[idx].Permissions, permission)
		}

	}

	return permissions, nil
}

type ProjectSampleNoteUpdate struct {
	Id      uuid.UUID
	Editor  uuid.UUID
	Content string
}

type ProjectSampleUpdate struct {
	Id               uuid.UUID
	Label            string
	Tags             []string
	PropertiesUpsert []Property
	PropertiesRemove []string
	NotesNew         []ProjectSampleNoteCreate
	NotesUpdate      []ProjectSampleNoteUpdate
	NotesRemove      []uuid.UUID
}

func (s *ProjectService) UpdateProjectSample(
	project uuid.UUID,
	update ProjectSampleUpdate,
) error {
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return err
	}
	defer tx.Rollback(s.ctx)

	err = s.update_project_sample_label(tx, project, update.Id, update.Label)
	if err != nil {
		return err
	}

	err = s.update_project_sample_tags(tx, project, update.Id, update.Tags)
	if err != nil {
		return err
	}

	// NB: Remove properties first incase a new one with
	// the same key but a different type is inserted.
	err = s.update_project_sample_remove_sample_properties(
		tx,
		update.Id,
		update.PropertiesRemove,
	)
	if err != nil {
		return err
	}

	err = s.update_project_sample_upsert_sample_properties(
		tx,
		update.Id,
		update.PropertiesUpsert,
	)
	if err != nil {
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
			"update", update,
		).Error("could not commit project sample update")
		return err
	}

	return nil
}

func (s *ProjectService) update_project_sample_label(
	tx pgx.Tx,
	project uuid.UUID,
	sample uuid.UUID,
	label string,
) error {
	query :=
		`UPDATE project_sample_membership_ SET label=$3 
		WHERE _project=$1 AND _sample=$2`
	_, err := tx.Exec(
		s.ctx,
		query,
		project,
		sample,
		label,
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
			"sample", sample,
			"label", label,
		).Error("could not update project sample label")
		return err
	}

	return nil
}

func (s *ProjectService) update_project_sample_tags(
	tx pgx.Tx,
	project uuid.UUID,
	sample uuid.UUID,
	tags []string,
) error {
	current_tags, err := s.get_project_sample_tags(project, sample)
	if err != nil {
		return err
	}

	add_tags := []string{}
	for _, tag := range tags {
		if !slices.Contains(current_tags, tag) {
			add_tags = append(add_tags, tag)
		}

	}

	remove_tags := []string{}
	for _, tag := range current_tags {
		if !slices.Contains(tags, tag) {
			remove_tags = append(remove_tags, tag)
		}
	}

	add_args := make([]any, len(add_tags)+2)
	add_args[0] = project
	add_args[1] = sample
	var add_query strings.Builder
	add_query.WriteString("INSERT INTO project_sample_tag_ (_project, _sample, _tag) VALUES ")
	for idx, tag := range add_tags {
		if idx > 0 {
			add_query.WriteString(", ")
		}

		idx_args := idx + 2
		fmt.Fprintf(&add_query, "($1, $2, $%d)", idx_args+1)
		add_args[idx_args] = tag
	}

	remove_args := make([]any, len(remove_tags)+2)
	remove_args[0] = project
	remove_args[1] = sample
	var remove_query strings.Builder
	remove_query.WriteString("DELETE FROM project_sample_tag_ WHERE ")
	for idx, tag := range remove_tags {
		if idx > 0 {
			remove_query.WriteString(" OR ")
		}

		idx_args := idx + 2
		fmt.Fprintf(&remove_query, "(_project=$1 AND _sample=$2 AND _tag=$%d)", idx_args+1)
		remove_args[idx_args] = tag
	}

	if len(add_tags) > 0 {
		_, err = tx.Exec(s.ctx, add_query.String(), add_args...)
		if err != nil {
			s.logger.With(
				"error", err,
				"project", project,
				"sample", sample,
				"tags", add_tags,
				"q", add_query.String(),
			).Error("could not add tags to project sample")
			return err
		}
	}
	if len(remove_tags) > 0 {
		_, err = tx.Exec(s.ctx, remove_query.String(), remove_args...)
		if err != nil {
			s.logger.With(
				"error", err,
				"project", project,
				"sample", sample,
				"tags", remove_tags,
			).Error("could not remove tags from project sample")
			return err
		}
	}

	return nil
}

func (s *ProjectService) update_project_sample_upsert_sample_properties(
	tx pgx.Tx,
	sample uuid.UUID,
	properties []Property,
) error {
	if len(properties) == 0 {
		return nil
	}

	properties_query := "SELECT _key, _type, value FROM sample_property_ WHERE _sample=$1"
	rows, _ := tx.Query(s.ctx, properties_query, sample)
	current_props, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Property, error) {
		var prop_val Property
		err := row.Scan(&prop_val.Key, &prop_val.Type, &prop_val.Value)
		return prop_val, err
	})
	if err != nil {
		s.logger.With(
			"error", err,
			"sample", sample,
		).Error("could not get sample properties")
		return err
	}

	var insert_idx []int
	var update_idx []int
	for idx, prop := range properties {
		curr_idx := slices.IndexFunc(current_props, func(curr_prop Property) bool {
			return prop.Key == curr_prop.Key
		})

		if curr_idx < 0 {
			insert_idx = append(insert_idx, idx)
		} else {
			curr_prop := current_props[curr_idx]
			if prop.Type != curr_prop.Type {
				s.logger.With(
					"sample", sample,
					"property", prop.Key,
					"stored_type", curr_prop.Type,
					"update_type", prop.Type,
				).Error("incompatible property type, can not update")
				return fmt.Errorf("incompatible property type {sample: %s, key: %s}", sample, prop.Key)
			}
			if prop.Value != curr_prop.Value {
				update_idx = append(update_idx, idx)
			}
		}
	}

	update_args := make([]any, len(update_idx)*2+1)
	update_args[0] = sample
	var update_query strings.Builder
	update_query.WriteString("UPDATE sample_property_ as t SET value=u.value FROM (VALUES ")
	for idx, prop_idx := range update_idx {
		if idx > 0 {
			update_query.WriteString(", ")
		}

		key_idx := 2*idx + 1
		value_idx := key_idx + 1
		fmt.Fprintf(&update_query, "($1::uuid, $%d::text, $%d::jsonb)", key_idx+1, value_idx+1)

		property := properties[prop_idx]
		property_value, err := json.Marshal(property.Value)
		if err != nil {
			s.logger.With(
				"error", err,
				"key", property.Key,
				"value", property.Value,
			).Error("could not serialize property")
			return err
		}

		update_args[key_idx] = property.Key
		update_args[value_idx] = property_value
	}
	update_query.WriteString(
		") AS u(sample, key, value) WHERE t._sample=u.sample AND t._key=u.key",
	)

	insert_args := make([]any, len(insert_idx)*3+1)
	insert_args[0] = sample
	var insert_query strings.Builder
	insert_query.WriteString("INSERT INTO sample_property_ (_sample, _key, _type, value) VALUES ")
	for idx, prop_idx := range insert_idx {
		if idx > 0 {
			insert_query.WriteString(", ")
		}

		key_idx := 3*idx + 1
		type_idx := key_idx + 1
		value_idx := type_idx + 1
		fmt.Fprintf(&insert_query, "($1, $%d, $%d, $%d)", key_idx+1, type_idx+1, value_idx+1)

		property := properties[prop_idx]
		property_value, err := json.Marshal(property.Value)
		if err != nil {
			s.logger.With(
				"error", err,
				"key", property.Key,
				"value", property.Value,
			).Error("could not serialize property")
			return err
		}

		insert_args[key_idx] = property.Key
		insert_args[type_idx] = property.Type
		insert_args[value_idx] = property_value
	}

	if len(update_idx) > 0 {
		_, err = tx.Exec(s.ctx, update_query.String(), update_args...)
		if err != nil {
			update_properties := make([]Property, len(update_idx))
			for idx, prop_idx := range update_idx {
				update_properties[idx] = properties[prop_idx]
			}
			s.logger.With(
				"error", err,
				"properties", update_properties,
				"query", update_query.String(),
			).Error("could not update sample properties")
			return err
		}
	}

	if len(insert_idx) > 0 {
		_, err = tx.Exec(s.ctx, insert_query.String(), insert_args...)
		if err != nil {
			insert_properties := make([]Property, len(insert_idx))
			for idx, prop_idx := range insert_idx {
				insert_properties[idx] = properties[prop_idx]
			}
			s.logger.With(
				"error", err,
				"properties", insert_properties,
			).Error("could not insert sample properties")
			return err
		}
	}

	return nil
}

func (s *ProjectService) update_project_sample_remove_sample_properties(
	tx pgx.Tx,
	sample uuid.UUID,
	properties []string,
) error {
	if len(properties) == 0 {
		return nil
	}

	query := "DELETE FROM sample_property_ WHERE _sample=$1 AND _key=ANY($2)"
	_, err := tx.Exec(s.ctx, query, sample, properties)
	if err != nil {
		s.logger.With(
			"error", err,
			"sample", sample,
			"properties", properties,
		).Error("could not remove properties")
		return err
	}

	return nil
}

type ProjectSampleMembership struct {
	Project   uuid.UUID
	Sample    uuid.UUID
	Creator   uuid.UUID
	Timestamp time.Time
	Label     string
}

func (s *ProjectService) GetProjectSampleMembershipsByProject(project uuid.UUID) ([]ProjectSampleMembership, error) {
	query := "SELECT _project, _sample, _creator, _timestamp, label FROM project_sample_membership_ WHERE _project=$1"
	rows, _ := s.db.Conn.Query(s.ctx, query, project)
	memberships, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ProjectSampleMembership, error) {
		var membership ProjectSampleMembership
		err := row.Scan(
			&membership.Project,
			&membership.Sample,
			&membership.Creator,
			&membership.Timestamp,
			&membership.Label,
		)
		return membership, err
	})
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
		).Error("could not get project sample memberships")
		return nil, err
	}

	return memberships, nil
}
