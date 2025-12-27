package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/jackc/pgx/v5/pgxpool"

	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type InitService struct {
	ctx    context.Context
	logger *slog.Logger
}

func NewInitService(
	logger *slog.Logger,
) *InitService {
	return &InitService{logger: logger}
}

func (s *InitService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

func (s *InitService) AppConfigLoad(app_dir string) (AppConfig, error) {
	config_file_path := filepath.Join(app_dir, CONFIG_FILE_NAME)
	config := AppConfig{}
	_, err := toml.DecodeFile(config_file_path, &config)
	if err != nil {
		if err != nil {
			var parse_err toml.ParseError
			if errors.Is(err, os.ErrNotExist) {
				s.logger.With("error", err).Error("app config file not found")
				return AppConfig{}, err
			} else if errors.As(err, &parse_err) {
				s.logger.With("error", err).Error("app config file invalid format")
				return AppConfig{}, err
			} else {
				s.logger.With("error", err).Error("app config file invalid")
				return AppConfig{}, err
			}
		}
	}

	return config, nil
}

func (s *InitService) AppConfigSave(app_dir string, config AppConfig) error {
	config_toml := new(bytes.Buffer)
	err := toml.NewEncoder(config_toml).Encode(config)
	if err != nil {
		return err
	}

	config_file_path := filepath.Join(app_dir, CONFIG_FILE_NAME)
	f, err := os.OpenFile(config_file_path, os.O_CREATE|os.O_WRONLY, FILE_PERMISSIONS_WRR)
	if err != nil {
		s.logger.With("error", err).Error("could not open app config file")
		return err
	}
	defer f.Close()

	err = f.Truncate(0)
	if err != nil {
		s.logger.With("error", err).Error("could not truncate app config file")
		return err
	}

	_, err = f.Write(config_toml.Bytes())
	if err != nil {
		s.logger.With("error", err).Error("could not write app config file")
		return err
	}

	return nil
}

func (s *InitService) ConnectToDatabase(url string, db_name string, username string, password string) (*pgxpool.Pool, error) {
	if len(username) == 0 {
		return nil, errors.New("invalid username")
	}
	if len(url) == 0 {
		return nil, errors.New("invalid url")
	}
	if len(db_name) == 0 {
		return nil, errors.New("invalid database name")
	}

	// postgresql://[user[:password]@][host[:port]]/[dbname]
	connectionString := fmt.Sprintf("postgresql://%s:%s@%s/%s", username, password, url, db_name)
	dbpool, err := pgxpool.New(context.Background(), connectionString)
	if err != nil {
		return nil, err
	}

	err = dbpool.Ping(s.ctx)
	if err != nil {
		return nil, err
	}

	return dbpool, nil

}
