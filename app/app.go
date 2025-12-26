package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.linka.cloud/go-appdir"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wneessen/go-mail"
)

const APP_NAME = "syredb"
const CONFIG_FILE_NAME = "config.toml"
const USER_AUTH_FILE_NAME = "user_auth.toml"

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

type UserNotAuthenticatedError struct{}

func (e *UserNotAuthenticatedError) Error() string {
	return "USER_NOT_AUTHENTICATED"
}

type InsufficientPermissionsError struct{}

func (e *InsufficientPermissionsError) Error() string {
	return "INSUFFICEINT_PERMISSIONS"
}

type AppConfigState = Result[AppConfig]
type DbConnectionState = Result[*pgxpool.Pool]

type AppConfig struct {
	DbUrl      string
	DbUsername string
	DbPassword string
	DbName     string
}

type AppState struct {
	user_id uuid.UUID
}

// func (app *App) Shutdown(ctx context.Context) {
// 	// TODO: Might break if not set, check err?
// 	app.db_pool.ok.Close()
// }

type AppService struct {
	app     *application.App
	ctx     context.Context
	app_dir string
	config  AppConfigState
	db_pool DbConnectionState
	state   AppState
}

func NewAppService(app *application.App) *AppService {
	return &AppService{app: app}
}

func (s *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	s.app_dir = GetConfigDir()

	err := os.MkdirAll(s.app_dir, FILE_PERMISSIONS_WRR)
	if err != nil {
		s.app.Logger.With("error", err).Error("could not create config dir")
		s.config.err = err
	}

	if err == nil {
		s.LoadAppConfig()
	}

	return nil
}

func (s *AppService) Logout() (Ok, error) {
	user_auth_file_path := filepath.Join(s.app_dir, USER_AUTH_FILE_NAME)
	err := os.Remove(user_auth_file_path)
	if err != nil {
		s.app.Logger.With("error", err).Error("could not remove user auth file")
	}

	s.state.user_id = uuid.Nil
	return Ok{}, err
}

// Get app config directory path.
func GetConfigDir() string {
	dirs := appdir.New(APP_NAME)
	return dirs.UserConfig()
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *AppService) send_mail(to string, subject string, body string) error {
	app_email_query := fmt.Sprintf(
		"SELECT key, value FROM _app_data_ WHERE key in ('%s', '%s', '%s', '%s')",
		APP_EMAIL_URL_KEY,
		APP_EMAIL_USERNAME_KEY,
		APP_EMAIL_PASSWORD_KEY,
		APP_EMAIL_FROM_KEY,
	)
	email_rows, _ := s.db_pool.ok.Query(s.ctx, app_email_query)
	defer email_rows.Close()

	var app_email_url string
	var app_email_username string
	var app_email_password string
	var app_email_from string
	var key string
	var value string
	for email_rows.Next() {
		err := email_rows.Scan(&key, &value)
		if err != nil {
			s.app.Logger.With("error", err).Error("could not get email value")
			return err
		}

		switch key {
		case APP_EMAIL_URL_KEY:
			app_email_url = value
		case APP_EMAIL_USERNAME_KEY:
			app_email_username = value
		case APP_EMAIL_PASSWORD_KEY:
			app_email_password = value
		case APP_EMAIL_FROM_KEY:
			app_email_from = value
		default:
			s.app.Logger.With("key", key).Error("invalid key")
			os.Exit(1)
		}
	}
	if app_email_url == "" {
		s.app.Logger.With("key", APP_EMAIL_URL_KEY).Error("required app data not found")
		return errors.New("invalid app email url")
	}
	if app_email_username == "" {
		s.app.Logger.With("key", APP_EMAIL_USERNAME_KEY).Error("required app data not found")
		return errors.New("invalid app email username")
	}
	if app_email_password == "" {
		s.app.Logger.With("key", APP_EMAIL_PASSWORD_KEY).Error("required app data not found")
		return errors.New("invalid app email password")
	}
	if app_email_from == "" {
		s.app.Logger.With("key", APP_EMAIL_FROM_KEY).Error("required app data not found")
		return errors.New("invalid app email from address")

	}

	message := mail.NewMsg()
	err := message.From(app_email_from)
	if err != nil {
		s.app.Logger.With("error", err).Error("invalid app email from address")
		return err
	}
	err = message.To(to)
	if err != nil {
		s.app.Logger.With("error", err).Error("invalid email to address")
		return err
	}

	message.Subject(subject)
	message.SetBodyString(mail.TypeTextPlain, body)

	client, err := mail.NewClient(app_email_url, mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(app_email_username), mail.WithPassword(app_email_password))
	if err != nil {
		s.app.Logger.With("error", err).Error("could not connect to email client")
		return err
	}

	err = client.DialAndSend(message)
	if err != nil {
		s.app.Logger.With("error", err).Error("could not send email")
		return err
	}

	return nil
}

func (s *AppService) user_has_role(role string) (bool, error) {
	if s.db_pool.err != nil {
		return false, s.db_pool.err
	}

	user_role_query := "SELECT 1 FROM user_ WHERE _id=$1 AND role=$2"
	user_row := s.db_pool.ok.QueryRow(context.Background(), user_role_query, s.state.user_id, role)
	err := user_row.Scan()
	granted := err != nil
	return granted, nil
}
