package app

import (
	"errors"
	"os"

	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/jackc/pgx/v5/pgxpool"

	"log/slog"
)

func (s *AppService) AppConfigLoad() (Ok, error) {
	s.config = AppConfigState{}
	config, err := app_config_load(s.app_dir, s.logger)
	if err != nil {
		s.config.err = err
	} else {
		s.config.ok = config
	}

	return Ok{}, s.config.err
}

func app_config_load(app_dir string, logger *slog.Logger) (AppConfig, error) {
	config_file_path := filepath.Join(app_dir, CONFIG_FILE_NAME)
	config := AppConfig{}
	_, err := toml.DecodeFile(config_file_path, &config)
	if err != nil {
		var parse_err toml.ParseError
		if errors.Is(err, os.ErrNotExist) {
			logger.With("error", err).Error("app config file not found")
			return AppConfig{}, err
		} else if errors.As(err, &parse_err) {
			logger.With("error", err).Error("app config file invalid format")
			return AppConfig{}, err
		} else {
			logger.With("error", err).Error("app config file invalid")
			return AppConfig{}, err
		}
	}

	return config, nil
}

func (s *AppService) AppConfigGet() (AppConfig, error) {
	err := s.config.err
	if errors.Is(err, os.ErrNotExist) {
		err = errors.New("FILE_NOT_FOUND")
	}
	return s.config.ok, err
}

func app_config_save(app_dir string, config AppConfig, logger *slog.Logger) error {
	config_toml := new(bytes.Buffer)
	err := toml.NewEncoder(config_toml).Encode(config)
	if err != nil {
		return err
	}

	config_file_path := filepath.Join(app_dir, CONFIG_FILE_NAME)
	f, err := os.OpenFile(config_file_path, os.O_CREATE|os.O_WRONLY, FILE_PERMISSIONS_WRR)
	if err != nil {
		logger.With("error", err).Error("could not open app config file")
		return err
	}
	defer f.Close()

	err = f.Truncate(0)
	if err != nil {
		logger.With("error", err).Error("could not truncate app config file")
		return err
	}

	_, err = f.Write(config_toml.Bytes())
	if err != nil {
		logger.With("error", err).Error("could not write app config file")
		return err
	}

	return nil
}

func (s *AppService) AppConfigSave(config AppConfig) (Ok, error) {
	err := app_config_save(s.app_dir, config, s.logger)
	if err != nil {
		var parse_err toml.ParseError
		if errors.As(err, &parse_err) {
			return Ok{}, err
		} else {
			s.config.err = err
			return Ok{}, err
		}
	}

	s.config.err = nil
	s.config.ok = config
	return Ok{}, nil
}

func (s *AppService) DatabaseConnect() (Ok, error) {
	if s.db.ok.conn != nil {
		return Ok{}, nil
	}
	if s.config.err != nil {
		err := s.config.err
		if errors.Is(err, os.ErrNotExist) {
			err = errors.New("FILE_NOT_FOUND")
		}

		return Ok{}, err
	}

	db, err := database_connect(
		s.ctx,
		s.config.ok.DbUrl,
		s.config.ok.DbName,
		s.config.ok.DbUsername,
		s.config.ok.DbPassword,
	)
	if err != nil {
		s.logger.With("error", err).Error("could not connect to database")
		s.db.err = err
		return Ok{}, err
	}

	s.db.ok.conn = db
	s.db.err = nil
	return Ok{}, nil

}

func database_connect(ctx context.Context, url string, db_name string, username string, password string) (*pgxpool.Pool, error) {
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
	db, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, err
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
