package handler

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"syredb/database"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	auth_user_id := c.Get(UserIdKey).(uuid.UUID)
	user_id_str := c.QueryParam("id")
	var user_id uuid.UUID
	if user_id_str == "" {
		user_id = auth_user_id
	} else {
		parsed, err := uuid.Parse(user_id_str)
		if err != nil {
			c.Logger().With(
				"error", err,
				"user", auth_user_id,
				"id", user_id_str,
			).Error("could not parse user id")
			return c.NoContent(http.StatusBadRequest)
		}
		user_id = parsed
	}

	user, err := h.user_service.UserById(user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", auth_user_id,
		).Error("could not get user from session token")
		return c.NoContent(http.StatusUnauthorized)
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) guard_user_has_permission(c *echo.Context, permission service.DbPermission) error {
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
	err := h.guard_user_has_permission(c, service.DbPermissionUserModify)
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
	err := h.guard_user_has_permission(c, service.DbPermissionUserCreate)
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
			Code:    AppErrCodeEmailNotSent,
			Payload: user.Password,
		}
		return c.JSON(http.StatusInternalServerError, pwd_err)
	}

	return c.JSON(http.StatusOK, user_id)
}

func (h *UserHandler) UserUpdate(c *echo.Context) error {
	err := h.guard_user_has_permission(c, service.DbPermissionUserModify)
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
	err := h.guard_user_has_permission(c, service.DbPermissionUserModify)
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

func (h *UserHandler) PasswordReset(c *echo.Context) error {
	type payloadData struct {
		User uuid.UUID
	}

	err := h.guard_user_has_permission(c, service.DbPermissionUserModify)
	if err != nil {
		return err
	}

	var payload payloadData
	err = c.Bind(&payload)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	user, err := h.user_service.UserById(payload.User)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.NoContent(http.StatusUnprocessableEntity)
		} else {
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	password := rand.Text()
	err = h.user_service.PasswordUpdate(user.Id, password)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	const subject = "SyreDB | Password reset"
	message := fmt.Sprintf(
		`Your SyreDB password has been reset. 
		You can log in with this email and the password:
		%s
		
		You can change your password once you log in.`,
		password,
	)
	err = h.app_service.SendMail(user.Email, subject, message)
	if err != nil {
		c.Logger().With("error", err).Error("could not send password reset email")
		pwd_err := AppError{
			Code:    AppErrCodeEmailNotSent,
			Payload: password,
		}
		return c.JSON(http.StatusInternalServerError, pwd_err)
	}

	return c.NoContent(http.StatusOK)

}

func (h *UserHandler) PasswordUpdate(c *echo.Context) error {
	type payloadData struct {
		Current string
		Update  string
	}

	user_id := c.Get(UserIdKey).(uuid.UUID)

	var payload payloadData
	err := c.Bind(&payload)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	c.Logger().Error("TEST", "payload", payload)
	valid, err := h.user_service.Authenticate(user_id, payload.Current)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	if !valid {
		pwd_err := AppError{
			Code:    AppErrorCodeInvalidPassword,
			Message: "Current password is invalid",
		}
		return c.JSON(http.StatusUnauthorized, pwd_err)
	}

	password := payload.Update
	lowerCase := regexp.MustCompile(`[a-z]`)
	upperCase := regexp.MustCompile(`[A-Z]`)
	numberPtrn := regexp.MustCompile(`\d`)
	specialChar := regexp.MustCompile(`[!@#$%^&*]`)
	if len(password) < 8 ||
		len(password) > 32 ||
		!lowerCase.MatchString(password) ||
		!upperCase.MatchString(password) ||
		!numberPtrn.MatchString(password) ||
		!specialChar.MatchString(password) {
		pwd_err := AppError{
			Code:    AppErrorCodeUserNotAuthenticated,
			Message: "New password is invalid",
		}
		return c.JSON(http.StatusUnauthorized, pwd_err)
	}

	err = h.user_service.PasswordUpdate(user_id, password)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
