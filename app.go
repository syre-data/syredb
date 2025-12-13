package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"crypto/rand"
	"crypto/subtle"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"golang.org/x/crypto/argon2"
)

const APP_NAME = "syredb"
const CONFIG_FILE_NAME = "config.toml"
const USER_AUTH_FILE_NAME = "user_auth.toml"

type Result[T any] struct {
	ok  T
	err error
}

type Ok struct{}

// App struct
type App struct {
	ctx     context.Context
	app_dir string
	config  AppConfigState
	db_pool DbConnectionState
}

type AppConfigState = Result[AppConfig]
type DbConnectionState = Result[*pgxpool.Pool]

type AppConfig struct {
	DbUrl      string
	DbUsername string
	DbPassword string
	DbName     string
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

		app.config.err = app_dir_err
	}

	if app_dir_err == nil {
		app.LoadAppConfig()
	}
}

func (app *App) shutdown(ctx context.Context) {
	// TODO: Might break if not set, check err?
	app.db_pool.ok.Close()
}

func (app *App) LoadAppConfig() (Ok, error) {
	app.config = AppConfigState{}
	config_file_path := filepath.Join(app.app_dir, CONFIG_FILE_NAME)
	_, err := toml.DecodeFile(config_file_path, &app.config.ok)
	if err != nil {
		var parse_err toml.ParseError
		if errors.Is(err, os.ErrNotExist) {
			runtime.LogErrorf(app.ctx, "app config file not found: %v", err)
			app.config.err = err
		} else if errors.As(err, &parse_err) {
			runtime.LogErrorf(app.ctx, "app config file invalid format: %v", err)
			app.config.err = err
		} else {
			runtime.LogErrorf(app.ctx, "app config file invalid: %v", err)
			app.config.err = err
		}
	}

	return Ok{}, app.config.err
}

func (app *App) GetAppConfig() (AppConfig, error) {
	err := app.config.err
	if errors.Is(err, os.ErrNotExist) {
		err = errors.New("FILE_NOT_FOUND")
	}
	return app.config.ok, err
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

	err = f.Truncate(0)
	if err != nil {
		app.config.err = err
		runtime.LogErrorf(app.ctx, "%v", err)
		return Ok{}, err
	}

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
	if app.db_pool.ok != nil {
		return Ok{}, nil
	}
	if app.config.err != nil {
		err := app.config.err
		if errors.Is(err, os.ErrNotExist) {
			err = errors.New("FILE_NOT_FOUND")
		}

		return Ok{}, err
	}

	config := app.config.ok
	if len(config.DbUsername) == 0 {
		return Ok{}, errors.New("invalid username")
	}
	if len(config.DbUrl) == 0 {
		return Ok{}, errors.New("invalid url")
	}
	if len(config.DbName) == 0 {
		return Ok{}, errors.New("invalid database name")
	}

	// postgresql://[user[:password]@][host[:port]]/[dbname]
	connectionString := fmt.Sprintf("postgresql://%s:%s@%s/%s", config.DbUsername, config.DbPassword, config.DbUrl, config.DbName)
	dbpool, err := pgxpool.New(context.Background(), connectionString)
	if err != nil {
		runtime.LogErrorf(app.ctx, "Unable to connect to database: %v\n", err)
		return Ok{}, err
	}

	err = dbpool.Ping(context.Background())
	if err != nil {
		return Ok{}, err
	}

	app.db_pool = Result[*pgxpool.Pool]{ok: dbpool, err: nil}
	return Ok{}, nil

}

type User struct {
	Id              uuid.UUID
	Email           string
	Name            string
	PermissionRoles []string `db:"permission_roles"`
}

type UserAuth struct {
	UserId    string
	AuthToken string
}

func (app *App) LoadUser() (User, error) {
	if app.db_pool.err != nil {
		return User{}, app.db_pool.err
	}

	var user_auth UserAuth
	auth_file_path := filepath.Join(app.app_dir, USER_AUTH_FILE_NAME)
	_, err := toml.DecodeFile(auth_file_path, &user_auth)
	if err != nil {
		var parse_err toml.ParseError
		if errors.Is(err, os.ErrNotExist) {
			runtime.LogErrorf(app.ctx, "user auth file not found: %v", err)
			return User{}, nil
		} else if errors.As(err, &parse_err) {
			runtime.LogErrorf(app.ctx, "user auth file invalid format: %v", err)
			return User{}, err
		} else {
			runtime.LogErrorf(app.ctx, "user auth file invalid: %v", err)
			return User{}, err
		}
	}

	auth_rows, _ := app.db_pool.ok.Query(context.Background(), "SELECT tokens FROM user_auth_ WHERE _id=$1", user_auth.UserId)
	defer auth_rows.Close()
	var db_auth_tokens []string
	for auth_rows.Next() {
		err = auth_rows.Scan(&db_auth_tokens)
		if err != nil {
			runtime.LogErrorf(app.ctx, "%v", err)
			return User{}, err
		}
	}

	if !slices.Contains(db_auth_tokens, user_auth.AuthToken) {
		return User{}, nil
	}

	user_rows, _ := app.db_pool.ok.Query(context.Background(), "SELECT _id, email, name, permission_roles FROM user_ WHERE _id=$1", user_auth.UserId)
	defer user_rows.Close()
	var user User
	for user_rows.Next() {
		err = user_rows.Scan(&user.Id, &user.Email, &user.Name, &user.PermissionRoles)
		if err != nil {
			runtime.LogErrorf(app.ctx, "%v", err)
			return User{}, err
		}
	}
	return user, nil
}

