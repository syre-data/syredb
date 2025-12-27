package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type IsoTimestamp string

const (
	USER_ROLE_OWNER = UserRole("owner")
	USER_ROLE_ADMIN = UserRole("admin")
	USER_ROLE_USER  = UserRole("user")
)

type UserRole string

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
	ctx       context.Context
	logger    *slog.Logger
	db        *DbConnection
	app_state *AppState
}

func NewProjectService(
	logger *slog.Logger,
	db *DbConnection,
	app_state *AppState,
) *ProjectService {
	return &ProjectService{logger: logger, db: db, app_state: app_state}
}

func (s *ProjectService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

type Project struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Label       string
	Description string
	Visibility  ProjectVisibility
}

func (s *ProjectService) GetUserProjects() ([]Project, error) {
	s.app_state._lock.RLock()
	user_id := s.app_state.user_id
	s.app_state._lock.RUnlock()

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
	s.app_state._lock.RLock()
	user_id := s.app_state.user_id
	s.app_state._lock.RUnlock()

	if user_id == uuid.Nil {
		return uuid.Nil, &UserNotAuthenticatedError{}
	}

	tx, err := s.db.conn.Begin(s.ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(s.ctx)

	var project_id uuid.UUID
	create_project_query := "INSERT INTO project_ (_creator, label, description, visibility) VALUES ($1, $2, $3, $4) RETURNING _id"
	err = tx.QueryRow(s.ctx, create_project_query, user_id, project.Label, project.Description, project.Visibility).Scan(&project_id)
	if err != nil {
		s.logger.With("error", err).Error("could not create project")
		return uuid.Nil, err
	}

	set_user_permission_query := "INSERT INTO project_user_permission_ (_project, _user, permission) VALUES ($1, $2, $3)"
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
	SampleGroups          []ProjectSampleGroup
	SampleGroupRelations  []SampleGroupRelation
	ProjectNoteCount      uint
	ProjectUserPermission ProjectUserPermission
}

func (s *ProjectService) GetProjectResources(project_id uuid.UUID) (ProjectResources, error) {
	s.app_state._lock.RLock()
	user_id := s.app_state.user_id
	s.app_state._lock.RUnlock()

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
	s.app_state._lock.RLock()
	user_id := s.app_state.user_id
	s.app_state._lock.RUnlock()
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

type ProjectSampleNoteCreate struct {
	Timestamp IsoTimestamp
	Content   string
}

type ProjectSampleCreate struct {
	Label      string
	Tags       []string
	Properties []Property
	Notes      []ProjectSampleNoteCreate
}

func (s *ProjectService) CreateProjectSamples(project uuid.UUID, samples []ProjectSampleCreate) (Ok, error) {
	s.app_state._lock.RLock()
	user_id := s.app_state.user_id
	s.app_state._lock.RUnlock()

	if user_id == uuid.Nil {
		return Ok{}, &UserNotAuthenticatedError{}
	}

	user_permission_query := "SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	var user_permission ProjectUserPermission
	err := s.db.conn.QueryRow(
		s.ctx,
		user_permission_query,
		project.String(),
		user_id.String(),
	).Scan(&user_permission)
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

	tx, err := s.db.conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return Ok{}, err
	}
	defer tx.Rollback(s.ctx)

	var sample_create_query strings.Builder
	sample_create_query.WriteString("INSERT INTO sample_ (_creator) VALUES ")
	for idx := range samples {
		if idx > 0 {
			fmt.Fprintf(&sample_create_query, ", ")
		}

		fmt.Fprintf(&sample_create_query, "('%s')", user_id)
	}
	sample_create_query.WriteString(" RETURNING _id")
	create_rows, err := tx.Query(s.ctx, sample_create_query.String())
	if err != nil {
		s.logger.With("error", err).Error("could not create samples")
		return Ok{}, err
	}

	sample_ids, err := pgx.CollectRows(create_rows, func(row pgx.CollectableRow) (uuid.UUID, error) {
		var id uuid.UUID
		err := row.Scan(&id)
		return id, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not collect user projects")
		return Ok{}, err
	}

	labels := make([]any, len(samples))
	var project_membership_query strings.Builder
	project_membership_query.WriteString("INSERT INTO project_sample_membership_ (_project, _sample, _creator, label) VALUES ")
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
	_, err = tx.Exec(s.ctx, project_membership_query.String(), labels...)
	if err != nil {
		s.logger.With("error", err).Error("could not create project sample memberships")
		return Ok{}, err
	}

	num_tags := 0
	for _, sample := range samples {
		num_tags += len(sample.Tags)
	}
	if num_tags > 0 {
		tags := make([]any, num_tags)
		tidx := 0
		var sample_tags_query strings.Builder
		sample_tags_query.WriteString("INSERT INTO project_sample_tag_ (_sample, _project, _tag) VALUES ")
		for idx, sample_id := range sample_ids {
			for _, tag := range samples[idx].Tags {
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
		_, err = tx.Exec(s.ctx, sample_tags_query.String(), tags...)
		if err != nil {
			s.logger.With("error", err).Error("could not create project sample tags")
			return Ok{}, err
		}
	}

	const NUM_PROPERTY_VALUES = 3
	num_properties := 0
	for _, sample := range samples {
		num_properties += len(sample.Properties)
	}
	if num_properties > 0 {
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
					return Ok{}, err
				}

				property_values[key_idx] = property.Key
				property_values[type_idx] = property.Type
				property_values[value_idx] = property_value
				pidx += NUM_PROPERTY_VALUES
			}
		}
		_, err = tx.Exec(s.ctx, sample_properties_query.String(), property_values...)
		if err != nil {
			s.logger.With("error", err).Error("could not create sample properties")
			return Ok{}, err
		}
	}

	const NUM_NOTE_VALUES = 2
	num_notes := 0
	for _, sample := range samples {
		num_notes += len(sample.Notes)
	}
	if num_notes > 0 {
		note_values := make([]any, num_notes*NUM_NOTE_VALUES)
		nidx := 0
		var sample_notes_query strings.Builder
		sample_notes_query.WriteString("INSERT INTO project_sample_note_ (_project, _sample, timestamp, content) VALUES ")
		for sidx, sample := range samples {
			for _, note := range sample.Notes {
				if nidx > 0 {
					fmt.Fprint(&sample_notes_query, ", ")
				}

				timestamp_idx := nidx
				note_idx := timestamp_idx + 1
				fmt.Fprintf(
					&sample_notes_query,
					"('%s', '%s', $%d, $%d)",
					project,
					sample_ids[sidx],
					timestamp_idx+1,
					note_idx+1,
				)

				note_values[timestamp_idx] = note.Timestamp
				note_values[note_idx] = note.Content
				nidx += NUM_NOTE_VALUES
			}
		}
		_, err = tx.Exec(s.ctx, sample_notes_query.String(), note_values...)
		if err != nil {
			s.logger.With("error", err).Error("could not create sample notes")
			return Ok{}, err
		}
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		return Ok{}, err
	}

	return Ok{}, nil
}
