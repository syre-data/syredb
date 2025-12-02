package main

import (
	"embed"
	"errors"
	"log"
	"os"
	"path/filepath"

	"go.linka.cloud/go-appdir"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

const LOG_FILE = "syredb.log"
const PERMISSIONS_WRR = 0644

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	conf_dir_path := getConfigDir()
	if !pathExists(conf_dir_path) {
		os.MkdirAll(conf_dir_path, PERMISSIONS_WRR)
	}

	log_file_path := filepath.Join(conf_dir_path, LOG_FILE)
	app := NewApp()
	err := wails.Run(&options.App{
		Title:  "syredb",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
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

// Get app config directory path.
func getConfigDir() string {
	dirs := appdir.New(APP_NAME)
	return dirs.UserConfig()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
