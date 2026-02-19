package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"syredb/database"

	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const AppEmailUrlKey = "app:email:url"
const AppEmailUsernameKey = "app:email:username"
const AppEmailPasswordKey = "app:email:password"
const AppEmailFromKey = "app:email:from"

type UserRole string

const (
	UserRoleOwner UserRole = "owner"
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

type AccountStatus string

const (
	AccountStatusActive      AccountStatus = "active"
	AccountStatusDeactivated AccountStatus = "deactivated"
)

type UserService struct {
	ctx    context.Context
	logger *slog.Logger
	db     *database.DBConnection
	auth   *AuthService
}

func NewUserService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DBConnection,
	auth_service *AuthService,
) *UserService {
	return &UserService{
		ctx:    ctx,
		logger: logger,
		db:     db,
		auth:   auth_service,
	}
}

type User struct {
	Id            uuid.UUID
	AccountStatus AccountStatus
	Email         string
	Name          string
	Role          UserRole
}

func (s *UserService) UserById(user_id uuid.UUID) (User, error) {
	user := User{Id: user_id}
	user_query := "SELECT account_status, email, name, role FROM user_ WHERE _id=$1"
	err := s.db.Conn.QueryRow(s.ctx, user_query, user_id).Scan(
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

type UserCreate struct {
	Email    string
	Name     string
	Password string
	Role     string
}

func (s *UserService) CreateUser(user UserCreate) (uuid.UUID, error) {
	tx, err := s.db.Conn.Begin(s.ctx)
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
	_, err = tx.Exec(s.ctx, insert_user_auth_query, user_id, s.auth.EncodePassword(user.Password))
	if err != nil {
		s.logger.With("error", err).Error("error inserting user authentication data")
		return uuid.Nil, err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("error committing create user transaction")
		return uuid.Nil, err
	}

	return user_id, nil
}

func (s *UserService) DeactivateUser(user_id uuid.UUID) error {
	// SAFETY: Users shall never be removed from the database.
	// Deactivating a user shall only remove their access to the database.
	// All information about the user shall be retained.
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin delete user transaction")
		return err
	}
	defer tx.Rollback(s.ctx)

	delete_user_auth_query := "DELETE FROM user_auth_ WHERE _id=$1"
	_, err = tx.Exec(s.ctx, delete_user_auth_query, user_id)
	if err != nil {
		s.logger.With("error", err).Error("could not remove user auth")
		return err
	}

	set_user_status_query := "UPDATE user_ SET account_status='disabled' WHERE _id=$1"
	_, err = tx.Exec(s.ctx, set_user_status_query, user_id)
	if err != nil {
		s.logger.With("error", err).Error("could not remove user auth")
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not commit remove user auth transaction")
		return err
	}

	return nil
}

func (s *UserService) UpdateUser(update User) error {
	update_user_query := "UPDATE user_ SET account_status=$2, email=$3, name=$4, role=$5 WHERE _id=$1"
	_, err := s.db.Conn.Exec(
		s.ctx,
		update_user_query,
		update.Id,
		update.AccountStatus,
		update.Email,
		update.Name,
		update.Role,
	)

	if err != nil {
		s.logger.With("error", err).Error("could not update user")
		return err
	}

	return nil
}

func (s *UserService) GetUserById(user_id uuid.UUID) (User, error) {
	user := User{Id: user_id}
	user_query := "SELECT account_status, email, name, role FROM user_ WHERE _id=$1"
	err := s.db.Conn.QueryRow(s.ctx, user_query, user_id).Scan(
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

func (s *UserService) GetUsersAll() ([]User, error) {
	users_query := "SELECT _id, account_status, email, name, role FROM user_ ORDER BY _id"
	user_rows, _ := s.db.Conn.Query(s.ctx, users_query)
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

	rows, _ := s.db.Conn.Query(s.ctx, users_query)
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

func (s *UserService) UserRole(user uuid.UUID) (UserRole, error) {
	if user == uuid.Nil {
		return UserRole(""), errors.New("nil id")
	}

	var user_role UserRole
	user_role_query := "SELECT role FROM user_ WHERE _id=$1"
	err := s.db.Conn.QueryRow(
		s.ctx,
		user_role_query,
		user,
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

func (s *UserService) UserHasRole(user uuid.UUID, role UserRole) (bool, error) {
	user_role_query := "SELECT 1 FROM user_ WHERE _id=$1 AND role=$2"
	user_row := s.db.Conn.QueryRow(s.ctx, user_role_query, user, role)
	err := user_row.Scan()
	has_role := err != nil
	return has_role, nil
}
