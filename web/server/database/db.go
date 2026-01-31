package database

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const DB_USERNAME_KEY = "SYREDB_PG_USERNAME"
const DB_PASSWORD_KEY = "SYREDB_PG_PASSWORD"
const DB_URL_KEY = "SYREDB_PG_URL"
const DB_NAME_KEY = "SYREDB_PG_DBNAME"
const PG_DEFAULT_URL = "localhost:5432"
const PG_DEFAULT_DBNAME = "syredb"

const MAXIMUM_CONNECTIONS = 10
const MINIMUM_CONNECTIONS = 2

var Conn *pgxpool.Pool

type DbCredentials struct {
	username string
	password string
	url      string
	db_name  string
}

func collect_db_credentials() (DbCredentials, error) {
	db_username_flag := flag.String("pg-username", "", "postgres username")
	db_password_flag := flag.String("pg-password", "", "postgres password")
	db_url_flag := flag.String("pg-url", "", "postgres url")
	db_name_flag := flag.String("pg-dbname", "", "postgres db name")
	flag.Parse()

	credentials := DbCredentials{
		username: *db_username_flag,
		password: *db_password_flag,
		url:      *db_url_flag,
		db_name:  *db_name_flag,
	}
	if credentials.username == "" {
		credentials.username = os.Getenv(DB_USERNAME_KEY)
	}
	if credentials.password == "" {
		credentials.password = os.Getenv(DB_PASSWORD_KEY)
	}
	if credentials.url == "" {
		credentials.url = os.Getenv(DB_URL_KEY)
	}
	if credentials.db_name == "" {
		credentials.db_name = os.Getenv(DB_NAME_KEY)
	}

	if credentials.url == "" {
		credentials.url = PG_DEFAULT_URL
	}
	if credentials.db_name == "" {
		credentials.db_name = PG_DEFAULT_DBNAME
	}
	if credentials.username == "" || credentials.password == "" {
		return DbCredentials{}, errors.New("invalid database credentials")
	}

	return credentials, nil
}

func Connect() error {
	credentials, err := collect_db_credentials()
	if err != nil {
		return fmt.Errorf("could not obtain database credentials")
	}

	connection_string := fmt.Sprintf(
		"postgresql://%s:%s@%s/%s",
		credentials.username,
		credentials.password,
		credentials.url,
		credentials.db_name,
	)

	config, err := pgxpool.ParseConfig(connection_string)
	if err != nil {
		return fmt.Errorf("unable to parse database URL: %w", err)
	}

	config.MaxConns = MAXIMUM_CONNECTIONS
	config.MinIdleConns = MINIMUM_CONNECTIONS
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	Conn = pool
	fmt.Println("Database connected successfully")
	return nil
}

func Close() {
	if Conn != nil {
		Conn.Close()
	}
}
