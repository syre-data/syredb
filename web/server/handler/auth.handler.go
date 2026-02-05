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

const ENV_SESSION_SECRET_KEY = "SYREDB_SESSION_SECRET"
const COOKIE_SESSION_TOKEN_KEY string = "session_token"
const SESSION_DURATION = time.Hour * 24

type JwtCustomClaims struct {
	SessionId uuid.UUID `json:"session_id"`
	jwt.RegisteredClaims
}

type AuthHandler struct {
	db   *database.DbConnection
	auth *service.AuthService
}

func NewAuthHandler(
	db *database.DbConnection,
	auth_service *service.AuthService,
) *AuthHandler {
	return &AuthHandler{db: db, auth: auth_service}
}

func (h *AuthHandler) Login(c *echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	remember := c.FormValue("remember") != ""

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

	authenticated, err := h.auth.ComparePasswordAndHash(password, hash)
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
		session_expiration = time.Now().Add(SESSION_DURATION)
	}

	session_id, err := h.auth.CreateSession(user_id, session_expiration)
	if err != nil {
		c.Logger().With("user", user_id, "expires", session_expiration).Error("could not create user session")
		return c.NoContent(http.StatusInternalServerError)
	}

	registered_claims := jwt.RegisteredClaims{}
	if !remember {
		registered_claims.ExpiresAt = jwt.NewNumericDate(session_expiration)
	}
	claims := &JwtCustomClaims{
		// TODO: add struct fields.
		session_id,
		registered_claims,
	}

	session_secret, session_secret_exists := os.LookupEnv(ENV_SESSION_SECRET_KEY)
	if !session_secret_exists {
		panic(fmt.Sprintf("%s does not exist, can not sign token", ENV_SESSION_SECRET_KEY))
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
		Name:     COOKIE_SESSION_TOKEN_KEY,
		Value:    token,
		Expires:  session_expiration,
	}
	c.SetCookie(cookie)

	return c.NoContent(http.StatusOK)
}
