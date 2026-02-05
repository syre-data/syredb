package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"syredb/database"
	"syredb/handler"
	"syredb/service"
)

const DEV_FRONTEND_PATH = "../frontend/dist"
const JWT_CONTEXT_KEY string = "user"
const ENV_KEY = "SYREDB_ENV"
const INCLUDE_PATH = "../frontend/dist"

func main() {
	db_credentials, err := database.CollectCredentialsFromEnvAndFlags()
	if err != nil {
		panic(fmt.Errorf("could not obtain database credentials: #%v", err))
	}

	db, err := database.Connect(db_credentials)
	if err != nil {
		panic(fmt.Errorf("could not connect to database: #%v", err))
	}
	defer db.Close()

	ctx := context.Background()
	e := echo.New()

	auth_service := service.NewAuthService(ctx, e.Logger, db)
	user_service := service.NewUserService(ctx, e.Logger, db, auth_service)

	app_handler := handler.NewAppHandler(db)
	auth_handler := handler.NewAuthHandler(db, auth_service)
	user_handler := handler.NewUserHandler(db, user_service)

	env_session_secret, env_session_secret_exists := os.LookupEnv(handler.ENV_SESSION_SECRET_KEY)
	if !env_session_secret_exists {
		env_session_secret = "secret"
		os.Setenv(handler.ENV_SESSION_SECRET_KEY, env_session_secret)
	}

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:8080", "https://localhost:8080"},
		AllowCredentials: true,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))
	e.Use(echojwt.WithConfig(echojwt.Config{
		ContextKey:  handler.COOKIE_SESSION_TOKEN_KEY,
		SigningKey:  []byte(env_session_secret),
		TokenLookup: fmt.Sprintf("cookie:%s", handler.COOKIE_SESSION_TOKEN_KEY),
		NewClaimsFunc: func(c *echo.Context) jwt.Claims {
			return new(handler.JwtCustomClaims)
		},
		ErrorHandler: func(c *echo.Context, err error) error {
			c.Logger().With(
				"error", err,
				"cookies", c.Request().Cookies(),
			).Error("could not get jwt from cookie")

			if errors.Is(err, echojwt.ErrJWTInvalid) {
				return c.NoContent(http.StatusUnauthorized)
			}

			return nil
		},
		ContinueOnIgnoredError: true,
	}))
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       DEV_FRONTEND_PATH,
		Index:      "index.html",
		HTML5:      true,
		Browse:     false,
		IgnoreBase: false,
		Filesystem: nil,
		Skipper: middleware.Skipper(func(c *echo.Context) bool {
			_, jwt_err := echo.ContextGet[*jwt.Token](c, handler.COOKIE_SESSION_TOKEN_KEY)
			valid_token := jwt_err == nil

			path := c.Request().URL.Path
			api_call := len(path) > 3 && path[:4] == "/api"

			return !valid_token || api_call
		}),
	}))

	e.Static("public", "public")
	e.Renderer = &echo.TemplateRenderer{
		Template: template.Must(template.ParseGlob("templates/*.tmpl")),
	}

	register_routes(
		e,
		app_handler,
		auth_handler,
		user_handler,
	)
	if os.Getenv("ENV") != "production" {
		proxy_to_vite(e)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{
		Address:         ":3000",
		GracefulTimeout: 5 * time.Second,
	}
	if err := sc.Start(ctx, e); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

func register_routes(
	e *echo.Echo,
	app_handler *handler.AppHandler,
	auth_handler *handler.AuthHandler,
	user_handler *handler.UserHandler,
) {
	e.GET("/", app_handler.Index)
	e.POST("/api/login", auth_handler.Login)
	e.GET("/api/user", user_handler.GetUserFromToken)
}

func proxy_to_vite(e *echo.Echo) {
	vite_url, _ := url.Parse("http://localhost:5173")
	proxy := httputil.NewSingleHostReverseProxy(vite_url)

	e.GET("/*", echo.WrapHandler(proxy))
}

// //go:embed include/*
// var embeddedFiles embed.FS

// env_production := os.Getenv(ENV_KEY) == "production"
// file_system := http.FileServer(getFileSystem(env_production))

// func getFileSystem(production bool) http.FileSystem {
// 	if production {
// 		return http.FS(os.DirFS(INCLUDE_PATH))
// 	}

// 	fsys, err := fs.Sub(embeddedFiles, INCLUDE_PATH)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return http.FS(fsys)
// }
