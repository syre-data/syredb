package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"syredb/database"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProjectPermission string

const (
	ProjectPermissionOwner           ProjectPermission = "owner"
	ProjectPermissionRead            ProjectPermission = "read"
	ProjectPermissionDataCreate      ProjectPermission = "data_create"
	ProjectPermissionDataGroupCreate ProjectPermission = "data_group_create"
)

type ProjectSamplePermission string

const (
	ProjectSamplePermissionModifyLabel      ProjectSamplePermission = "modify_label"
	ProjectSamplePermissionModifyTags       ProjectSamplePermission = "modify_tags"
	ProjectSamplePermissionModifyProperties ProjectSamplePermission = "modify_properties"
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
	Id          uuid.UUID  `db:"_id"`
	Creator     uuid.UUID  `db:"_creator"`
	Label       string     `db:"label"`
	Description string     `db:"description"`
	Visibility  Visibility `db:"visibility"`
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
		`INSERT INTO project_user_permission_ (_project, _user, _permission) 
		VALUES ($1, $2, $3)`
	_, err = tx.Exec(
		s.ctx,
		set_user_permission_query,
		project_id,
		user_id,
		ProjectPermissionOwner,
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

func (s *ProjectService) ProjectUserPermissions(
	project uuid.UUID,
	user uuid.UUID,
) ([]ProjectPermission, error) {
	query := "SELECT _permission FROM project_user_permission_ WHERE _project=$1 AND _user=$2"
	rows, _ := s.db.Conn.Query(
		s.ctx,
		query,
		project,
		user,
	)
	permissions, err := pgx.CollectRows(rows, pgx.RowTo[ProjectPermission])
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.With(
				"error", err,
				"user", user,
				"project", project,
			).Error("could not retrieve project user permissions")
		}
		return nil, err
	}

	return permissions, nil
}

func (s *ProjectService) UserHasProjectPermission(
	needle ProjectPermission,
	user uuid.UUID,
	project uuid.UUID,
) (bool, error) {
	permissions, err := s.ProjectUserPermissions(project, user)
	if err != nil {
		return false, err
	}

	sufficient := slices.Contains(permissions, ProjectPermissionOwner) ||
		slices.Contains(permissions, needle)

	return sufficient, nil
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
	Key   string       `db:"_key"`
	Type  PropertyType `db:"_type"`
	Value any          `db:"value"` // TODO: Match value with type
}

type ProjectSampleNote struct {
	Id        uuid.UUID
	Sample    uuid.UUID
	Project   uuid.UUID
	Creator   uuid.UUID
	Timestamp time.Time
	Content   string
}

type ProjectData struct {
	Id                uuid.UUID
	Type              uuid.UUID
	Creator           DataCreator
	MembershipCreator uuid.UUID
	Timestamp         time.Time
	Label             *string
	Tags              []string
	Properties        []Property
	NoteCount         uint
}

type DataRx struct {
	Id          uuid.UUID       `db:"_id"`
	Type        uuid.UUID       `db:"_type"`
	CreatorType DataCreatorType `db:"_creator_type"`
	Timestamp   time.Time       `db:"timestamp"`
	Visibility  Visibility      `db:"visibility"`
}

type DerivedData struct {
	Parent     uuid.UUID
	Transform  uuid.UUID
	SampleData uuid.UUID
	Schema     uuid.UUID
}

type ProjectDataGroup struct {
	Id          uuid.UUID
	Creator     uuid.UUID
	Label       string
	Description string
	Properties  []Property
	Samples     []uuid.UUID
}

type DataGroupRelation struct {
	Parent uuid.UUID
	Child  uuid.UUID
}

