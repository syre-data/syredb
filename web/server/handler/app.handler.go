package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"syredb/database"
	"syredb/service"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type AppHandler struct {
	db          *database.DBConnection
	app_service *service.AppService
}

func NewAppHandler(
	db *database.DBConnection,
	app_service *service.AppService,
) *AppHandler {
	return &AppHandler{db: db, app_service: app_service}
}

func (h AppHandler) Index(c *echo.Context) error {
	_, err := echo.ContextGet[*jwt.Token](c, SessionTokenKey)
	if err != nil {
		return h.indexUnauthenticatedHandler(c)
	}

	panic("unreachable: session token validity checked in middleware")
}

func (h *AppHandler) indexUnauthenticatedHandler(c *echo.Context) error {
	const AppAccountNameKey = "app:account:name"
	const AppAccountLogoKey = "app:account:logo"
	const StaticFilePrefix = "public"

	var account_name string
	var logo_path string
	app_query := fmt.Sprintf(
		`SELECT key, value FROM _app_data_ WHERE key IN ('%s', '%s')`,
		AppAccountNameKey,
		AppAccountLogoKey,
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
		case AppAccountNameKey:
			account_name = value
		case AppAccountLogoKey:
			logo_path = filepath.Join(StaticFilePrefix, value)
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

func (h *AppHandler) DbPermissions(c *echo.Context) error {
	permissions, err := h.app_service.DbPermissionsAll()
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, permissions)
}
