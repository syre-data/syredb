package app

import (
	"context"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type FsService struct {
	app    *application.App
	logger *slog.Logger
}

func NewFsService(
	app *application.App,
	logger *slog.Logger,
) *FsService {
	return &FsService{app: app, logger: logger}
}

func (s *FsService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return nil
}

type CancelledByUserError struct{}

func (e *CancelledByUserError) Error() string {
	return "CANCELLED_BY_USER"
}

// TODO: Should be able to remove as it just mimics `application.FileFilter`
// but `application.FileFilter` is not exported to the frontend for some reason.
type FileFilter application.FileFilter

func (s *FsService) OpenFileDialogSingle(title string, filters []FileFilter) (string, error) {
	filters_app := make([]application.FileFilter, len(filters))
	for idx, f := range filters {
		filters_app[idx] = application.FileFilter{DisplayName: f.DisplayName, Pattern: f.Pattern}
	}

	path, err := s.app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:   title,
		Filters: filters_app,
	}).PromptForSingleSelection()

	if err != nil {
		if err.Error() == "cancelled by user" {
			return "", &CancelledByUserError{}

		} else {

			return "", err
		}
	}

	return path, nil
}
