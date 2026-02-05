package handler

import (
	"net/http"
	"syredb/database"
	"syredb/service"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

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
	token, err := echo.ContextGet[*jwt.Token](c, COOKIE_SESSION_TOKEN_KEY)
	if err != nil {
		c.Logger().With("token", token).Error("invalid jwt token")
		return c.NoContent(http.StatusUnauthorized)
	}

	claims := token.Claims.(*JwtCustomClaims)
	user, err := h.user_service.UserFromToken(claims.SessionId)
	if err != nil {
		c.Logger().With("token", token).Error("could not get user from session token")
		return c.NoContent(http.StatusUnauthorized)
	}

	return c.JSON(http.StatusOK, user)
}
