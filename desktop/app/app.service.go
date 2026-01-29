package app

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const APP_NAME = "syredb"
const CONFIG_FILE_NAME = "config.toml"

const APP_EMAIL_URL_KEY = "app:email:url"
const APP_EMAIL_USERNAME_KEY = "app:email:username"
const APP_EMAIL_PASSWORD_KEY = "app:email:password"
const APP_EMAIL_FROM_KEY = "app:email:from"

const FILE_PERMISSIONS_WRR = 0644

type Result[T any] struct {
	ok  T
	err error
}

type Ok struct{}

type DbConnection struct {
	conn *pgxpool.Pool
}

type AppConfigState = Result[AppConfig]
type DbConnectionState = Result[*DbConnection]

type AppConfig struct {
	DbUrl      string
	DbUsername string
	DbPassword string
	DbName     string
}

type AppState struct {
	_lock   sync.RWMutex
	user_id uuid.UUID
}

type AppService struct {
	logger  *slog.Logger
	ctx     context.Context
	app_dir string
	config  AppConfigState
	db      DbConnectionState
	state   AppState
}

func NewAppService(logger *slog.Logger, db *DbConnection, app_dir string) *AppService {
	db_state := DbConnectionState{ok: db, err: nil}
	return &AppService{logger: logger, db: db_state, app_dir: app_dir}
}

func (s *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx

	err := os.MkdirAll(s.app_dir, FILE_PERMISSIONS_WRR)
	if err != nil {
		s.logger.With("error", err).Error("could not create config dir")
		s.config.err = err
	}

	if err == nil {
		_, err = s.AppConfigLoad()
		if err == nil {

		}
	}

	return nil
}

func (s *AppService) ServiceShutdown() error {
	s.db.ok.conn.Close()
	return nil
}

//wails:internal
func (s *AppService) AppState() *AppState {
	return &s.state
}
