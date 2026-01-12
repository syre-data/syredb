package app

import (
	"context"
	"log/slog"
	"os"

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
			s.logger.With("error", err).Error("could not get path")
			return "", err
		}
	}

	return path, nil
}

// SaveFileSingle prompts a user for a save path for the given data.
// If a file already exists at the chosen path, they are prompted to confirm they wish to overwrite the existing data.
// Returns the path the user selected.
func (s *FsService) SaveFileSingle(data []byte, title string, filters []FileFilter) (string, error) {
	filters_app := make([]application.FileFilter, len(filters))
	for idx, f := range filters {
		filters_app[idx] = application.FileFilter{DisplayName: f.DisplayName, Pattern: f.Pattern}
	}

	path, err := s.app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:   title,
		Filters: filters_app,
	}).PromptForSingleSelection()

	if err != nil {
		if err.Error() == "cancelled by user" {
			return "", &CancelledByUserError{}
		} else {
			s.logger.With("error", err).Error("could not get save path")
			return "", err
		}
	}

	err = s.save_file(path, data)
	if err != nil {
		s.logger.With(
			"error", err,
			"file path", path,
		).Error("could not save data")
	}

	return path, nil

	// NB: Doesn't seem to be needed, at least on Windows.
	// OS takes care of confirmation.
	// _, err = os.Stat(path)
	// if err == nil {
	// 	dialog := s.app.Dialog.Question().
	// 		SetTitle("Confirm Overwrite").
	// 		SetMessage("File already exists. Overwrite?")

	// 	overwriteBtn := dialog.AddButton("Overwrite")
	// 	overwriteBtn.OnClick(func() {
	// 		err = s.save_file(path, data)
	// 		if err != nil {
	// 			s.logger.With(
	// 				"error", err,
	// 				"file path", path,
	// 			).Error("could not save data")
	// 		}
	// 	})

	// 	cancelBtn := dialog.AddButton("Cancel")
	// 	dialog.SetDefaultButton(cancelBtn)
	// 	dialog.SetCancelButton(cancelBtn)
	// 	dialog.Show()

	// 	return path, nil
	// } else if errors.Is(err, os.ErrNotExist) {
	// 	err = s.save_file(path, data)
	// 	if err != nil {
	// 		s.logger.With(
	// 			"error", err,
	// 			"file path", path,
	// 		).Error("could not save data")
	// 	}

	// 	return path, nil
	// } else {
	// 	s.logger.With("error", err).Error("could not verify save path")
	// 	return path, err
	// }
}

func (s *FsService) save_file(path string, data []byte) error {
	return os.WriteFile(path, data, FILE_PERMISSIONS_WRR)
}