type UserCredentials struct {
	Email    string
	Password string
}

func (app *App) AuthenticateAndGetUser(credentials UserCredentials, remember bool) (User, error) {
	if app.db_pool.err != nil {
		return User{}, app.db_pool.err
	}

	user_rows, _ := app.db_pool.ok.Query(context.Background(), "SELECT _id, email, name, permission_roles FROM user_ WHERE email=$1", credentials.Email)
	defer user_rows.Close()
	var user_id uuid.UUID
	var email string
	var name string
	var permission_roles []string
	for user_rows.Next() {
		err := user_rows.Scan(&user_id, &email, &name, &permission_roles)
		if err != nil {
			runtime.LogErrorf(app.ctx, "%v", err)
			return User{}, err
		}
	}

	auth_rows, _ := app.db_pool.ok.Query(context.Background(), "SELECT auth FROM user_auth_ WHERE _id=$1", user_id.String())
	defer auth_rows.Close()
	var auth_hash string
	for auth_rows.Next() {
		err := auth_rows.Scan(&auth_hash)
		if err != nil {
			runtime.LogErrorf(app.ctx, "%v", err)
			return User{}, err
		}
	}

	authorized, err := comparePasswordAndHash(credentials.Password, auth_hash)
	if err != nil {
		runtime.LogErrorf(app.ctx, "could not compare password to hash: %v", err)
		return User{}, errors.New("server error: invalid password comparison")
	}

	if !authorized {
		return User{}, nil
	}

	user := User{
		Id:              user_id,
		Email:           email,
		Name:            name,
		PermissionRoles: permission_roles,
	}

	if remember {
		func() {
			auth_token_b := make([]byte, PASSWORD_HASH_SALT_LENGTH_BYTES)
			rand.Read(auth_token_b)
			auth_token := base64.RawStdEncoding.EncodeToString(auth_token_b)

			append_token_query := "UPDATE user_auth_ SET tokens=ARRAY_APPEND(tokens, $1) WHERE _id=$2"
			_, err := app.db_pool.ok.Exec(context.Background(), append_token_query, auth_token, user_id)
			if err != nil {
				runtime.LogErrorf(app.ctx, "could not insert user auth token: %v", err)
				return
			}

			user_auth := UserAuth{UserId: user_id.String(), AuthToken: auth_token}
			user_auth_toml := new(bytes.Buffer)
			err = toml.NewEncoder(user_auth_toml).Encode(user_auth)
			if err != nil {
				runtime.LogErrorf(app.ctx, "could not save user auth token: %v", err)
				return
			}

			user_auth_file_path := filepath.Join(app.app_dir, USER_AUTH_FILE_NAME)
			f, err := os.OpenFile(user_auth_file_path, os.O_CREATE|os.O_WRONLY, PERMISSIONS_WRR)
			if err != nil {
				runtime.LogErrorf(app.ctx, "could not save user auth token: %v", err)
				return
			}
			defer f.Close()
			_, err = f.Write(user_auth_toml.Bytes())
			if err != nil {
				runtime.LogErrorf(app.ctx, "could not save user auth token: %v", err)
				return
			}
		}()
	} else {
		user_auth_file_path := filepath.Join(app.app_dir, USER_AUTH_FILE_NAME)
		err = os.Remove(user_auth_file_path)
		if err != nil {
			runtime.LogErrorf(app.ctx, "could not remove user auth file: %v", err)
		}
	}

	return user, nil
}

const PASSWORD_HASH_ALGO_ID = "argon2id"
const PASSWORD_HASH_HASH_LENGTH_BYTES = 512
const PASSWORD_HASH_SALT_LENGTH_BYTES = 64
const PASSWORD_HASH_ITERATIONS = 2
const PASSWORD_HASH_MEMORY = 64 * 1024
const PASSWORD_HASH_PARALLELISM = 4

func encodePassword(password string) string {
	salt := make([]byte, PASSWORD_HASH_SALT_LENGTH_BYTES)
	rand.Read(salt)
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		PASSWORD_HASH_ITERATIONS,
		PASSWORD_HASH_MEMORY,
		PASSWORD_HASH_PARALLELISM,
		PASSWORD_HASH_HASH_LENGTH_BYTES,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		PASSWORD_HASH_ALGO_ID,
		argon2.Version,
		PASSWORD_HASH_MEMORY,
		PASSWORD_HASH_ITERATIONS,
		PASSWORD_HASH_PARALLELISM,
		b64Salt,
		b64Hash,
	)
}

type passwordHashParameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

func decodePasswordHash(encoded_hash string) (p *passwordHashParameters, salt, hash []byte, err error) {
	vals := strings.Split(encoded_hash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, fmt.Errorf("invalid hash format")
	}

	if vals[1] != PASSWORD_HASH_ALGO_ID {
		return nil, nil, nil, errors.New("incompatible hash alogrithm")
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, errors.New("incompatible version of argon2")
	}

	p = &passwordHashParameters{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	p.saltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	p.keyLength = uint32(len(hash))

	return p, salt, hash, nil
}

func comparePasswordAndHash(clear_text_password string, encoded_hash string) (match bool, err error) {
	p, salt, stored_hash, err := decodePasswordHash(encoded_hash)
	if err != nil {
		return false, err
	}

	password_hash := argon2.IDKey(
		[]byte(clear_text_password),
		salt,
		p.iterations,
		p.memory,
		p.parallelism,
		p.keyLength,
	)

	// SAFETY: `subtle.ConstantTimeCompare()` prevents timing attacks.
	if subtle.ConstantTimeCompare(stored_hash, password_hash) == 1 {
		return true, nil
	}
	return false, nil
}

type UserCreate struct {
	Email           string
	Name            string
	Password        string
	PermissionRoles []string
}

func (app *App) CreateUser(user UserCreate) (uuid.UUID, error) {

	if app.db_pool.err != nil {
		return uuid.Nil, app.db_pool.err
	}

	tx, err := app.db_pool.ok.Begin(context.Background())
	if err != nil {
		runtime.LogErrorf(app.ctx, "Unable to begin transaction: %v\n", err)
		return uuid.Nil, err
	}
	defer tx.Rollback(context.Background())

	var user_id uuid.UUID
	insert_user_query := "INSERT INTO user_ (email, name) VALUES ($1, $2, $3) RETURNING _id"
	err = tx.QueryRow(context.Background(), insert_user_query, user.Email, user.Name).Scan(&user_id)
	if err != nil {
		runtime.LogErrorf(app.ctx, "Error inserting user: %v\n", err)
		return uuid.Nil, err
	}

	insert_user_auth_query := "INSERT INTO user_auth_ (_id, auth) VALUES ($1, $2)"
	_, err = tx.Exec(context.Background(), insert_user_auth_query, user_id, encodePassword(user.Password))
	if err != nil {
		runtime.LogErrorf(app.ctx, "Error inserting user authentication data: %v\n", err)
		return uuid.Nil, err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		runtime.LogErrorf(app.ctx, "Error committing create user transaction: %v\n", err)
		return uuid.Nil, err
	}

	return user_id, nil
}

type Sample struct {
	Id    uuid.UUID
	Owner uuid.UUID
	Label string
}

type SampleGroup struct {
	Id          uuid.UUID
	Owner       uuid.UUID
	Label       string
	Description string
}

type SampleGroupsWithSamples = []Sample

func (app *App) GetRecentSampleGroupsWithSamples() (SampleGroupsWithSamples, error) {
	if app.db_pool.err != nil {
		runtime.LogDebugf(app.ctx, "%#v", app.db_pool.err)
		return SampleGroupsWithSamples{}, app.db_pool.err
	}

	rows, err := app.db_pool.ok.Query(context.Background(), "SELECT * from sample_")
	if err != nil {
		runtime.LogDebugf(app.ctx, "%#v", err)
		return SampleGroupsWithSamples{}, err
	}
	runtime.LogDebugf(app.ctx, "%#v", rows)
	// sample_groups, err := pgx.CollectRows(rows, pgx.RowTo[Sample])
	sample_groups, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Sample, error) {
		var id uuid.UUID
		var owner uuid.UUID
		var label string
		err := row.Scan(&id, &owner, &label)
		return Sample{id, owner, label}, err
	})
	if err != nil {
		runtime.LogDebugf(app.ctx, "%#v", err)
		return SampleGroupsWithSamples{}, err
	}

	return sample_groups, nil
}

type SampleCreate struct {
	Label string
}
type SampleGroupCreate struct {
	Label       string
	Description string
	Parents     []uuid.UUID
	Samples     []SampleCreate
}

func (app *App) CreateSampleGroup(group SampleGroupCreate) (Ok, error) {
	if app.db_pool.err != nil {
		runtime.LogDebugf(app.ctx, "%#v", app.db_pool.err)
		return Ok{}, app.db_pool.err
	}

	// TODO
	// sample_group_query := "INSERT INTO sample_group (_owner, label, description) VALUES ($1, $2, $3)"
	// batch.Queue(sample_group_query, group.Label, group.Description).QueryRow(func(row pgx.Row) error {
	// 	err := row.Scan()
	// 	return err
	// })

	// sample_group_result, err := app.db_conn.ok.Query(context.Background(), sample_group_query)
	// if err != nil {
	// 	runtime.LogDebugf(app.ctx, "%#v", app.db_conn.err)
	// 	return Ok{}, err
	// }

	return Ok{}, nil
}
