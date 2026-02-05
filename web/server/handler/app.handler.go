package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"syredb/database"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type AppHandler struct {
	db *database.DbConnection
}

func NewAppHandler(
	db *database.DbConnection,
) *AppHandler {
	return &AppHandler{db: db}
}

func (h AppHandler) Index(c *echo.Context) error {
	_, err := echo.ContextGet[*jwt.Token](c, COOKIE_SESSION_TOKEN_KEY)
	if err != nil {
		return h.indexUnauthenticatedHandler(c)
	}

	panic("unreachable: session token validity checked in middleware")
}

func (h *AppHandler) indexUnauthenticatedHandler(c *echo.Context) error {
	const APP_ACCOUNT_NAME_KEY = "app:account:name"
	const APP_ACCOUNT_LOGO_KEY = "app:account:logo"
	const STATIC_FILE_PREFIX = "public"

	var account_name string
	var logo_path string
	app_query := fmt.Sprintf(
		`SELECT key, value FROM _app_data_ WHERE key IN ('%s', '%s')`,
		APP_ACCOUNT_NAME_KEY,
		APP_ACCOUNT_LOGO_KEY,
	)
	rows, err := h.db.Conn.Query(c.Request().Context(), app_query)
	if err != nil {
		c.Logger().With("error", err).Error("could not get app data")
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value string
		err := rows.Scan(&key, &value)
		if err != nil {
			c.Logger().With("error", err).Error("could not get app data")
		}

		switch key {
		case APP_ACCOUNT_NAME_KEY:
			account_name = value
		case APP_ACCOUNT_LOGO_KEY:
			logo_path = filepath.Join(STATIC_FILE_PREFIX, value)
		default:
			c.Logger().With("key", key).Error("invalid key")
			panic("invalid key")
		}
	}

	title := "SyreDB"
	if account_name != "" {
		title += " | " + account_name
	}

	data := struct {
		Title       string
		AccountName string
		IconPath    template.HTMLAttr
		LogoPath    template.HTMLAttr
	}{
		Title:       title,
		AccountName: account_name,
		IconPath:    template.HTMLAttr(fmt.Sprintf(`href="%s"`, logo_path)),
		LogoPath:    template.HTMLAttr(fmt.Sprintf(`src="%s"`, logo_path)),
	}
	return c.Render(http.StatusOK, "index", data)
}
