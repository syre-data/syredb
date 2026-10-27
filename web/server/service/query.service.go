package service

import (
	"context"
	"log/slog"
	"syredb/database"
	"syredb/query"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type QueryService struct {
	ctx          context.Context
	logger       *slog.Logger
	db           *database.DBConnection
	data_service *DataService
}

func NewQueryService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DBConnection,
	data_service *DataService,
) *QueryService {
	return &QueryService{
		ctx:          ctx,
		logger:       logger,
		db:           db,
		data_service: data_service,
	}
}

// Get ids of data matching the query expressions.
func (s *QueryService) SearchData(user uuid.UUID, exprs []query.Expr) ([]uuid.UUID, error) {
	q := query.BuildQuery(user, exprs)
	rows, _ := s.db.Conn.Query(s.ctx, q.Sql.String(), q.Args...)
	data, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		s.logger.With("error", err).Error("could not query data")
		return nil, err
	}
	return data, nil
}

// Get ids of data n the project matching the query expressions.
func (s *QueryService) SearchProjectData(user uuid.UUID, project uuid.UUID, exprs []query.Expr) ([]uuid.UUID, error) {
	q := query.BuildProjectQuery(user, project, exprs)
	s.logger.Info("QUERY", "sql", q.Sql.String(), "args", q.Args)
	rows, _ := s.db.Conn.Query(s.ctx, q.Sql.String(), q.Args...)
	data, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		s.logger.With("error", err).Error("could not query project data")
		return nil, err
	}
	return data, nil
}