type ProjectResources struct {
	Project            Project
	Tags               []string
	Data               []ProjectData
	DataTypes          []DataType
	DataSchemas        []DataSchema
	DataGroups         []ProjectDataGroup
	DataGroupRelations []DataGroupRelation
	ProjectNoteCount   uint
	ProjectPermissions []ProjectPermission
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

	project_resources.ProjectPermissions, err = s.ProjectUserPermissions(project_id, user_id)
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get project user permission")
		return ProjectResources{}, err
	}

	project_tags_query := "SELECT _tag FROM project_tag_ WHERE _project=$1"
	rows, _ := s.db.Conn.Query(s.ctx, project_tags_query, project_id)
	project_resources.Tags, err = pgx.CollectRows(
		rows,
		pgx.RowTo[string],
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

	project_data_membership_query :=
		`SELECT _project, _data, _creator, label
		FROM project_data_membership_ 
		WHERE _project=$1`
	rows, _ = s.db.Conn.Query(
		s.ctx,
		project_data_membership_query,
		project_id,
	)
	project_data, err := pgx.CollectRows(
		rows,
		pgx.RowToStructByName[ProjectDataMembershipRx],
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project_id,
		).Error("could not get project data memberships")
	}

	data_ids := make([]uuid.UUID, len(project_data))
	for idx, data := range project_data {
		data_ids[idx] = data.Data
	}
	data_info, err := s.getProjectResourcesDataInfo(user_id, project_id, data_ids)
	if err != nil {
		return ProjectResources{}, err
	}
	data_type_ids := make([]uuid.UUID, 0, len(data_info))
	project_resources.Data = make([]ProjectData, len(data_info))
	for idx, info := range data_info {
		pdidx := slices.IndexFunc(project_data, func(data ProjectDataMembershipRx) bool {
			return data.Data == info.Id
		})
		if pdidx < 0 {
			s.logger.With("data", info.Id).Error("could not find project data")
			panic("could not find project data")
		}

		project_resources.Data[idx].Id = info.Id
		project_resources.Data[idx].Type = info.Type
		project_resources.Data[idx].Creator = info.Creator
		project_resources.Data[idx].MembershipCreator = project_data[pdidx].Creator
		project_resources.Data[idx].Timestamp = info.Timestamp
		project_resources.Data[idx].Label = project_data[pdidx].Label
		project_resources.Data[idx].Tags = info.Tags
		project_resources.Data[idx].Properties = info.Properties
		project_resources.Data[idx].NoteCount = info.NoteCount

		data_type_ids = append(data_type_ids, info.Type)
	}

	project_resources.DataTypes, err = s.data_service.DataTypesById(data_type_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"data types", data_type_ids,
		).Error("could not get data types")
		return ProjectResources{}, err
	}

	data_schema_ids := []uuid.UUID{}
	// TODO: Get relevent data schemas
	project_resources.DataSchemas, err = s.data_service.DataSchemasById(data_schema_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"schemas", data_schema_ids,
		).Error("could not get data schemas")
		return ProjectResources{}, err
	}

	return project_resources, nil
}

type ProjectDataInfo struct {
	Id         uuid.UUID
	Type       uuid.UUID
	Creator    DataCreator
	Timestamp  time.Time
	Tags       []string
	Properties []Property
	NoteCount  uint
}

