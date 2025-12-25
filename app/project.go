package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	PROJECT_PUBLIC  = ProjectVisibility("public")
	PROJECT_PRIVATE = ProjectVisibility("private")
)

type IsoTimestamp string

type ProjectVisibility string

const (
	PROJECT_USER_PERMISSION_READ       = ProjectUserPermission("read")
	PROJECT_USER_PERMISSION_READ_WRITE = ProjectUserPermission("read_write")
	PROJECT_USER_PERMISSION_ADMIN      = ProjectUserPermission("admin")
	PROJECT_USER_PERMISSION_OWNER      = ProjectUserPermission("owner")
)

type ProjectUserPermission string

type Project struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Label       string
	Description string
	Visibility  ProjectVisibility
}

func (app *App) GetUserProjects() ([]Project, error) {
	if app.db_pool.err != nil {
		return nil, app.db_pool.err
	}
	if app.state.user_id == uuid.Nil {
		return nil, &UserNotAuthenticedError{}
	}

	user_project_query := "SELECT _id, _creator, label, description, visibility FROM project_ WHERE _creator=$1 ORDER BY _id"
	rows, _ := app.db_pool.ok.Query(app.ctx, user_project_query, app.state.user_id)
	user_projects, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Project, error) {
		var project Project
		err := row.Scan(&project.Id, &project.Creator, &project.Label, &project.Description, &project.Visibility)
		return project, err
	})
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not collect user projects: %v", err)
		return nil, err
	}

	return user_projects, nil
}

type ProjectCreate struct {
	Label       string
	Description string
	Visibility  ProjectVisibility
}

func (app *App) CreateProject(project ProjectCreate) (uuid.UUID, error) {
	if app.db_pool.err != nil {
		return uuid.Nil, app.db_pool.err
	}
	if app.state.user_id == uuid.Nil {
		return uuid.Nil, &UserNotAuthenticedError{}
	}

	tx, err := app.db_pool.ok.Begin(app.ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(app.ctx)

	var project_id uuid.UUID
	create_project_query := "INSERT INTO project_ (_creator, label, description, visibility) VALUES ($1, $2, $3, $4) RETURNING _id"
	err = tx.QueryRow(app.ctx, create_project_query, app.state.user_id, project.Label, project.Description, project.Visibility).Scan(&project_id)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not create project: %v", err)
		return uuid.Nil, err
	}

	set_user_permission_query := "INSERT INTO project_user_permission_ (_project, _user, permission) VALUES ($1, $2, $3)"
	_, err = tx.Exec(app.ctx, set_user_permission_query, project_id, app.state.user_id, PROJECT_USER_PERMISSION_OWNER)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not create user project permission: %v", err)
		return uuid.Nil, err
	}

	err = tx.Commit(app.ctx)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not create project: %v", err)
		return uuid.Nil, err
	}

	return project_id, nil
}

type PropertyType string

