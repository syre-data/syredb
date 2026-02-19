package service

import (
	"context"
	"errors"
	"log/slog"
	"syredb/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SampleUserPermission string

const (
	SampleUserPermissionOwner            SampleUserPermission = "owner"
	SampleUserPermissionRead             SampleUserPermission = "read"
	SampleUserPermissionAddData          SampleUserPermission = "add_data"
	SampleUserPermissionCreateNote       SampleUserPermission = "create_note"
	SampleUserPermissionModifyProperties SampleUserPermission = "modify_properties"
)

type SampleService struct {
	logger *slog.Logger
	ctx    context.Context
	db     *database.DBConnection
}

func NewSampleService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DBConnection,
) *SampleService {
	return &SampleService{
		logger: logger,
		ctx:    ctx,
		db:     db,
	}
}

func (s *SampleService) SampleUserPermission(
	sample uuid.UUID,
	user uuid.UUID,
) (SampleUserPermission, error) {
	var permission SampleUserPermission
	query :=
		`SELECT _permission FROM sample_user_permission_ 
		WHERE _sample=$1 AND _user=$2`
	err := s.db.Conn.QueryRow(s.ctx, query, sample, user).Scan(&permission)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.With(
				"error", err,
				"user", user,
				"sample", sample,
			).Error("could not retrieve project user permission")
		}
		return "", err
	}

	return permission, nil
}
