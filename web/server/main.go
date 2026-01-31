package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syredb/database"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	err := database.Connect()
	if err != nil {
		panic(err)
	}
	defer database.Close()

	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.Static("public", "public")
	e.Renderer = &echo.TemplateRenderer{
		Template: template.Must(template.ParseGlob("templates/*.tmpl")),
	}

	e.GET("/", IndexHandler)
	e.POST("/api/login", LoginHandler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{
		Address:         ":8080",
		GracefulTimeout: 5 * time.Second,
	}
	if err := sc.Start(ctx, e); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

func IndexHandler(c *echo.Context) error {
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
	rows, err := database.Conn.Query(c.Request().Context(), app_query)
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
		c.Logger().With("value", value, "key", key).Error("vars")

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
		LogoPath    template.HTMLAttr
	}{
		Title:       title,
		AccountName: account_name,
		LogoPath:    template.HTMLAttr(fmt.Sprintf(`src="%s"`, logo_path)),
	}
	return c.Render(http.StatusOK, "index", data)
}

func LoginHandler(c *echo.Context) error {
	// email := c.FormValue("email")
	// password := c.FormValue("password")

	return c.HTML(http.StatusOK, "hi")
}
