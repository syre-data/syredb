package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"syredb/database"
	"syredb/handler"
	"syredb/service"
)

const devFrontendPath = "../frontend/dist"
const JWTContextKey string = "user"
const EnvKey = "SYREDB_ENV"
const IncludePath = "../frontend/dist"

func main() {
	ctx := context.Background()
	db_credentials, err := database.CollectCredentialsFromEnvAndFlags()
	if err != nil {
		panic(fmt.Errorf("could not obtain database credentials: #%v", err))
	}

	db, err := database.Connect(db_credentials)
	if err != nil {
		panic(fmt.Errorf("could not connect to database: #%v", err))
	}
	defer db.Close()

	e := echo.New()
	api_middleware := NewApiMiddleware(ctx, db)

	app_service := service.NewAppService(ctx, e.Logger, db)
	auth_service := service.NewAuthService(ctx, e.Logger, db)
	user_service := service.NewUserService(ctx, e.Logger, db, auth_service)
	sample_service := service.NewSampleService(ctx, e.Logger, db)
	data_service := service.NewDataService(ctx, e.Logger, db, app_service, user_service)
	project_service := service.NewProjectService(ctx, e.Logger, db, user_service, data_service)

	app_handler := handler.NewAppHandler(db, app_service)
	auth_handler := handler.NewAuthHandler(db, auth_service)
	user_handler := handler.NewUserHandler(db, user_service, app_service)
	project_handler := handler.NewProjectHandler(db, project_service, user_service, sample_service)
	data_handler := handler.NewDataHandler(db, data_service, user_service, project_service)

	transform_daemon_logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	transform_daemon := NewTransformDaemon(ctx, transform_daemon_logger, db, app_service, data_service)
	go transform_daemon.Start(ctx)

	env_session_secret, env_session_secret_exists := os.LookupEnv(handler.EnvSessionSecretKey)
	if !env_session_secret_exists {
		env_session_secret = "secret"
		os.Setenv(handler.EnvSessionSecretKey, env_session_secret)
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
		ContextKey:  handler.SessionTokenKey,
		SigningKey:  []byte(env_session_secret),
		TokenLookup: fmt.Sprintf("cookie:%s", handler.SessionTokenKey),
		NewClaimsFunc: func(c *echo.Context) jwt.Claims {
			return new(handler.JWTCustomClaims)
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

	static_root, err := filepath.Abs(devFrontendPath)
	if err != nil {
		panic(err)
	}
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       static_root,
		Index:      "index.html",
		HTML5:      true,
		Browse:     false,
		IgnoreBase: false,
		Filesystem: nil,
		Skipper: middleware.Skipper(func(c *echo.Context) bool {
			_, jwt_err := echo.ContextGet[*jwt.Token](c, handler.SessionTokenKey)
			valid_token := jwt_err == nil

			path := c.Request().URL.Path
			api_call := len(path) > 3 && path[:4] == "/api"

			return !valid_token || api_call
		}),
	}))

	e.Static("public", "_public")
	e.Renderer = &echo.TemplateRenderer{
		Template: template.Must(template.ParseGlob("templates/*.tmpl")),
	}

	register_routes(
		e,
		api_middleware,
		app_handler,
		auth_handler,
		user_handler,
		project_handler,
		data_handler,
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

type ApiMiddleware struct {
	ctx context.Context
	db  *database.DBConnection
}

func NewApiMiddleware(ctx context.Context, db *database.DBConnection) *ApiMiddleware {
	return &ApiMiddleware{ctx: ctx, db: db}
}

func (m *ApiMiddleware) SessionTokenFromJWT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		token, err := echo.ContextGet[*jwt.Token](c, handler.SessionTokenKey)
		if err != nil {
			c.Logger().With(
				"error", err,
				"token", token,
			).Error("invalid jwt token")
			return err
		}

		claims := token.Claims.(*handler.JWTCustomClaims)
		c.Set(handler.SessionTokenKey, claims.SessionId)
		return next(c)
	}
}

func (m *ApiMiddleware) UserIdFromSessionToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		session_token := c.Get(handler.SessionTokenKey)
		if session_token == nil {
			panic("session token not set on context")
		}
		session_token = session_token.(uuid.UUID)

		var user uuid.UUID
		query := "SELECT _user FROM _user_session_ WHERE _token=$1 AND _expires>$2 AND active=true"
		err := m.db.Conn.QueryRow(m.ctx, query, session_token, time.Now()).Scan(&user)
		if err != nil {
			c.Logger().With(
				"error", err,
				"token", session_token,
			).Error("could not get session user")
			return err
		}

		c.Set(handler.UserIdKey, user)
		return next(c)
	}
}

func register_routes(
	e *echo.Echo,
	api_middleware *ApiMiddleware,
	app *handler.AppHandler,
	auth *handler.AuthHandler,
	user *handler.UserHandler,
	project *handler.ProjectHandler,
	data *handler.DataHandler,
) {
	e.GET("/", app.Index)
	e.POST("/api/login", auth.Login)
	e.GET("/logout", auth.Logout)

	api := e.Group("/api")
	api.Use(api_middleware.SessionTokenFromJWT)
	api.Use(api_middleware.UserIdFromSessionToken)

	api.GET("/app/db-permissions", app.DbPermissions)
	api.GET("/user", user.UserGet)
	api.PUT("/user", user.UserUpdate)
	api.POST("/user/create", user.UserCreate)
	api.PUT("/user/deactivate", user.DeactivateUser)
	api.GET("/users", user.UsersAll)
	api.GET("/project", project.GetProjectWithUserPermission)
	api.POST("/project", project.CreateProject)
	api.GET("/projects", project.GetUserProjects)
	api.GET("/project/resources", project.ProjectResources)
	api.GET("/project/sample-resources", project.GetProjectSampleResources)
	api.POST("/project/samples", project.CreateProjectSamples)
	api.PUT("/project/sample", project.UpdateProjectSample)
	api.GET("/data-schemas", data.DataSchemasGetAll)
	api.POST("/data-schema", data.DataSchemaCreate)
	api.GET("/data-schema", data.DataSchemaResources)
	api.PUT("/data-schema", data.DataSchemaUpdate)
	api.GET("/data-types", data.DataTypesGetAll)
	api.POST("/data-type", data.DataTypeCreate)
	api.GET("/data-type", data.DataTypeGet)
	api.PUT("/data-type", data.DataTypeUpdate)
	api.GET("/data-type-transforms", data.DataTypeTransformsGetAll)
	api.POST("/data-type-transform", data.DataTypeTransformCreate)
	api.GET("/sample-data/single", data.DownloadRawDataSingle)
	api.GET("/sample-data/project", data.DownloadRawDataProject)
}

func proxy_to_vite(e *echo.Echo) {
	vite_url, _ := url.Parse("http://localhost:5173")
	proxy := httputil.NewSingleHostReverseProxy(vite_url)

	e.GET("/*", echo.WrapHandler(proxy))
}

// //go:embed _include/*
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
