package database

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const EnvDBUsernameKey = "SYREDB_DB_USERNAME"
const EnvDBPasswordKey = "SYREDB_DB_PASSWORD"
const EnvDBHostKey = "SYREDB_DB_HOST"
const EnvDBNameKey = "SYREDB_DB_NAME"
const FlagDBUsernameKey = "db-username"
const FlagDBPasswordKey = "db-password"
const FlagDBHostKey = "db-host"
const FlagDBNameKey = "db-name"
const DBDefaultHost = "localhost:5432"
const DBDefaultName = "syredb"

const MaximumConnections = 10
const MinimumConnections = 2

type DBConnection struct {
	Conn *pgxpool.Pool
}

func (c *DBConnection) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}

type DBCredentials struct {
	username string
	password string
	host     string
	db_name  string
}

func CollectCredentialsFromEnvAndFlags() (DBCredentials, error) {
	db_username_flag := flag.String(FlagDBUsernameKey, "", "database username")
	db_password_flag := flag.String(FlagDBPasswordKey, "", "database password")
	db_url_flag := flag.String(FlagDBHostKey, "", "database host with post")
	db_name_flag := flag.String(FlagDBNameKey, "", "database name")
	flag.Parse()

	credentials := DBCredentials{
		username: *db_username_flag,
		password: *db_password_flag,
		host:     *db_url_flag,
		db_name:  *db_name_flag,
	}
	if credentials.username == "" {
		credentials.username = os.Getenv(EnvDBUsernameKey)
	}
	if credentials.password == "" {
		credentials.password = os.Getenv(EnvDBPasswordKey)
	}
	if credentials.host == "" {
		credentials.host = os.Getenv(EnvDBHostKey)
	}
	if credentials.db_name == "" {
		credentials.db_name = os.Getenv(EnvDBNameKey)
	}

	if credentials.host == "" {
		credentials.host = DBDefaultHost
	}
	if credentials.db_name == "" {
		credentials.db_name = DBDefaultName
	}
	if credentials.username == "" || credentials.password == "" {
		return DBCredentials{}, errors.New("invalid database credentials")
	}

	return credentials, nil
}

func Connect(credentials DBCredentials) (*DBConnection, error) {
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

	config.MaxConns = MaximumConnections
	config.MinIdleConns = MinimumConnections
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &DBConnection{Conn: pool}, nil
}
