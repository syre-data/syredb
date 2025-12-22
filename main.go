package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"syredb/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

const LOG_FILE = "syredb.log"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	conf_dir_path := app.GetConfigDir()
	if !app.PathExists(conf_dir_path) {
		os.MkdirAll(conf_dir_path, app.FILE_PERMISSIONS_WRR)
	}

	log_file_path := filepath.Join(conf_dir_path, LOG_FILE)
	app := app.NewApp()
	err := wails.Run(&options.App{
		Title:  "syredb",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
		Logger:             logger.NewFileLogger(log_file_path),
		LogLevel:           logger.TRACE,
		LogLevelProduction: logger.ERROR,
		Debug:              options.Debug{OpenInspectorOnStartup: true},
	})

	if err != nil {
		log.Fatal(err)
	}
}
