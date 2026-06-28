package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syredb/database"

	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/wneessen/go-mail"
)

type FileFilter struct {
	DisplayName string // Filter information EG: "Image Files (*.jpg, *.png)"
	Pattern     string // semicolon separated list of extensions, EG: "*.jpg;*.png"
}

type AppService struct {
	ctx    context.Context
	logger *slog.Logger
	db     *database.DBConnection
}

func NewAppService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DBConnection,
) *AppService {
	return &AppService{
		ctx:    ctx,
		logger: logger,
		db:     db,
	}
}

func (s *AppService) SendMail(to string, subject string, body string) error {
	app_email_query := fmt.Sprintf(
		"SELECT key, value FROM _app_data_ WHERE key IN ('%s', '%s', '%s', '%s')",
		AppEmailUrlKey,
		AppEmailUsernameKey,
		AppEmailPasswordKey,
		AppEmailFromKey,
	)
	email_rows, _ := s.db.Conn.Query(s.ctx, app_email_query)
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
			s.logger.With("error", err).Error("could not get email value")
			return err
		}

		switch key {
		case AppEmailUrlKey:
			app_email_url = value
		case AppEmailUsernameKey:
			app_email_username = value
		case AppEmailPasswordKey:
			app_email_password = value
		case AppEmailFromKey:
			app_email_from = value
		default:
			s.logger.With("key", key).Error("invalid key")
			os.Exit(1)
		}
	}
	if app_email_url == "" {
		s.logger.With("key", AppEmailUrlKey).Error("required app data not found")
		return errors.New("invalid app email url")
	}
	if app_email_username == "" {
		s.logger.With("key", AppEmailUsernameKey).Error("required app data not found")
		return errors.New("invalid app email username")
	}
	if app_email_password == "" {
		s.logger.With("key", AppEmailPasswordKey).Error("required app data not found")
		return errors.New("invalid app email password")
	}
	if app_email_from == "" {
		s.logger.With("key", AppEmailFromKey).Error("required app data not found")
		return errors.New("invalid app email from address")

	}

	message := mail.NewMsg()
	err := message.From(app_email_from)
	if err != nil {
		s.logger.With("error", err).Error("invalid app email from address")
		return err
	}
	err = message.To(to)
	if err != nil {
		s.logger.With("error", err).Error("invalid email to address")
		return err
	}

	message.Subject(subject)
	message.SetBodyString(mail.TypeTextPlain, body)

	client, err := mail.NewClient(app_email_url, mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(app_email_username), mail.WithPassword(app_email_password))
	if err != nil {
		s.logger.With("error", err).Error("could not connect to email client")
		return err
	}

	err = client.DialAndSend(message)
	if err != nil {
		s.logger.With("error", err).Error("could not send email")
		return err
	}

	return nil
}

type DbPermissionRecord struct {
	Id          string `db:"_id"`
	Label       string `db:"label"`
	Description string `db:"description"`
}

func (s *AppService) DbPermissionsAll() ([]DbPermissionRecord, error) {
	query := "SELECT _id, label, description FROM _db_permission_"
	rows, _ := s.db.Conn.Query(s.ctx, query)
	permissions, err := pgx.CollectRows(rows, pgx.RowToStructByName[DbPermissionRecord])
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not get db permissions")
		return nil, err
	}

	return permissions, nil
}

type AppDataKey string

const (
	AppDataKeyEmailUrl      AppDataKey = "app:email:url"
	AppDataKeyEmailUsername AppDataKey = "app:email:username"
	AppDataKeyEmailPassword AppDataKey = "app:email:password"
	AppDataKeyEmailFrom     AppDataKey = "app:email:from"
	AppDataKeyAccountName   AppDataKey = "app:account:name"
	AppDataKeyAccountLogo   AppDataKey = "app:account:logo"
	AppDataKeyDataPath      AppDataKey = "app:data:path"
)

func (s *AppService) AppData(key AppDataKey) (string, error) {
	var value string
	query := "SELECT value FROM _app_data_ WHERE key=$1"
	err := s.db.Conn.QueryRow(s.ctx, query, string(key)).Scan(&value)
	return value, err
}

type AppDataDir string

const (
	AppDataDirTransform  AppDataDir = "transform"
	AppDataDirDataSource AppDataDir = "data_source"
)

// AppDataDir gets the path to the associated data directiory.
func (s *AppService) AppDataDir(dir AppDataDir) (string, error) {
	app_dir, err := s.AppData(AppDataKeyDataPath)
	if err != nil {
		s.logger.With(
			"error", err,
			"key", AppDataKeyDataPath,
		).Error("could not get app data path")
		return "", err
	}

	dirpath := filepath.Join(app_dir, string(dir))
	return dirpath, nil

}
