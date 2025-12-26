package main

import (
	"embed"
	"fmt"
	"io"
	"log/slog"
	"os"

	"path/filepath"

	sdb "syredb/app"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const LOG_FILE = "syredb.log"

//go:embed frontend/dist
var assets embed.FS

func main() {
	conf_dir_path := sdb.GetConfigDir()
	if !sdb.PathExists(conf_dir_path) {
		os.MkdirAll(conf_dir_path, sdb.FILE_PERMISSIONS_WRR)
	}

	log_file_path := filepath.Join(conf_dir_path, LOG_FILE)
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
		OnShutdown: func() {
			//TODO
			// app.db_pool.ok.Close()
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                  "SyreDB",
		Width:                  1024,
		Height:                 768,
		BackgroundColour:       application.NewRGB(0, 0, 0),
		OpenInspectorOnStartup: true,
	})

	app.RegisterService(application.NewService(sdb.NewAppService(app)))

	err = app.Run()
	if err != nil {
		panic(err)
	}
}
