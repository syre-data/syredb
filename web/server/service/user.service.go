package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"syredb/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const AppEmailUrlKey = "app:email:url"
const AppEmailUsernameKey = "app:email:username"
const AppEmailPasswordKey = "app:email:password"
const AppEmailFromKey = "app:email:from"

type DbPermissionId string

const (
	DbPermissionIdOwner            DbPermissionId = "owner"
	DbPermissionIdUserCreate       DbPermissionId = "user_create"
	DbPermissionIdUserModify       DbPermissionId = "user_modify"
	DbPermissionIdDataSchemaCreate DbPermissionId = "data_schema_create"
	DbPermissionIdDataSchemaModify DbPermissionId = "data_schema_modify"
	DbPermissionIdDataTypeCreate   DbPermissionId = "data_type_create"
	DbPermissionIdDataTypeModify   DbPermissionId = "data_type_modify"
	DbPermissionIdTransformCreate  DbPermissionId = "transform_create"
	DbPermissionIdTransformModify  DbPermissionId = "transform_modify"
	DbPermissionIdProjectCreate    DbPermissionId = "project_create"
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
	DbPermissions []DbPermissionId
}

func (s *UserService) UserById(user_id uuid.UUID) (User, error) {
	user := User{Id: user_id}
	user_query := "SELECT account_status, email, name FROM user_ WHERE _id=$1"
	err := s.db.Conn.QueryRow(s.ctx, user_query, user_id).Scan(
		&user.AccountStatus,
		&user.Email,
		&user.Name,
	)
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user_id,
		).Error("could not get user")
		return User{}, err
	}

	user.DbPermissions, err = s.UserPermissions(user_id)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *UserService) UserPermissions(user_id uuid.UUID) ([]DbPermissionId, error) {
	query := "SELECT _permission FROM db_user_permission_ WHERE _user=$1"
	rows, _ := s.db.Conn.Query(s.ctx, query, user_id)
	permissions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DbPermissionId, error) {
		var permission DbPermissionId
		err := row.Scan(&permission)
		return permission, err
	})
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user_id,
		).Error("could not get user db permissions")
		return nil, err
	}

	return permissions, nil
}

type UserCreate struct {
	Email         string           `form:"email"`
	Name          string           `form:"name"`
	Password      string           `form:"password"`
	DbPermissions []DbPermissionId `form:"db_permissions"`
}

func (s *UserService) CreateUser(user UserCreate) (uuid.UUID, error) {
	if len(user.Email) == 0 || len(user.Name) == 0 {
		return uuid.Nil, errors.New("invalid user")
	}

	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("unable to begin create user transaction")
		return uuid.Nil, err
	}
	defer tx.Rollback(s.ctx)

	var user_id uuid.UUID
	insert_user_query := "INSERT INTO user_ (email, name) VALUES ($1, $2) RETURNING _id"
	err = tx.QueryRow(s.ctx, insert_user_query, user.Email, user.Name).Scan(&user_id)
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

	if len(user.DbPermissions) > 0 {
		err = s.SetUserPermissions(tx, user_id, user.DbPermissions)
		if err != nil {
			return uuid.Nil, err
		}
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

func (s *UserService) UserUpdate(update User) error {
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not begin transaction")
		return err
	}
	defer tx.Rollback(s.ctx)

	update_user_query := "UPDATE user_ SET account_status=$2, email=$3, name=$4 WHERE _id=$1"
	_, err = tx.Exec(
		s.ctx,
		update_user_query,
		update.Id,
		update.AccountStatus,
		update.Email,
		update.Name,
	)
	if err != nil {
		s.logger.With("error", err).Error("could not update user")
		return err
	}

	err = s.SetUserPermissions(tx, update.Id, update.DbPermissions)
	if err != nil {
		return err
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.logger.With(
			"error", err,
		).Error("could not commit transaction")
		return err
	}
	return nil
}

func (s *UserService) SetUserPermissions(tx pgx.Tx, user uuid.UUID, permissions []DbPermissionId) error {
	add_permission := []DbPermissionId{}
	remove_permission := []DbPermissionId{}
	current, err := s.UserPermissions(user)
	if err != nil {
		return err
	}
	for _, permission := range permissions {
		if !slices.Contains(current, permission) {
			add_permission = append(add_permission, permission)
		}
	}
	for _, permission := range current {
		if !slices.Contains(permissions, permission) {
			remove_permission = append(remove_permission, permission)
		}
	}

	if len(remove_permission) > 0 {
		remove_permission_query := "DELETE FROM db_user_permission_ WHERE _user=$1 AND _permission=ANY($2)"
		_, err = tx.Exec(s.ctx, remove_permission_query, user, remove_permission)
		if err != nil {
			s.logger.With(
				"error", err,
				"user", user,
				"permissions", remove_permission,
			).Error("could not remove db user permissions")
			return err
		}
	}

	if len(add_permission) > 0 {
		const argIdxOffset = 1
		add_args := make([]any, len(add_permission)+argIdxOffset)
		add_args[0] = user
		var add_permission_query strings.Builder
		add_permission_query.WriteString("INSERT INTO db_user_permission_ (_user, _permission) VALUES ")
		for idx, permission := range add_permission {
			arg_idx := idx + argIdxOffset
			add_args[arg_idx] = permission
			if idx > 0 {
				add_permission_query.WriteString(", ")
			}
			fmt.Fprintf(&add_permission_query, "($1, $%d)", arg_idx+1)
		}
		_, err = tx.Exec(s.ctx, add_permission_query.String(), add_args...)
		if err != nil {
			s.logger.With(
				"error", err,
				"user", user,
				"permissions", add_permission,
			).Error("could not add db user permissions")
			return err
		}
	}
	return nil
}

func (s *UserService) AllUsers() ([]User, error) {
	users_query := "SELECT _id, account_status, email, name FROM user_ ORDER BY _id"
	user_rows, _ := s.db.Conn.Query(s.ctx, users_query)
	users, err := pgx.CollectRows(user_rows, func(row pgx.CollectableRow) (User, error) {
		var user User
		err := row.Scan(&user.Id, &user.AccountStatus, &user.Email, &user.Name)
		return user, err
	})
	if err != nil {
		s.logger.With("error", err).Error("could not collect users")
		return nil, err
	}

	// TODO: get user permissions

	return users, nil
}

func (s *UserService) UsersById(user_ids []uuid.UUID) ([]User, error) {
	user_ids_str := make([]string, len(user_ids))
	for idx, id := range user_ids {
		user_ids_str[idx] = fmt.Sprintf("'%s'", id)
	}

	users_query := fmt.Sprintf(
		`SELECT _id, account_status, email, name FROM user_
		WHERE _id IN (%s)`,
		strings.Join(user_ids_str, ", "),
	)

	rows, _ := s.db.Conn.Query(s.ctx, users_query)
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) {
		var user User
		err := row.Scan(&user.Id, &user.AccountStatus, &user.Email, &user.Name)
		return user, err
	})
	if err != nil {
		s.logger.With("error", err, "user ids", user_ids).Error("could not collect users")
		return nil, err
	}

	return users, nil
}

func (s *UserService) UserHasPermission(user uuid.UUID, permission DbPermissionId) (bool, error) {
	query := fmt.Sprintf(
		"SELECT 1 FROM db_user_permission_ WHERE _user=$1 AND _permission=ANY('{%s, $2}')",
		DbPermissionIdOwner,
	)
	user_row := s.db.Conn.QueryRow(s.ctx, query, user, permission)
	err := user_row.Scan()
	has_permission := err != nil
	return has_permission, nil
}
