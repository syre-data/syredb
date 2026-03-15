package handler

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"syredb/database"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v5"
)

const UserIdKey string = "user_id"

type UserHandler struct {
	db           *database.DBConnection
	user_service *service.UserService
	app_service  *service.AppService
}

func NewUserHandler(
	db *database.DBConnection,
	user_service *service.UserService,
	app_service *service.AppService,
) *UserHandler {
	return &UserHandler{
		db:           db,
		user_service: user_service,
		app_service:  app_service,
	}
}

func (h *UserHandler) UserGet(c *echo.Context) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	user, err := h.user_service.UserById(user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
		).Error("could not get user from session token")
		return c.NoContent(http.StatusUnauthorized)
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) guard_user_has_permission(c *echo.Context, permission service.DbPermissionId) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	is_owner, err := h.user_service.UserHasPermission(user_id, permission)
	if err != nil {
		c.Logger().With("error", err).Error("could not verify user permission")
		return c.NoContent(http.StatusInternalServerError)
	}
	if !is_owner {
		return c.NoContent(http.StatusUnauthorized)
	}

	return nil
}

func (h *UserHandler) UsersAll(c *echo.Context) error {
	err := h.guard_user_has_permission(c, service.DbPermissionIdUserModify)
	if err != nil {
		return err
	}

	users, err := h.user_service.AllUsers()
	if err != nil {
		c.Logger().With("error", err).Error("could not get user")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, users)
}

func (h *UserHandler) UserCreate(c *echo.Context) error {
	err := h.guard_user_has_permission(c, service.DbPermissionIdUserCreate)
	if err != nil {
		return err
	}

	var user service.UserCreate
	err = c.Bind(&user)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user,
			"request body", c.Request().Body,
		).Error("could not bind request data")
		return c.NoContent(http.StatusBadRequest)
	}

	user.Password = rand.Text()
	user_id, err := h.user_service.CreateUser(user)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user,
		).Error("could not create user")
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == string(DBErrCodeDuplicateRecord) {
				return c.NoContent(http.StatusConflict)
			}
		}
		return c.NoContent(http.StatusInternalServerError)
	}

	const subject = "SyreDB | Welcome!"
	message := fmt.Sprintf(
		`Welcome to SyreDB. 
		You can log in with this email and the password:
		%s
		
		You can change your password once you log in.`,
		user.Password,
	)
	err = h.app_service.SendMail(user.Email, subject, message)
	if err != nil {
		c.Logger().With("error", err).Error("could not send user creation email")
		pwd_err := AppError{
			Code:    AppErrCodeUserWelcomeEmailNotSent,
			Payload: user.Password,
		}
		return c.JSON(http.StatusInternalServerError, pwd_err)
	}

	return c.JSON(http.StatusOK, user_id)
}

func (h *UserHandler) UserUpdate(c *echo.Context) error {
	err := h.guard_user_has_permission(c, service.DbPermissionIdUserModify)
	if err != nil {
		return err
	}

	update := new(service.User)
	err = c.Bind(update)
	c.Logger().With("update", update).Error("UPDATE")
	if err != nil {
		c.Logger().With("error", err).Error("could not bind request data")
		return c.NoContent(http.StatusBadRequest)
	}
	err = h.user_service.UserUpdate(*update)
	if err != nil {
		c.Logger().With("error", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return nil
}

func (h *UserHandler) DeactivateUser(c *echo.Context) error {
	err := h.guard_user_has_permission(c, service.DbPermissionIdUserModify)
	if err != nil {
		return err
	}
	user_id := c.Get(UserIdKey).(uuid.UUID)

	user_to_deactivate, err := uuid.Parse(c.FormValue("user"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	// SAFETY: User shall not be able to deactivate themselves.
	if user_id == user_to_deactivate {
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	err = h.user_service.DeactivateUser(user_to_deactivate)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
