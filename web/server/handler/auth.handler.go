package handler

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"syredb/database"
	"syredb/service"
)

const EnvSessionSecretKey string = "SYREDB_SESSION_SECRET"
const SessionTokenKey string = "session_token"
const SessionDuration = time.Hour * 24

type JWTCustomClaims struct {
	SessionId uuid.UUID `json:"session_id"`
	jwt.RegisteredClaims
}

type AuthHandler struct {
	db           *database.DBConnection
	auth_service *service.AuthService
}

func NewAuthHandler(
	db *database.DBConnection,
	auth_service *service.AuthService,
) *AuthHandler {
	return &AuthHandler{db: db, auth_service: auth_service}
}

func (h *AuthHandler) Login(c *echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	remember := c.FormValue("remember") != ""

	if email == "" || password == "" {
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	var user_id uuid.UUID
	user_id_query := "SELECT _id FROM user_ WHERE email=$1"
	err := h.db.Conn.QueryRow(c.Request().Context(), user_id_query, email).Scan(&user_id)
	if err != nil {
		c.Logger().With("error", err).Error("could not retrieve user id")
		return c.NoContent(http.StatusNotFound)
	}

	var hash string
	auth_query := "SELECT auth FROM user_auth_ WHERE _id=$1"
	err = h.db.Conn.QueryRow(c.Request().Context(), auth_query, user_id).Scan(&hash)
	if err != nil {
		c.Logger().With("error", err).Error("could not retrieve user auth hash")
		return c.NoContent(http.StatusNotFound)
	}

	authenticated, err := h.auth_service.ComparePasswordAndHash(password, hash)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user id", user_id,
		).Error("could not authenticate user")
		return c.NoContent(http.StatusInternalServerError)
	}

	if !authenticated {
		return echo.ErrUnauthorized
	}

	var session_expiration time.Time
	if remember {
		session_expiration = time.Now().Add(time.Hour * 24 * 365)
	} else {
		session_expiration = time.Now().Add(SessionDuration)
	}

	session_id, err := h.auth_service.CreateSession(user_id, session_expiration)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"expires", session_expiration,
		).Error("could not create user session")
		return c.NoContent(http.StatusInternalServerError)
	}

	registered_claims := jwt.RegisteredClaims{}
	if !remember {
		registered_claims.ExpiresAt = jwt.NewNumericDate(session_expiration)
	}
	claims := &JWTCustomClaims{
		// TODO: add struct field names.
		session_id,
		registered_claims,
	}

	session_secret, session_secret_exists := os.LookupEnv(EnvSessionSecretKey)
	if !session_secret_exists {
		panic(fmt.Sprintf("%s does not exist, can not sign token", EnvSessionSecretKey))
	}
	token_data := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := token_data.SignedString([]byte(session_secret))
	if err != nil {
		return err
	}

	// TODO: Check security.
	cookie := &http.Cookie{
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		Domain:   "",
		SameSite: http.SameSiteLaxMode,
		Name:     SessionTokenKey,
		Value:    token,
		Expires:  session_expiration,
	}
	c.SetCookie(cookie)

	return c.NoContent(http.StatusOK)
}

func (h *AuthHandler) Logout(c *echo.Context) error {
	session_token := c.Get(SessionTokenKey).(uuid.UUID)
	err := h.auth_service.DeactivateSession(session_token)
	if err != nil {
		c.Logger().With("session token", session_token).Error("could not deactivate session")
		return c.NoContent(http.StatusInternalServerError)
	}

	cookie := &http.Cookie{
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		Domain:   "",
		SameSite: http.SameSiteLaxMode,
		Name:     SessionTokenKey,
		Value:    "",
		MaxAge:   -1,
	}
	c.SetCookie(cookie)
	return c.NoContent(http.StatusOK)
}
