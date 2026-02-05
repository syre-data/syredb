package database

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ENV_DB_USERNAME_KEY = "SYREDB_DB_USERNAME"
const ENV_DB_PASSWORD_KEY = "SYREDB_DB_PASSWORD"
const ENV_DB_HOST_KEY = "SYREDB_DB_HOST"
const ENV_DB_NAME_KEY = "SYREDB_DB_NAME"
const FLAG_DB_USERNAME_KEY = "db-username"
const FLAG_DB_PASSWORD_KEY = "db-password"
const FLAG_DB_HOST_KEY = "db-host"
const FLAG_DB_NAME_KEY = "db-name"
const DB_DEFAULT_HOST = "localhost:5432"
const DB_DEFAULT_NAME = "syredb"

const MAXIMUM_CONNECTIONS = 10
const MINIMUM_CONNECTIONS = 2

type DbConnection struct {
	Conn *pgxpool.Pool
}

func (c *DbConnection) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}

type DbCredentials struct {
	username string
	password string
	host     string
	db_name  string
}

func CollectCredentialsFromEnvAndFlags() (DbCredentials, error) {
	db_username_flag := flag.String(FLAG_DB_USERNAME_KEY, "", "database username")
	db_password_flag := flag.String(FLAG_DB_PASSWORD_KEY, "", "database password")
	db_url_flag := flag.String(FLAG_DB_HOST_KEY, "", "database host with post")
	db_name_flag := flag.String(FLAG_DB_NAME_KEY, "", "database name")
	flag.Parse()

	credentials := DbCredentials{
		username: *db_username_flag,
		password: *db_password_flag,
		host:     *db_url_flag,
		db_name:  *db_name_flag,
	}
	if credentials.username == "" {
		credentials.username = os.Getenv(ENV_DB_USERNAME_KEY)
	}
	if credentials.password == "" {
		credentials.password = os.Getenv(ENV_DB_PASSWORD_KEY)
	}
	if credentials.host == "" {
		credentials.host = os.Getenv(ENV_DB_HOST_KEY)
	}
	if credentials.db_name == "" {
		credentials.db_name = os.Getenv(ENV_DB_NAME_KEY)
	}

	if credentials.host == "" {
		credentials.host = DB_DEFAULT_HOST
	}
	if credentials.db_name == "" {
		credentials.db_name = DB_DEFAULT_NAME
	}
	if credentials.username == "" || credentials.password == "" {
		return DbCredentials{}, errors.New("invalid database credentials")
	}

	return credentials, nil
}

func Connect(credentials DbCredentials) (*DbConnection, error) {
	connection_string := fmt.Sprintf(
		"postgresql://%s:%s@%s/%s",
		credentials.username,
		credentials.password,
		credentials.host,
		credentials.db_name,
	)

	config, err := pgxpool.ParseConfig(connection_string)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	config.MaxConns = MAXIMUM_CONNECTIONS
	config.MinIdleConns = MINIMUM_CONNECTIONS
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &DbConnection{Conn: pool}, nil
}
