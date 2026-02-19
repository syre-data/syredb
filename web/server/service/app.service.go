package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"syredb/database"

	"errors"

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
