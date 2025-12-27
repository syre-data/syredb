package main

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"go.linka.cloud/go-appdir"

	sdb "syredb/app"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const LOG_FILE = "syredb.log"

//go:embed frontend/dist
var assets embed.FS

func main() {
	config_dir_path := get_config_dir()
	if !path_exists(config_dir_path) {
		os.MkdirAll(config_dir_path, sdb.FILE_PERMISSIONS_WRR)
	}

	log_file_path := filepath.Join(config_dir_path, LOG_FILE)
	log_file, err := os.OpenFile(log_file_path, os.O_CREATE|os.O_WRONLY, sdb.FILE_PERMISSIONS_WRR)
	if err != nil {
		panic(fmt.Sprintf("could not open log file: %v", err))
	}
	defer log_file.Close()

	logger_writer := io.MultiWriter(os.Stdout, log_file)
	logger_opts := slog.HandlerOptions{AddSource: true, Level: slog.LevelError}
	logger := slog.New(slog.NewJSONHandler(logger_writer, &logger_opts))

	app := application.New(application.Options{
		Name:     "SyreDB",
		Logger:   logger,
		Services: []application.Service{},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                  "SyreDB",
		Width:                  1024,
		Height:                 768,
		BackgroundColour:       application.NewRGB(0, 0, 0),
		OpenInspectorOnStartup: true,
	})

	db := &sdb.DbConnection{}
	app_service := sdb.NewAppService(logger, db, config_dir_path)
	auth_service := sdb.NewAuthService(logger, db, config_dir_path, app_service.AppState())
	user_service := sdb.NewUserService(logger, db, app_service.AppState())
	project_service := sdb.NewProjectService(logger, db, app_service.AppState())
	data_service := sdb.NewDataService(logger, db, app_service.AppState())

	app.RegisterService(application.NewService(app_service))
	app.RegisterService(application.NewService(auth_service))
	app.RegisterService(application.NewService(user_service))
	app.RegisterService(application.NewService(project_service))
	app.RegisterService(application.NewService(data_service))

	err = app.Run()
	if err != nil {
		panic(err)
	}
}

// Get app config directory path.
func get_config_dir() string {
	dirs := appdir.New(sdb.APP_NAME)
	return dirs.UserConfig()
}

func path_exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