const (
	PROPERTY_TYPE_STRING   = PropertyType("string")
	PROPERTY_TYPE_BOOL     = PropertyType("bool")
	PROPERTY_TYPE_UINT     = PropertyType("uint")
	PROPERTY_TYPE_INT      = PropertyType("int")
	PROPERTY_TYPE_FLOAT    = PropertyType("float")
	PROPERTY_TYPE_QUANTITY = PropertyType("quantity")
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

func (app *App) GetProjectResources(project_id uuid.UUID) (ProjectResources, error) {
	if app.db_pool.err != nil {
		return ProjectResources{}, app.db_pool.err
	}
	if app.state.user_id == uuid.Nil {
		return ProjectResources{}, &UserNotAuthenticedError{}
	}

	var project_resources ProjectResources
	project_query := "SELECT _id, _creator, label, description, visibility FROM project_ WHERE _id=$1"
	err := app.db_pool.ok.QueryRow(
		app.ctx,
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
		runtime.LogErrorf(app.ctx, "could not get project: %v", err)
		return ProjectResources{}, err
	}

	project_user_permission_query := "SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	err = app.db_pool.ok.QueryRow(app.ctx, project_user_permission_query, project_id, app.state.user_id).Scan(&project_resources.ProjectUserPermission)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not get project user permission: %v", err)
		return ProjectResources{}, err
	}

	project_tags_query := "SELECT _tag FROM project_tag_ WHERE _project=$1"
	project_tag_rows, _ := app.db_pool.ok.Query(app.ctx, project_tags_query, project_id)
	project_resources.ProjectTags, err = pgx.CollectRows(project_tag_rows, func(row pgx.CollectableRow) (string, error) {
		var tag string
		err := row.Scan(&tag)
		return tag, err
	})
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not get project tags: %v", err)
	}

	project_note_count_query := "SELECT COUNT(*) FROM project_note_ WHERE _project=$1"
	err = app.db_pool.ok.QueryRow(app.ctx, project_note_count_query, project_id).Scan(&project_resources.ProjectNoteCount)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not get project note count: %v", err)
	}

	project_sample_membership_query := `
		SELECT _sample, _creator, _timestamp, label 
		FROM project_sample_membership_ 
		WHERE _project=$1
	`
	project_sample_membership_rows, _ := app.db_pool.ok.Query(
		app.ctx,
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
		runtime.LogErrorf(app.ctx, "could not get project sample memberships: %v", err)
		project_resources.Samples = []ProjectSample{}
	}

	sample_tags_query := "SELECT _tag FROM project_sample_tag_ WHERE _project=$1 AND _sample=$2"
	sample_properties_query := "SELECT _key, _type, value FROM sample_property_ WHERE _sample=$1"
	sample_note_count_query := "SELECT COUNT(*) FROM project_sample_note_ WHERE _sample=$1"
	for i := range project_resources.Samples {
		sample_id := project_resources.Samples[i].Id
		sample_tag_rows, _ := app.db_pool.ok.Query(app.ctx, sample_tags_query, project_id, sample_id)
		project_resources.Samples[i].Tags, err = pgx.CollectRows(sample_tag_rows, func(row pgx.CollectableRow) (string, error) {
			var tag string
			err := row.Scan(&tag)
			return tag, err
		})
		if err != nil {
			runtime.LogErrorf(app.ctx, "could not get project sample tags of `%s` in project `%s`: %v", sample_id, project_id, err)
		}

		sample_properties_rows, _ := app.db_pool.ok.Query(app.ctx, sample_properties_query, sample_id)
		project_resources.Samples[i].Properties, err = pgx.CollectRows(sample_properties_rows, func(row pgx.CollectableRow) (Property, error) {
			var property Property
			err := row.Scan(&property.Key, &property.Type, &property.Value)
			return property, err
		})
		if err != nil {
			runtime.LogErrorf(app.ctx, "could not get sample properties of `%s`: %v", sample_id, err)
		}

		err = app.db_pool.ok.QueryRow(app.ctx, sample_note_count_query, sample_id).Scan(&project_resources.Samples[i].NoteCount)
		if err != nil {
			runtime.LogErrorf(app.ctx, "could not get sample properties of `%s`: %v", sample_id, err)
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

func (app *App) GetProjectWithUserPermission(project_id uuid.UUID) (ProjectWithUserPermission, error) {
	if app.db_pool.err != nil {
		return ProjectWithUserPermission{}, app.db_pool.err
	}
	if app.state.user_id == uuid.Nil {
		return ProjectWithUserPermission{}, &UserNotAuthenticedError{}
	}

	var project ProjectWithUserPermission
	project_query := "SELECT _id, _creator, label, description, visibility FROM project_ WHERE _id=$1"
	err := app.db_pool.ok.QueryRow(
		app.ctx,
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
		runtime.LogErrorf(app.ctx, "could not get project: %v", err)
		return ProjectWithUserPermission{}, err
	}

	project_user_permission_query := "SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	err = app.db_pool.ok.QueryRow(app.ctx, project_user_permission_query, project_id, app.state.user_id).Scan(&project.UserPermission)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not get project user permission: %v", err)
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

func (app *App) CreateProjectSamples(project uuid.UUID, samples []ProjectSampleCreate) (Ok, error) {
	if app.db_pool.err != nil {
		return Ok{}, app.db_pool.err
	}
	if app.state.user_id == uuid.Nil {
		return Ok{}, &UserNotAuthenticedError{}
	}

	user_permission_query := "SELECT permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	var user_permission ProjectUserPermission
	err := app.db_pool.ok.QueryRow(
		app.ctx,
		user_permission_query,
		project.String(),
		app.state.user_id.String(),
	).Scan(&user_permission)
	if err != nil ||
		(user_permission != PROJECT_USER_PERMISSION_OWNER &&
			user_permission != PROJECT_USER_PERMISSION_ADMIN &&
			user_permission != PROJECT_USER_PERMISSION_READ_WRITE) {
		runtime.LogDebugf(
			app.ctx,
			"insufficient permissions to create samples for user %v on project %v",
			project,
			app.state.user_id,
		)
		return Ok{}, &InsufficientPermissionsError{}
	}

	if len(samples) == 0 {
		return Ok{}, nil
	}

	tx, err := app.db_pool.ok.Begin(app.ctx)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not begin transaction: %v", err)
		return Ok{}, err
	}
	defer tx.Rollback(app.ctx)

	var sample_create_query strings.Builder
	sample_create_query.WriteString("INSERT INTO sample_ (_creator) VALUES ")
	for idx := range samples {
		if idx > 0 {
			fmt.Fprintf(&sample_create_query, ", ")
		}

		fmt.Fprintf(&sample_create_query, "('%s')", app.state.user_id)
	}
	sample_create_query.WriteString(" RETURNING _id")
	create_rows, err := tx.Query(app.ctx, sample_create_query.String())
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not create samples: %v", err)
		return Ok{}, err
	}

	sample_ids, err := pgx.CollectRows(create_rows, func(row pgx.CollectableRow) (uuid.UUID, error) {
		var id uuid.UUID
		err := row.Scan(&id)
		return id, err
	})
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not collect user projects: %v", err)
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
			app.state.user_id,
			idx+1,
		)
		labels[idx] = samples[idx].Label
	}
	_, err = tx.Exec(app.ctx, project_membership_query.String(), labels...)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not create project sample memberships: %v", err)
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
		_, err = tx.Exec(app.ctx, sample_tags_query.String(), tags...)
		if err != nil {
			runtime.LogErrorf(app.ctx, "could not create project sample tags: %v", err)
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
					runtime.LogErrorf(
						app.ctx,
						"could not serialize property %s with value %v: %v",
						property.Key,
						property.Value,
						err,
					)
					return Ok{}, err
				}

				property_values[key_idx] = property.Key
				property_values[type_idx] = property.Type
				property_values[value_idx] = property_value
				pidx += NUM_PROPERTY_VALUES
			}
		}
		_, err = tx.Exec(app.ctx, sample_properties_query.String(), property_values...)
		if err != nil {
			runtime.LogErrorf(app.ctx, "could not create sample properties: %v", err)
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
		_, err = tx.Exec(app.ctx, sample_notes_query.String(), note_values...)
		if err != nil {
			runtime.LogErrorf(app.ctx, "could not create sample notes: %v", err)
			return Ok{}, err
		}
	}

	err = tx.Commit(app.ctx)
	if err != nil {
		return Ok{}, err
	}

	return Ok{}, nil

}
