package app

import (
	"errors"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (s *AppService) AppConfigLoad() (Ok, error) {
	s.config = AppConfigState{}
	config, err := s.init_service.AppConfigLoad(s.app_dir)
	if err != nil {
		s.config.err = err
	} else {
		s.config.ok = config
	}

	return Ok{}, s.config.err
}

func (s *AppService) GetAppConfig() (AppConfig, error) {
	err := s.config.err
	if errors.Is(err, os.ErrNotExist) {
		err = errors.New("FILE_NOT_FOUND")
	}
	return s.config.ok, err
}

func (s *AppService) AppConfigSave(config AppConfig) (Ok, error) {
	err := s.init_service.AppConfigSave(s.app_dir, config)
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

func (s *AppService) ConnectToDatabase() (Ok, error) {
	if s.db.ok != nil {
		return Ok{}, nil
	}
	if s.config.err != nil {
		err := s.config.err
		if errors.Is(err, os.ErrNotExist) {
			err = errors.New("FILE_NOT_FOUND")
		}

		return Ok{}, err
	}

	db, err := s.init_service.ConnectToDatabase(
		s.config.ok.DbUrl,
		s.config.ok.DbName,
		s.config.ok.DbUsername,
		s.config.ok.DbPassword,
	)
	if err != nil {
		s.logger.With("error", err).Error("could not connect to database")
		return Ok{}, err
	}

	s.db = Result[*pgxpool.Pool]{ok: db, err: nil}
	return Ok{}, nil

}

type User struct {
	Id            uuid.UUID
	AccountStatus string
	Email         string
	Name          string
	Role          UserRole
}
