package service

import (
	"context"
	"log/slog"
	"syredb/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProjectUserPermission string

const (
	PROJECT_USER_PERMISSION_READ       ProjectUserPermission = "read"
	PROJECT_USER_PERMISSION_READ_WRITE ProjectUserPermission = "read_write"
	PROJECT_USER_PERMISSION_ADMIN      ProjectUserPermission = "admin"
	PROJECT_USER_PERMISSION_OWNER      ProjectUserPermission = "owner"
)

type Visibility string

const (
	VISIBILITY_PUBLIC  Visibility = "public"
	VISIBILITY_PRIVATE Visibility = "private"
)

type SampleUserPermission string

const (
	SAMPLE_USER_PERMISSION_OWNER             SampleUserPermission = "owner"
	SAMPLE_USER_PERMISSION_READ              SampleUserPermission = "read"
	SAMPLE_USER_PERMISSION_ADD_DATA          SampleUserPermission = "add_data"
	SAMPLE_USER_PERMISSION_CREATE_NOTE       SampleUserPermission = "create_note"
	SAMPLE_USER_PERMISSION_MODIFY_PROPERTIES SampleUserPermission = "modify_properties"
)

type ProjectService struct {
	ctx    context.Context
	logger *slog.Logger
	db     *database.DbConnection
}

func NewProjectService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DbConnection,
) *ProjectService {
	return &ProjectService{
		ctx:    ctx,
		logger: logger,
		db:     db,
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

	user_project_query := "SELECT _id, _creator, label, description, visibility FROM project_ WHERE _creator=$1 ORDER BY _id"
	rows, _ := s.db.Conn.Query(s.ctx, user_project_query, user)
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
