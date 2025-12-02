package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/jackc/pgx/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const APP_NAME = "syredb"
const CONFIG_FILE_NAME = "config.toml"

type Ok struct{}

// App struct
type App struct {
	ctx     context.Context
	app_dir string
	config  AppConfigState
	db_conn DbConnectionState
}

type AppConfig struct {
	DbUrl      string
	DbUsername string
	DbPassword string
}

type AppConfigState struct {
	ok  AppConfig
	err error
}

type DbConnectionState struct {
	ok  pgx.Conn
	err error
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (app *App) startup(ctx context.Context) {
	app.ctx = ctx
	app.app_dir = getConfigDir()

	app_dir_err := os.MkdirAll(app.app_dir, PERMISSIONS_WRR)
	if app_dir_err != nil {
		runtime.LogErrorf(app.ctx, "%v", app_dir_err)
		runtime.EventsEmit(app.ctx, "config_err", app_dir_err)

		app.config.err = app_dir_err
	}

	if app_dir_err == nil {
		config_file_path := filepath.Join(app.app_dir, CONFIG_FILE_NAME)
		app.config = AppConfigState{}
		_, err := toml.DecodeFile(config_file_path, &app.config.ok)
		if err != nil {
			var parse_err toml.ParseError
			if errors.Is(err, os.ErrNotExist) {
				runtime.LogErrorf(app.ctx, "%v", err)
				runtime.EventsEmit(app.ctx, "config_err", err)

				app.config.err = err
			} else if errors.As(err, &parse_err) {
				runtime.LogErrorf(app.ctx, "%v", err)
				runtime.EventsEmit(app.ctx, "config_err", err)

				app.config.err = err
			} else {
				runtime.LogErrorf(app.ctx, "%v", err)
				runtime.EventsEmit(app.ctx, "config_err", err)

				app.config.err = err
			}
		}
	}
}

func (app *App) shutdown(ctx context.Context) {
	// TODO: Might break if not set, check err?
	app.db_conn.ok.Close(context.Background())
}

func (app *App) GetConfig() (AppConfig, error) {
	return app.config.ok, app.config.err
}

func (app *App) SaveConfig(config AppConfig) (Ok, error) {
	config_toml := new(bytes.Buffer)
	err := toml.NewEncoder(config_toml).Encode(config)
	if err != nil {
		return Ok{}, err
	}

	config_file_path := filepath.Join(app.app_dir, CONFIG_FILE_NAME)
	f, err := os.OpenFile(config_file_path, os.O_CREATE|os.O_WRONLY, PERMISSIONS_WRR)
	if err != nil {
		app.config.err = err
		runtime.LogErrorf(app.ctx, "%v", err)
		return Ok{}, err
	}
	defer f.Close()
	_, err = f.Write(config_toml.Bytes())
	if err != nil {
		app.config.err = err
		runtime.LogErrorf(app.ctx, "%v", err)
		return Ok{}, err
	}

	app.config.err = nil
	app.config.ok = config

	return Ok{}, nil
}

func (app *App) ConnectToDatabase() (Ok, error) {
	runtime.LogDebug(app.ctx, "connect")
	if app.config.err != nil {
		return Ok{}, app.config.err
	}

	// postgresql://[user[:password]@][host[:port]]/[dbname]
	connectionString := fmt.Sprintf("postgresql://%s:%s@%s", app.config.ok.DbUsername, app.config.ok.DbPassword, app.config.ok.DbUrl)
	conn, err := pgx.Connect(context.Background(), connectionString)
	if err != nil {
		runtime.LogErrorf(app.ctx, "Unable to connect to database: %v\n", err)
		return Ok{}, err
	}
	app.db_conn = DbConnectionState{ok: *conn, err: err}
	return Ok{}, nil
}
