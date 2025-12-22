package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type UserCreate struct {
	Email    string
	Name     string
	Password string
	Role     string
}

func (app *App) CreateUser(user UserCreate) (uuid.UUID, error) {
	if app.db_pool.err != nil {
		return uuid.Nil, app.db_pool.err
	}

	tx, err := app.db_pool.ok.Begin(context.Background())
	if err != nil {
		runtime.LogErrorf(app.ctx, "unable to begin create user transaction: %v\n", err)
		return uuid.Nil, err
	}
	defer tx.Rollback(context.Background())

	var user_id uuid.UUID
	insert_user_query := "INSERT INTO user_ (email, name, role) VALUES ($1, $2, $3) RETURNING _id"
	err = tx.QueryRow(context.Background(), insert_user_query, user.Email, user.Name, user.Role).Scan(&user_id)
	if err != nil {
		runtime.LogErrorf(app.ctx, "error inserting user: %v\n", err)
		return uuid.Nil, err
	}

	insert_user_auth_query := "INSERT INTO user_auth_ (_id, auth) VALUES ($1, $2)"
	_, err = tx.Exec(context.Background(), insert_user_auth_query, user_id, encodePassword(user.Password))
	if err != nil {
		runtime.LogErrorf(app.ctx, "error inserting user authentication data: %v\n", err)
		return uuid.Nil, err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		runtime.LogErrorf(app.ctx, "error committing create user transaction: %v\n", err)
		return uuid.Nil, err
	}

	const subject = "syredb | Welcome!"
	message := fmt.Sprintf("Welcome to syredb. You can log in with this email and the password:\n%s\n\nYou can change your password once you log in.", user.Password)
	err = app.send_mail(user.Email, subject, message)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not send user creation email: %v", err)
		return user_id, fmt.Errorf("WELCOME_EMAIL_NOT_SENT {password: %s}", user.Password)
	}

	return user_id, nil
}

func (app *App) DeactivateUser(user_id uuid.UUID) (Ok, error) {
	if app.db_pool.err != nil {
		runtime.LogErrorf(app.ctx, "%v", app.db_pool.err)
		return Ok{}, app.db_pool.err
	}

	// SAFETY: Deleteing a user should only remove their access to the database.
	// All information about the user should be retained.
	tx, err := app.db_pool.ok.Begin(context.Background())
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not begin delete user transaction: %v", err)
		return Ok{}, err
	}
	defer tx.Rollback(context.Background())

	delete_user_auth_query := "DELETE FROM user_auth_ WHERE _id=$1"
	_, err = tx.Exec(context.Background(), delete_user_auth_query, user_id)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not remove user auth: %v", err)
		return Ok{}, err
	}

	set_user_status_query := "UPDATE user_ SET account_status='disabled' WHERE _id=$1"
	_, err = tx.Exec(context.Background(), set_user_status_query, user_id)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not remove user auth: %v", err)
		return Ok{}, err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not commit remove user auth transaction: %v", err)
	}

	return Ok{}, err
}

func (app *App) UpdateUser(update User) (Ok, error) {
	if app.db_pool.err != nil {
		runtime.LogErrorf(app.ctx, "%v", app.db_pool.err)
		return Ok{}, app.db_pool.err
	}

	update_user_query := "UPDATE user_ SET account_status=$2, email=$3, name=$4, role=$5 WHERE _id=$1"
	_, err := app.db_pool.ok.Exec(
		context.Background(),
		update_user_query,
		update.AccountStatus,
		update.Id,
		update.Email,
		update.Name,
		update.Role,
	)

	if err != nil {
		runtime.LogErrorf(app.ctx, "could not update user: %v", err)
	}

	return Ok{}, err
}

func (app *App) GetUsers() ([]User, error) {
	if app.db_pool.err != nil {
		return nil, app.db_pool.err
	}
	if app.state.user_id == uuid.Nil {
		return nil, &UserNotAuthenticedError{}
	}

	user_has_permission, err := app.user_has_role("owner")
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not get user permissions: %v", err)
	}
	if !user_has_permission {
		runtime.LogErrorf(app.ctx, "[user._id: %s] insufficient permissions for users list ", app.state.user_id)
		return nil, &InsufficientPermissionsError{}
	}

	users_query := "SELECT _id, account_status, email, name, role FROM user_ ORDER BY _id"
	user_rows, _ := app.db_pool.ok.Query(context.Background(), users_query)
	users, err := pgx.CollectRows(user_rows, func(row pgx.CollectableRow) (User, error) {
		var user User
		err := row.Scan(&user.Id, &user.AccountStatus, &user.Email, &user.Name, &user.Role)
		return user, err
	})
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not collect users: %v", err)
		return nil, err
	}

	return users, nil
}
