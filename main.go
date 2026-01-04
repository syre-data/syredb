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
	"github.com/wailsapp/wails/v3/pkg/events"
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
	log_file, err := os.OpenFile(
		log_file_path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		sdb.FILE_PERMISSIONS_WRR,
	)
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

	w_main := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                  "SyreDB",
		Width:                  1024,
		Height:                 768,
		BackgroundColour:       application.NewRGB(0, 0, 0),
		EnableFileDrop:         true,
		OpenInspectorOnStartup: true,
	})

	w_main.OnWindowEvent(
		events.Common.WindowFilesDropped,
		func(event *application.WindowEvent) {},
	)

	db := &sdb.DbConnection{}
	fs_service := sdb.NewFsService(app, logger)
	app_service := sdb.NewAppService(logger, db, config_dir_path)
	auth_service := sdb.NewAuthService(logger, db, config_dir_path, app_service.AppState())
	user_service := sdb.NewUserService(logger, db, app_service.AppState())
	data_service := sdb.NewDataService(logger, db, app_service.AppState(), user_service)
	project_service := sdb.NewProjectService(logger, db, app_service.AppState(), data_service)

	app.RegisterService(application.NewService(fs_service))
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

// w_main.OnWindowEvent(
// 	events.Common.WindowDropZoneFilesDropped,
// 	func(event *application.WindowEvent) {

// 		droppedFiles := event.Context().DroppedFiles()
// 		details := event.Context().DropZoneDetails()

// 		logger.Error("Dropped files count", "count", len(droppedFiles))
// 		logger.Error("Event context", "ctx", event.Context())

// 		if details != nil {
// 			logger.Error("DropZone details found:")
// 			logger.Error("  ElementID:", "id", details.ElementID)
// 			logger.Error("  ClassList:", "classes", details.ClassList)
// 			logger.Error("  ", "x", details.X, "y", details.Y)
// 			logger.Error("  Attributes", "attrs", details.Attributes)

// 			// Call the App method with the extracted data
// 			FilesDroppedOnTarget(
// 				logger,
// 				droppedFiles,
// 				details.ElementID,
// 				details.ClassList,
// 				float64(details.X),
// 				float64(details.Y),
// 				details.ElementID != "", // isTargetDropzone based on whether an ID was found
// 				details.Attributes,
// 			)
// 		} else {
// 			logger.Error("DropZone details are nil - drop was not on a specific registered zone")
// 			// This case might occur if DropZoneDetails are nil, meaning the drop was not on a specific registered zone
// 			// or if the context itself was problematic.
// 			FilesDroppedOnTarget(logger, droppedFiles, "", nil, 0, 0, false, nil)
// 		}

// 		payload := FileDropInfo{
// 			Files:         droppedFiles,
// 			TargetID:      details.ElementID,
// 			TargetClasses: details.ClassList,
// 			DropX:         float64(details.X),
// 			DropY:         float64(details.Y),
// 			Attributes:    details.Attributes, // Add the attributes
// 		}

// 		logger.Error("Emitting event payload", "payload", payload)
// 		application.Get().Event.Emit("frontend:FileDropInfo", payload)
// 		logger.Error(
// 			"=============== End WindowDropZoneFilesDropped Event Debug ===============",
// 		)
// 	},
// )

// // FileDropInfo defines the payload for the file drop event sent to the frontend.
// type FileDropInfo struct {
// 	Files         []string          `json:"files"`
// 	TargetID      string            `json:"targetID"`
// 	TargetClasses []string          `json:"targetClasses"`
// 	DropX         float64           `json:"dropX"`
// 	DropY         float64           `json:"dropY"`
// 	Attributes    map[string]string `json:"attributes,omitempty"`
// }
// // FilesDroppedOnTarget is called when files are dropped onto a registered drop target
// // or the window if no specific target is hit.
// func FilesDroppedOnTarget(
// 	logger *slog.Logger,
// 	files []string,
// 	targetID string,
// 	targetClasses []string,
// 	dropX float64,
// 	dropY float64,
// 	isTargetDropzone bool, // This parameter is kept for logging but not sent to frontend in this event
// 	attributes map[string]string,
// ) {
// 	logger.Error("=============== Go: FilesDroppedOnTarget Debug Info ===============")
// 	logger.Error(fmt.Sprintf("  Files: %v", files))
// 	logger.Error(fmt.Sprintf("  Target ID: '%s'", targetID))
// 	logger.Error(fmt.Sprintf("  Target Classes: %v", targetClasses))
// 	logger.Error(fmt.Sprintf("  Drop X: %f, Drop Y: %f", dropX, dropY))
// 	logger.Error(
// 		fmt.Sprintf(
// 			"  Drop occurred on a designated dropzone (runtime validated before this Go event): %t",
// 			isTargetDropzone,
// 		),
// 	)
// 	logger.Error(fmt.Sprintf("  Element Attributes: %v", attributes))
// 	logger.Error("================================================================")

// 	payload := FileDropInfo{
// 		Files:         files,
// 		TargetID:      targetID,
// 		TargetClasses: targetClasses,
// 		DropX:         dropX,
// 		DropY:         dropY,
// 		Attributes:    attributes,
// 	}

// 	logger.Error("Go: Emitted 'frontend:FileDropInfo' event with payload:", "paylaod", payload)
// }
