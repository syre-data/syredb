package handler

import (
	"net/http"
	"syredb/database"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

const USER_ID_KEY string = "user_id"

type UserHandler struct {
	db           *database.DbConnection
	user_service *service.UserService
}

func NewUserHandler(
	db *database.DbConnection,
	user_service *service.UserService,
) *UserHandler {
	return &UserHandler{db: db, user_service: user_service}
}

func (h *UserHandler) GetUserFromToken(c *echo.Context) error {
	maybe_user_id := c.Get(USER_ID_KEY)
	if maybe_user_id == nil {
		panic("user id not set")
	}
	user_id := maybe_user_id.(uuid.UUID)

	user, err := h.user_service.UserById(user_id)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", maybe_user_id,
		).Error("could not get user from session token")
		return c.NoContent(http.StatusUnauthorized)
	}

	return c.JSON(http.StatusOK, user)
}
