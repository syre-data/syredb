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
	"github.com/google/uuid"
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
	api_middleware := NewApiMiddleware(ctx, db)

	auth_service := service.NewAuthService(ctx, e.Logger, db)
	user_service := service.NewUserService(ctx, e.Logger, db, auth_service)
	project_service := service.NewProjectService(ctx, e.Logger, db)
	data_service := service.NewDataService(ctx, e.Logger, db)

	app_handler := handler.NewAppHandler(db)
	auth_handler := handler.NewAuthHandler(db, auth_service)
	user_handler := handler.NewUserHandler(db, user_service)
	project_handler := handler.NewProjectHandler(db, project_service)
	data_handler := handler.NewDataHandler(db, data_service)

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
		ContextKey:  handler.SESSION_TOKEN_KEY,
		SigningKey:  []byte(env_session_secret),
		TokenLookup: fmt.Sprintf("cookie:%s", handler.SESSION_TOKEN_KEY),
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
			_, jwt_err := echo.ContextGet[*jwt.Token](c, handler.SESSION_TOKEN_KEY)
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
	db  *database.DbConnection
}

func NewApiMiddleware(ctx context.Context, db *database.DbConnection) *ApiMiddleware {
	return &ApiMiddleware{ctx: ctx, db: db}
}

func (m *ApiMiddleware) SessionTokenFromJwt(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		token, err := echo.ContextGet[*jwt.Token](c, handler.SESSION_TOKEN_KEY)
		if err != nil {
			c.Logger().With(
				"error", err,
				"token", token,
			).Error("invalid jwt token")
			return err
		}

		claims := token.Claims.(*handler.JwtCustomClaims)
		c.Set(handler.SESSION_TOKEN_KEY, claims.SessionId)
		return next(c)
	}
}

func (m *ApiMiddleware) UserIdFromSessionToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		session_token := c.Get(handler.SESSION_TOKEN_KEY)
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

		c.Set(handler.USER_ID_KEY, user)
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

	api := e.Group("/api")
	api.Use(api_middleware.SessionTokenFromJwt)
	api.Use(api_middleware.UserIdFromSessionToken)

	api.POST("/login", auth.Login)
	api.GET("/user", user.GetUserFromToken)
	api.GET("/projects", project.GetUserProjects)
	api.GET("/data-schemas", data.GetDataSchemasAll)
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