func (s *ProjectService) getProjectResourcesDataInfo(
	user_id uuid.UUID,
	project_id uuid.UUID,
	data_ids []uuid.UUID,
) ([]ProjectDataInfo, error) {
	info := make([]ProjectDataInfo, len(data_ids))
	for idx, data_id := range data_ids {
		info[idx].Id = data_id
	}
	info_query := "SELECT _id, _type, _creator_type, timestamp FROM data_ WHERE _id=ANY($1)"
	rows, err := s.db.Conn.Query(s.ctx, info_query, data_ids)
	if err != nil {
		s.logger.With("error", err).Error("could not query data")
		return nil, err
	}
	creator_user_ids := make([]uuid.UUID, 0, len(data_ids))
	creator_transform_ids := make([]uuid.UUID, 0, len(data_ids))
	for rows.Next() {
		var data_id uuid.UUID
		var data_type uuid.UUID
		var creator_type DataCreatorType
		var timestamp time.Time
		err := rows.Scan(&data_id, &data_type, &creator_type, &timestamp)
		if err != nil {
			s.logger.With("error", err).Error("could not get data info")
			return nil, err
		}

		data_info_idx := slices.IndexFunc(info, func(data_info ProjectDataInfo) bool {
			return data_info.Id == data_id
		})
		if data_info_idx < 0 {
			s.logger.With("data", data_id).Error("could not find data")
			panic("could not find data")
		}

		info[data_info_idx].Type = data_type
		info[data_info_idx].Timestamp = timestamp

		switch creator_type {
		case DataCreatorTypeTransform:
			creator_transform_ids = append(creator_transform_ids, data_id)
		case DataCreatorTypeUser:
			creator_user_ids = append(creator_user_ids, data_id)
		}
	}

	creator_user_query :=
		`SELECT _data, _creator FROM data_creator_user_ 
		WHERE _data=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, creator_user_query, creator_user_ids)
	for rows.Next() {
		var data_id uuid.UUID
		var creator_id uuid.UUID
		err := rows.Scan(&data_id, &creator_id)
		if err != nil {
			s.logger.With("error", err).Error("could not get data creator")
			return nil, err
		}

		data_info_idx := slices.IndexFunc(info, func(data_info ProjectDataInfo) bool {
			return data_info.Id == data_id
		})
		if data_info_idx < 0 {
			s.logger.With("data", data_id).Error("could not find data")
			panic("could not find data")
		}

		info[data_info_idx].Creator = DataCreatorUser{Id: creator_id}
	}

	creator_transform_query :=
		`SELECT _data, _creator FROM data_creator_transform_ WHERE _data=ANY($1)`
	rows, _ = s.db.Conn.Query(s.ctx, creator_transform_query, creator_transform_ids)
	for rows.Next() {
		var data_id uuid.UUID
		var creator_id uuid.UUID
		err := rows.Scan(&data_id, &creator_id)
		if err != nil {
			s.logger.With("error", err).Error("could not get data creator")
			return nil, err
		}

		data_info_idx := slices.IndexFunc(info, func(data_info ProjectDataInfo) bool {
			return data_info.Id == data_id
		})
		if data_info_idx < 0 {
			s.logger.With("data", data_id).Error("could not find data")
			panic("could not find data")
		}

		info[data_info_idx].Creator = DataCreatorTransform{Id: creator_id}
	}

	tags_query :=
		`SELECT _data, _tag FROM project_data_tag_
		WHERE _project=$1 AND _data=ANY($2)
		GROUP BY _data`
	rows, _ = s.db.Conn.Query(s.ctx, tags_query, project_id, data_ids)
	for rows.Next() {
		var data_id uuid.UUID
		var tags []string
		err := rows.Scan(&data_id, &tags)
		if err != nil {
			s.logger.With("error", err).Error("could not get data tags")
			return nil, err
		}

		data_info_idx := slices.IndexFunc(info, func(data_info ProjectDataInfo) bool {
			return data_info.Id == data_id
		})
		if data_info_idx < 0 {
			s.logger.With("data", data_id).Error("could not find data")
			panic("could not find data")
		}

		info[data_info_idx].Tags = tags
	}

	properties_query :=
		`SELECT _data, _key, _type, value FROM project_data_property_
		WHERE _project=$1 AND _data=ANY($2) GROUP BY _data`
	rows, _ = s.db.Conn.Query(s.ctx, properties_query, project_id, data_ids)
	for rows.Next() {
		var data_id uuid.UUID
		var properties []Property
		err := rows.Scan(&data_id, &properties)
		if err != nil {
			s.logger.With(
				"error", err,
				"query", properties_query,
				"data", data_ids,
			).Error("could not get data properties")
			return nil, err
		}

		data_info_idx := slices.IndexFunc(info, func(data_info ProjectDataInfo) bool {
			return data_info.Id == data_id
		})
		if data_info_idx < 0 {
			s.logger.With("data", data_id).Error("could not find data")
			panic("could not find data")
		}

		info[data_info_idx].Properties = properties
	}

	note_count_query :=
		`SELECT _data, COUNT(*) FROM project_data_note_ 
		WHERE _project=$1, _data=ANY($2) AND (_creator=$3 OR visibility='public')
		GROUP BY _data`
	rows, _ = s.db.Conn.Query(s.ctx, note_count_query, project_id, data_ids, user_id)
	for rows.Next() {
		var data_id uuid.UUID
		var count uint
		err := rows.Scan(&data_id, &count)
		if err != nil {
			s.logger.With(
				"error", err,
				"query", note_count_query,
				"data", data_ids,
				"user", user_id,
			).Error("could not get data note count")
			return nil, err
		}

		data_info_idx := slices.IndexFunc(info, func(data_info ProjectDataInfo) bool {
			return data_info.Id == data_id
		})
		if data_info_idx < 0 {
			s.logger.With("data", data_id).Error("could not find data")
			panic("could not find data")
		}

		info[data_info_idx].NoteCount = count
	}

	return info, nil
}

type ProjectWithUserPermission struct {
	Id          uuid.UUID  `db:"_id"`
	Creator     uuid.UUID  `db:"_creator"`
	Label       string     `db:"label"`
	Description string     `db:"description"`
	Visibility  Visibility `db:"visibility"`
	Permissions []ProjectPermission
}

func (s *ProjectService) GetProjectWithUserPermission(
	user_id uuid.UUID,
	project_id uuid.UUID,
) (ProjectWithUserPermission, error) {
	var project ProjectWithUserPermission
	project_query :=
		"SELECT _id, _creator, label, description, visibility FROM project_ WHERE _id=$1"
	rows, _ := s.db.Conn.Query(
		s.ctx,
		project_query,
		project_id,
	)
	project, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByNameLax[ProjectWithUserPermission])
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user_id,
			"project", project_id,
		).Error("could not get project")
		return ProjectWithUserPermission{}, err
	}

	project.Permissions, err = s.ProjectUserPermissions(project_id, user_id)
	if err != nil {
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

	raw_data_ids, err := s.create_project_samples_create_raw_data(
		tx,
		sample_ids,
		user_id,
	)
	if err != nil {
		return err
	}

	err = s.create_project_samples_create_raw_data_properties(
		tx,
		raw_data_ids,
		samples,
	)
	if err != nil {
		return err
	}

	// TODO: Store raw data

	err = s.create_project_samples_raw_data_user_permisson_as_owner(
		tx,
		raw_data_ids,
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

type rawDataIdx struct {
	RawIndex  int
	DataIndex int
}

func (s *ProjectService) create_project_samples_create_raw_data(
	tx pgx.Tx,
	sample_ids []uuid.UUID,
	user_id uuid.UUID,
) (map[rawDataIdx]uuid.UUID, error) {
	panic("")
}

func (s *ProjectService) create_project_samples_raw_data_user_permisson_as_owner(
	tx pgx.Tx,
	raw_data_ids map[rawDataIdx]uuid.UUID,
	user uuid.UUID,
) error {
	if len(raw_data_ids) == 0 {
		return nil
	}

	args := make([]any, len(raw_data_ids)+2)
	args[0] = user
	args[1] = DataUserPermissionOwner
	arg_idx := 2
	var query strings.Builder
	query.WriteString("INSERT INTO raw_data_user_permission_ (_sample_data, _user, _permission) VALUES ")
	for _, sample_data := range raw_data_ids {
		args[arg_idx] = sample_data
		fmt.Fprintf(&query, "($%d, $1, $2)", arg_idx+1)
		arg_idx += 1
	}

	_, err := tx.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"raw data", raw_data_ids,
			"user", user,
		).Error("could not insert raw data user permission")
		return err
	}

	return nil
}

func (s *ProjectService) create_project_samples_create_raw_data_properties(
	tx pgx.Tx,
	raw_data_ids map[rawDataIdx]uuid.UUID,
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
			sample_data_idx := rawDataIdx{
				RawIndex:  sample_idx,
				DataIndex: data_idx,
			}
			sample_data_id, present := raw_data_ids[sample_data_idx]
			if !present {
				s.logger.With(
					"sample data ids", raw_data_ids,
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
	args[2] = ProjectSamplePermissionModifyLabel
	args[3] = ProjectSamplePermissionModifyTags
	args[4] = ProjectSamplePermissionModifyProperties

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
	Permissions []ProjectSamplePermission
}

type ProjectSampleResources struct {
	Id                           uuid.UUID
	Creator                      uuid.UUID
	Properties                   []Property
	ProjectMembership            ProjectSampleMembershipAsResource
	ProjectTags                  []string
	ProjectNotes                 []ProjectSampleNote
	Data                         []DataRx
	DerivedData                  []DerivedData
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

	data_ids := []uuid.UUID{}
	for _, data := range resources.Data {
		data_ids = append(data_ids, data.Id)
	}
	resources.DerivedData, err = s.get_project_sample_resources_derived_data(data_ids)
	if err != nil {
		return ProjectSampleResources{}, nil
	}

	data_schema_ids := []uuid.UUID{}
	for _, data := range resources.DerivedData {
		data_schema_ids = append(data_schema_ids, data.Schema)
	}

	resources.DataSchemas, err = s.data_service.DataSchemasById(data_schema_ids)
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
	// TODO: Get user's associated with data
	resources.Users, err = s.user_service.UsersById(user_ids)
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

	membership.Creator, err = s.user_service.UserById(user_id)
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
) ([]DataRx, error) {
	data_query := "SELECT _id, _sample, _schema, _creator, timestamp FROM sample_data_ WHERE _sample=$1"
	rows, _ := s.db.Conn.Query(s.ctx, data_query, sample_id)
	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[DataRx])
	if err != nil {
		s.logger.With("error", err, "query", data_query).Error("could not get sample data")
		return nil, err
	}

	return data, nil
}

// TODO: Make better
func (s *ProjectService) get_project_sample_resources_derived_data(
	sample_data_ids []uuid.UUID,
) ([]DerivedData, error) {
	all_schema_query := "SELECT _id FROM data_schema_"
	rows, _ := s.db.Conn.Query(s.ctx, all_schema_query)
	schema_ids, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (uuid.UUID, error) {
		var id uuid.UUID
		err := row.Scan(&id)
		return id, err
	})
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get data schema ids")
		return nil, err
	}

	var data []DerivedData
	for _, schema_id := range schema_ids {
		query := fmt.Sprintf(
			"SELECT _parent, _transform, _sample_data FROM %s WHERE _sample_data=ANY($1)",
			DataStorageTableNameFromSchemaId(schema_id),
		)
		rows, _ = s.db.Conn.Query(s.ctx, query, sample_data_ids)
		schema_data, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DerivedData, error) {
			sdata := DerivedData{Schema: schema_id}
			err := row.Scan(&sdata.Parent, &sdata.Transform, &sdata.SampleData)
			return sdata, err
		})
		if err != nil {
			s.logger.With(
				"error", err,
				"schema", schema_id,
			).Error("could not get derived data")
			// return nil, err
			continue
		}

		data = slices.Concat(data, schema_data)
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
		var permission ProjectSamplePermission
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
				ProjectSampleUserPermissions{User: user, Permissions: []ProjectSamplePermission{permission}},
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

type ProjectDataMembershipRx struct {
	Project uuid.UUID `db:"_project"`
	Data    uuid.UUID `db:"_data"`
	Creator uuid.UUID `db:"_creator"`
	Label   *string   `db:"label"`
}

func (s *ProjectService) DataMembershipCreate(memberships []ProjectDataMembershipRx) error {
	const numFields = 4
	args := make([]any, len(memberships)*numFields)
	var query strings.Builder
	query.WriteString(
		"INSERT INTO project_data_membership_ (_project, _data, _creator, label) VALUES ",
	)
	for idx, membership := range memberships {
		project_idx := idx * numFields
		data_idx := project_idx + 1
		creator_idx := data_idx + 1
		label_idx := creator_idx + 1

		args[project_idx] = membership.Project
		args[data_idx] = membership.Data
		args[creator_idx] = membership.Creator
		args[label_idx] = membership.Label

		if idx > 0 {
			query.WriteString(", ")
		}
		fmt.Fprintf(
			&query,
			"($%d, $%d, $%d, $%d)",
			project_idx+1,
			data_idx+1,
			creator_idx+1,
			label_idx+1,
		)
	}

	_, err := s.db.Conn.Exec(s.ctx, query.String(), args...)
	if err != nil {
		s.logger.With(
			"error", err,
			"memberships", memberships,
		).Error("could not create project data memberships")
		return err
	}

	return nil
}
