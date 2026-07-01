package service

import (
	"context"
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

func (s *ProjectService) ProjectById(id uuid.UUID) (Project, error) {
	query :=
		`SELECT _id, _creator, label, description, visibility FROM project_ 
		WHERE _id=$1`
	rows, _ := s.db.Conn.Query(s.ctx, query, id)
	project, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Project])
	if err != nil {
		s.logger.With(
			"error", err,
			"project", id,
		).Error("could not get project")
		return Project{}, err
	}

	return project, nil
}

func (s *ProjectService) GetUserProjects(user uuid.UUID) ([]Project, error) {
	if user == uuid.Nil {
		panic("invalid user id")
	}

	query :=
		`SELECT _id, _creator, label, description, visibility FROM project_ 
		WHERE _creator=$1 ORDER BY _id`
	rows, _ := s.db.Conn.Query(s.ctx, query, user)
	projects, err := pgx.CollectRows(rows, pgx.RowToStructByName[Project])
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user,
		).Error("could not collect user projects")
		return nil, err
	}

	return projects, nil
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
	query :=
		`SELECT _permission FROM project_user_permission_ 
		WHERE _project=$1 AND _user=$2`
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

func (s *ProjectService) ProjectsUserPermissions(
	projects []uuid.UUID,
	user uuid.UUID,
) (map[uuid.UUID][]ProjectPermission, error) {
	type rx struct {
		Project    uuid.UUID         `db:"_project"`
		Permission ProjectPermission `db:"_permission"`
	}
	query :=
		`SELECT _project, _permission FROM project_user_permission_ 
		WHERE _project=ANY($1) AND _user=$2`
	rows, _ := s.db.Conn.Query(
		s.ctx,
		query,
		projects,
		user,
	)
	rxs, err := pgx.CollectRows(rows, pgx.RowTo[rx])
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.With(
				"error", err,
				"user", user,
				"project", projects,
			).Error("could not retrieve projects' user permissions")
		}
		return nil, err
	}

	permissions := make(map[uuid.UUID][]ProjectPermission)
	for _, r := range rxs {
		permissions[r.Project] = append(
			permissions[r.Project],
			r.Permission,
		)
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
	DataRelations      map[uuid.UUID][]uuid.UUID
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

	project_resources.DataRelations, err = s.data_service.DataTree(data_ids)
	if err != nil {
		s.logger.With(
			"error", err,
			"data", data_ids,
		).Error("could not get data tree")
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
	project, err := pgx.CollectExactlyOneRow(
		rows,
		pgx.RowToStructByNameLax[ProjectWithUserPermission],
	)
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

func (s *ProjectService) DataMembershipsCreate(memberships []ProjectDataMembershipRx) error {
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

type ProjectDataMembershipRx struct {
	Project uuid.UUID `db:"_project"`
	Data    uuid.UUID `db:"_data"`
	Creator uuid.UUID `db:"_creator"`
	Label   *string   `db:"label"`
}

func (s *ProjectService) DataMembershipCreate(
	project uuid.UUID,
	data uuid.UUID,
	creator uuid.UUID,
	label *string,
) error {
	query :=
		`INSERT INTO project_data_membership_ (_project, _data, _creator, label) 
		VALUES ($1, $2, $3, $4)`

	_, err := s.db.Conn.Exec(s.ctx, query, project, data, creator, label)
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
			"data", data,
			"creator", creator,
			"label", label,
		).Error("could not create project data membership")
		return err
	}

	return nil
}

func (s *ProjectService) DataMembership(project uuid.UUID, data uuid.UUID) (ProjectDataMembershipRx, error) {
	query :=
		`SELECT _project, _data, _creator, label FROM project_data_membership_
		WHERE _project=$1 AND _data=$2`
	rows, _ := s.db.Conn.Query(s.ctx, query, project, data)
	membership, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[ProjectDataMembershipRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
			"data", data,
		).Error("could not get project data membership")
		return ProjectDataMembershipRx{}, err
	}

	return membership, nil
}

func (s *ProjectService) DataMemberships(project uuid.UUID, data []uuid.UUID) ([]ProjectDataMembershipRx, error) {
	query :=
		`SELECT _project, _data, _creator, label FROM project_data_membership_
		WHERE _project=$1 AND _data=ANY($2)`
	rows, _ := s.db.Conn.Query(s.ctx, query, project, data)
	memberships, err := pgx.CollectRows(rows, pgx.RowToStructByName[ProjectDataMembershipRx])
	if err != nil {
		s.logger.With(
			"error", err,
			"project", project,
			"data", data,
		).Error("could not get project data memberships")
		return nil, err
	}

	return memberships, nil
}
