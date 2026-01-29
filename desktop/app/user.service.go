package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wneessen/go-mail"
)

const (
	USER_ROLE_OWNER = UserRole("owner")
	USER_ROLE_ADMIN = UserRole("admin")
	USER_ROLE_USER  = UserRole("user")
)

type UserRole string

type UserService struct {
	ctx       context.Context
	logger    *slog.Logger
	db        *DbConnection
	app_state *AppState
}

func NewUserService(
	logger *slog.Logger,
	db *DbConnection,
	app_state *AppState,
) *UserService {
	return &UserService{logger: logger, db: db, app_state: app_state}
}

func (s *UserService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

func (s *UserService) user_id() uuid.UUID {
	s.app_state._lock.RLock()
	defer s.app_state._lock.RUnlock()
	return s.app_state.user_id
}

type User struct {
	Id            uuid.UUID
	AccountStatus string
	Email         string
	Name          string
	Role          UserRole
}

type UserCreate struct {
	Email    string
	Name     string
	Password string
	Role     string
}

func (s *UserService) CreateUser(user UserCreate) (uuid.UUID, error) {
	tx, err := s.db.conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("unable to begin create user transaction")
		return uuid.Nil, err
	}
	defer tx.Rollback(s.ctx)

	var user_id uuid.UUID
	insert_user_query := "INSERT INTO user_ (email, name, role) VALUES ($1, $2, $3) RETURNING _id"
	err = tx.QueryRow(s.ctx, insert_user_query, user.Email, user.Name, user.Role).Scan(&user_id)
	if err != nil {
		s.logger.With("error", err).Error("error inserting user")
		return uuid.Nil, err
	}

	insert_user_auth_query := "INSERT INTO user_auth_ (_id, auth) VALUES ($1, $2)"
	_, err = tx.Exec(s.ctx, insert_user_auth_query, user_id, encodePassword(user.Password))
	if err != nil {
		s.logger.With("error", err).Error("error inserting user authentication data")
		return uuid.Nil, err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("error committing create user transaction")
		return uuid.Nil, err
	}

	const subject = "syredb | Welcome!"
	message := fmt.Sprintf("Welcome to syredb. You can log in with this email and the password:\n%s\n\nYou can change your password once you log in.", user.Password)
	err = s.send_mail(user.Email, subject, message)
	if err != nil {
		s.logger.With("error", err).Error("could not send user creation email")
		return user_id, fmt.Errorf("WELCOME_EMAIL_NOT_SENT {password: %s}", user.Password)
	}

	return user_id, nil
}

func (s *UserService) DeactivateUser(user_id uuid.UUID) (Ok, error) {
	// SAFETY: Deleteing a user should only remove their access to the database.
	// All information about the user should be retained.
	tx, err := s.db.conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin delete user transaction")
		return Ok{}, err
	}
	defer tx.Rollback(s.ctx)

	delete_user_auth_query := "DELETE FROM user_auth_ WHERE _id=$1"
	_, err = tx.Exec(s.ctx, delete_user_auth_query, user_id)
	if err != nil {
		s.logger.With("error", err).Error("could not remove user auth")
		return Ok{}, err
	}

	set_user_status_query := "UPDATE user_ SET account_status='disabled' WHERE _id=$1"
	_, err = tx.Exec(s.ctx, set_user_status_query, user_id)
	if err != nil {
		s.logger.With("error", err).Error("could not remove user auth")
		return Ok{}, err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not commit remove user auth transaction")
	}

	return Ok{}, err
}

func (s *UserService) UpdateUser(update User) (Ok, error) {
	update_user_query := "UPDATE user_ SET account_status=$2, email=$3, name=$4, role=$5 WHERE _id=$1"
	_, err := s.db.conn.Exec(
		s.ctx,
		update_user_query,
		update.AccountStatus,
		update.Id,
		update.Email,
		update.Name,
		update.Role,
	)

	if err != nil {
		s.logger.With("error", err).Error("could not update user")
	}

	return Ok{}, err
}

func (s *UserService) GetUserById(user_id uuid.UUID) (User, error) {
	user := User{Id: user_id}
	user_query := "SELECT account_status, email, name, role FROM user_ WHERE _id=$1"
	err := s.db.conn.QueryRow(s.ctx, user_query, user_id).Scan(
		&user.AccountStatus,
		&user.Email,
		&user.Name,
		&user.Role,
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user_id,
		).Error("could not get user")
		return User{}, err
	}

	return user, nil
}

