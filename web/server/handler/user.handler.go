package handler

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
			).Info("could not parse user id")
			return c.JSON(http.StatusUnprocessableEntity, AppError{
				RequestId: requestId(c),
				Code:      APP_ERR_INVALID_UUID,
				Message:   "Invalid id query",
				Payload:   nil,
			})
		}
		user_id = parsed
	}

	user, valid, err := h.user_service.UserById(user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", auth_user_id,
		).Error("could not get user from session token")
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not get user",
			Payload:   nil,
		})
	}

	if !valid {
		return c.NoContent(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) guardUserHasPermission(c *echo.Context, permission service.DbPermission) error {
	user_id := c.Get(UserIdKey).(uuid.UUID)
	has_permission, err := h.user_service.UserHasPermission(user_id, permission)
	if err != nil {
		c.Logger().With("error", err).Error("could not verify user permission")
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not verify user permission",
			Payload:   nil,
		})
	}
	if !has_permission {
		c.Logger().With(
			"user", user_id,
		).Info("insufficient permission for project resources")
		return c.NoContent(http.StatusForbidden)
	}

	return nil
}

func (h *UserHandler) UsersAll(c *echo.Context) error {
	err := h.guardUserHasPermission(c, service.DbPermissionUserModify)
	if err != nil {
		return err
	}

	users, err := h.user_service.AllUsers()
	if err != nil {
		c.Logger().With("error", err).Error("could not get user")
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not verify user permission",
			Payload:   c.Get("request_id"),
		})
	}

	return c.JSON(http.StatusOK, users)
}

func (h *UserHandler) UserCreate(c *echo.Context) error {
	err := h.guardUserHasPermission(c, service.DbPermissionUserCreate)
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
		return c.JSON(http.StatusBadRequest, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_INVALID_PAYLOAD,
			Message:   "Could not create user",
			Payload:   nil,
		})
	}

	user.Password = rand.Text()
	user_id, err := h.user_service.CreateUser(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == string(DB_ERR_DUPLICATE_RECORD) {
				return c.NoContent(http.StatusConflict)
			}
		} else {
			c.Logger().With(
				"error", err,
				"user", user,
			).Error("could not create user")
			return c.JSON(http.StatusInternalServerError, AppError{
				RequestId: requestId(c),
				Code:      APP_ERR_DB_QUERY,
				Message:   "Could not create user",
				Payload:   nil,
			})
		}
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
		return c.JSON(http.StatusInternalServerError, AppError{
			Code:    APP_ERR_EMAIL_NOT_SENT,
			Payload: user.Password,
		})
	}

	res := ResourceCreated{Id: user_id}
	loc := url.URL{Path: "/api/user"}
	loc.RawQuery = res.ToUrlQuery().Encode()
	c.Response().Header().Set(echo.HeaderLocation, loc.String())
	return c.JSON(http.StatusCreated, res)
}

func (h *UserHandler) UserUpdate(c *echo.Context) error {
	err := h.guardUserHasPermission(c, service.DbPermissionUserModify)
	if err != nil {
		return err
	}

	update := new(service.User)
	err = c.Bind(update)
	if err != nil {
		c.Logger().With("error", err).Error("could not bind request data")
		return c.JSON(http.StatusBadRequest, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_INVALID_PAYLOAD,
			Message:   "Invalid request payload",
			Payload:   nil,
		})
	}
	err = h.user_service.UserUpdate(*update)
	if err != nil {
		c.Logger().With(
			"error", err,
			"update", update,
		).Error("could not update user")
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not update user",
			Payload:   nil,
		})
	}

	return nil
}

func (h *UserHandler) DeactivateUser(c *echo.Context) error {
	err := h.guardUserHasPermission(c, service.DbPermissionUserModify)
	if err != nil {
		return err
	}
	user_id := c.Get(UserIdKey).(uuid.UUID)

	user_to_deactivate, err := uuid.Parse(c.FormValue("user"))
	if err != nil {
		c.Logger().With(
			"active user", user_id,
			"deactivate user", c.FormValue("user"),
		).Info("could not parse user id")
		return c.JSON(http.StatusUnprocessableEntity, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_INVALID_UUID,
			Message:   "Invalid user query",
			Payload:   nil,
		})
	}

	// SAFETY: User must not be able to deactivate themselves.
	if user_id == user_to_deactivate {
		return c.JSON(http.StatusUnprocessableEntity, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_LOGICAL,
			Message:   "A user can not deactivate themselves",
			Payload:   nil,
		})
	}

	err = h.user_service.DeactivateUser(user_to_deactivate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not deactivate user",
			Payload:   nil,
		})
	}

	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) PasswordReset(c *echo.Context) error {
	type payloadData struct {
		User uuid.UUID
	}

	err := h.guardUserHasPermission(c, service.DbPermissionUserModify)
	if err != nil {
		return err
	}

	var payload payloadData
	err = c.Bind(&payload)
	if err != nil {
		return c.JSON(http.StatusBadRequest, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_INVALID_PAYLOAD,
			Message:   "Invalid request payload",
			Payload:   payload,
		})
	}

	user, valid, err := h.user_service.UserById(payload.User)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not get user",
			Payload:   nil,
		})
	}
	if !valid {
		return c.JSON(http.StatusUnprocessableEntity, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_ENTITY_NOT_FOUND,
			Message:   "User does not exist",
			Payload:   nil,
		})
	}

	password := rand.Text()
	err = h.user_service.PasswordUpdate(user.Id, password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Password could not be updated",
			Payload:   payload,
		})
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
		return c.JSON(http.StatusInternalServerError, AppError{
			Code:    APP_ERR_EMAIL_NOT_SENT,
			Payload: password,
		})
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
		return c.JSON(http.StatusBadRequest, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_INVALID_PAYLOAD,
			Message:   "Invalid request payload",
			Payload:   payload,
		})
	}

	valid, err := h.user_service.Authenticate(user_id, payload.Current)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not authenticate user",
			Payload:   nil,
		})
	}
	if !valid {
		return c.NoContent(http.StatusUnauthorized)
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
		return c.JSON(http.StatusBadRequest, AppError{
			Code:    APP_ERR_INVALID_PASSWORD,
			Message: "New password is invalid",
		})
	}

	err = h.user_service.PasswordUpdate(user_id, password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AppError{
			RequestId: requestId(c),
			Code:      APP_ERR_DB_QUERY,
			Message:   "Could not update password",
			Payload:   payload,
		})
	}

	return c.NoContent(http.StatusOK)
}