func (s *UserService) GetUsers() ([]User, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return nil, &UserNotAuthenticatedError{}
	}

	user_has_permission, err := s.user_has_role("owner")
	if err != nil {
		s.logger.With("error", err).Error("could not get user permissions")
	}
	if !user_has_permission {
		s.logger.With("user", user_id).Error("insufficient permissions for users list")
		return nil, &InsufficientPermissionsError{}
	}

	users_query := "SELECT _id, account_status, email, name, role FROM user_ ORDER BY _id"
	user_rows, _ := s.db.conn.Query(s.ctx, users_query)
	users, err := pgx.CollectRows(user_rows, func(row pgx.CollectableRow) (User, error) {
		var user User
		err := row.Scan(&user.Id, &user.AccountStatus, &user.Email, &user.Name, &user.Role)
		return user, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not collect users")
		return nil, err
	}

	return users, nil
}

func (s *UserService) GetUsersById(user_ids []uuid.UUID) ([]User, error) {
	user_ids_str := make([]string, len(user_ids))
	for idx, id := range user_ids {
		user_ids_str[idx] = fmt.Sprintf("'%s'", id)
	}

	users_query := fmt.Sprintf(
		`SELECT _id, account_status, email, name, role FROM user_
		WHERE _id IN (%s)`,
		strings.Join(user_ids_str, ", "),
	)

	rows, _ := s.db.conn.Query(s.ctx, users_query)
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) {
		var user User
		err := row.Scan(&user.Id, &user.AccountStatus, &user.Email, &user.Name, &user.Role)
		return user, err
	})
	if err != nil {
		s.logger.With("error", err, "user ids", user_ids).Error("could not collect users")
		return nil, err
	}

	return users, nil
}

func (s *UserService) UserRole() (UserRole, error) {
	user_id := s.user_id()
	if user_id == uuid.Nil {
		return UserRole(""), &UserNotAuthenticatedError{}
	}

	var user_role UserRole
	user_role_query := "SELECT role FROM user_ WHERE _id=$1"
	err := s.db.conn.QueryRow(
		s.ctx,
		user_role_query,
		user_id,
	).Scan(&user_role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserRole(""), nil
		} else {
			return UserRole(""), err
		}
	}

	return user_role, nil
}

func (s *UserService) user_has_role(role string) (bool, error) {
	user_id := s.user_id()
	user_role_query := "SELECT 1 FROM user_ WHERE _id=$1 AND role=$2"
	user_row := s.db.conn.QueryRow(context.Background(), user_role_query, user_id, role)
	err := user_row.Scan()
	granted := err != nil
	return granted, nil
}

func (s *UserService) send_mail(to string, subject string, body string) error {
	app_email_query := fmt.Sprintf(
		"SELECT key, value FROM _app_data_ WHERE key IN ('%s', '%s', '%s', '%s')",
		APP_EMAIL_URL_KEY,
		APP_EMAIL_USERNAME_KEY,
		APP_EMAIL_PASSWORD_KEY,
		APP_EMAIL_FROM_KEY,
	)
	email_rows, _ := s.db.conn.Query(s.ctx, app_email_query)
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
		case APP_EMAIL_URL_KEY:
			app_email_url = value
		case APP_EMAIL_USERNAME_KEY:
			app_email_username = value
		case APP_EMAIL_PASSWORD_KEY:
			app_email_password = value
		case APP_EMAIL_FROM_KEY:
			app_email_from = value
		default:
			s.logger.With("key", key).Error("invalid key")
			os.Exit(1)
		}
	}
	if app_email_url == "" {
		s.logger.With("key", APP_EMAIL_URL_KEY).Error("required app data not found")
		return errors.New("invalid app email url")
	}
	if app_email_username == "" {
		s.logger.With("key", APP_EMAIL_USERNAME_KEY).Error("required app data not found")
		return errors.New("invalid app email username")
	}
	if app_email_password == "" {
		s.logger.With("key", APP_EMAIL_PASSWORD_KEY).Error("required app data not found")
		return errors.New("invalid app email password")
	}
	if app_email_from == "" {
		s.logger.With("key", APP_EMAIL_FROM_KEY).Error("required app data not found")
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
