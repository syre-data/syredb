package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"

	"golang.org/x/crypto/argon2"
)

const PasswordHashAlgoId = "argon2id"
const PasswordHashHashLengthBytes = 512
const PasswordHashSaltLengthBytes = 64
const PasswordHashIterations = 2
const PasswordHashMemory = 64 * 1024
const PasswordHashParallelism = 4

const AppEmailUrlKey = "app:email:url"
const AppEmailUsernameKey = "app:email:username"
const AppEmailPasswordKey = "app:email:password"
const AppEmailFromKey = "app:email:from"
const AppAccountNameKey = "app:account:name"
const AppAccountLogoKey = "app:account:logo"
const AppDataPathKey = "app:data:path"

const DataPathDefaultRel = "syredb"

func main() {
	home_dir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(fmt.Errorf("can not get user's home directory: #%v", err))
		os.Exit(1)
	}

	cmd := flag.String("cmd", "", "sub command")
	pg_user := flag.String("pg-user", "", "postgres username")
	pg_password := flag.String("pg-password", "", "postgres password")
	pg_url := flag.String("pg-url", "localhost:5432", "postgres database url")
	pg_db := flag.String("pg-db", "syredb", "postgres database name")

	// db flags
	db_owner_user_email := flag.String("db-owner-email", "", "user's email")
	db_owner_user_name := flag.String("db-owner-name", "", "user's name")
	db_owner_user_password := flag.String("db-owner-password", "", "user's password")

	// app email flags
	app_email_url := flag.String("app-email-url", "", "app email url")
	app_email_username := flag.String("app-email-username", "", "app email username")
	app_email_password := flag.String("app-email-password", "", "app email password")
	app_email_from_address := flag.String("app-email-from-address", "", "app email from address")

	// account info flags
	account_name := flag.String("account-name", "", "account name")
	account_logo := flag.String("account-logo", "", "path to the account logo")

	// data flags
	data_path_default := filepath.Join(home_dir, DataPathDefaultRel)
	data_path := flag.String("data-path", data_path_default, "data path")

	flag.Parse()

	connectionString := fmt.Sprintf("postgresql://%s:%s@%s/%s", *pg_user, *pg_password, *pg_url, *pg_db)
	conn, err := pgx.Connect(context.Background(), connectionString)
	if err != nil {
		os.Exit(2)
	}
	defer conn.Close(context.Background())

	switch *cmd {
	case "db-owner":
		_, err = mail.ParseAddress(*db_owner_user_email)
		if err != nil {
			os.Exit(10)
		}

		err = create_db_owner_user(conn, *db_owner_user_email, *db_owner_user_name, *db_owner_user_password)
		if err != nil {
			fmt.Println(err)
			os.Exit(11)
		}

	case "app-email":
		_, err = mail.ParseAddress(*app_email_from_address)
		if err != nil {
			os.Exit(20)
		}

		err = set_app_email(conn, *app_email_url, *app_email_username, *app_email_password, *app_email_from_address)
		if err != nil {
			fmt.Println(err)
			os.Exit(21)
		}

	case "account-info":
		err = set_account_info(conn, *account_name, *account_logo)
		if err != nil {
			fmt.Println(err)
			os.Exit(30)
		}

	case "data":
		err = set_data(conn, *data_path)
		if err != nil {
			fmt.Println(err)
			os.Exit(40)
		}

	default:
	}

}

func create_db_owner_user(conn *pgx.Conn, email string, name string, password string) error {
	tx, err := conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	var user_id uuid.UUID
	insert_user_query := "INSERT INTO user_ (email, name) VALUES ($1, $2) RETURNING _id"
	err = tx.QueryRow(context.Background(), insert_user_query, email, name).Scan(&user_id)
	if err != nil {
		return err
	}

	insert_user_auth_query := "INSERT INTO user_auth_ (_id, auth) VALUES ($1, $2)"
	_, err = tx.Exec(context.Background(), insert_user_auth_query, user_id, encode_password(password))
	if err != nil {
		return err
	}

	insert_user_permission_query := "INSERT INTO db_user_permission_ (_user, _permission) VALUES ($1, $2)"
	_, err = tx.Exec(context.Background(), insert_user_permission_query, user_id, "owner")
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func encode_password(password string) string {
	salt := make([]byte, PasswordHashSaltLengthBytes)
	rand.Read(salt)
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		PasswordHashIterations,
		PasswordHashMemory,
		PasswordHashParallelism,
		PasswordHashHashLengthBytes,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		PasswordHashAlgoId,
		argon2.Version,
		PasswordHashMemory,
		PasswordHashIterations,
		PasswordHashParallelism,
		b64Salt,
		b64Hash,
	)
}

func set_app_email(conn *pgx.Conn, url string, username string, password string, from string) error {
	insert_user_query := "INSERT INTO _app_data_ (key, value) VALUES ($1, $2), ($3, $4), ($5, $6), ($7, $8)"
	_, err := conn.Exec(
		context.Background(),
		insert_user_query,
		AppEmailUrlKey,
		url,
		AppEmailUsernameKey,
		username,
		AppEmailPasswordKey,
		password,
		AppEmailFromKey,
		from,
	)
	return err
}

func set_account_info(conn *pgx.Conn, name string, logo_path string) error {
	inser_account_info_query := "INSERT INTO _app_data_ (key, value) VALUES ($1, $2), ($3, $4)"
	_, err := conn.Exec(
		context.Background(),
		inser_account_info_query,
		AppAccountNameKey,
		name,
		AppAccountLogoKey,
		logo_path,
	)
	return err
}

func set_data(conn *pgx.Conn, data_path string) error {
	inser_data_query := "INSERT INTO _app_data_ (key, value) VALUES ($1, $2)"
	_, err := conn.Exec(
		context.Background(),
		inser_data_query,
		AppDataPathKey,
		data_path,
	)
	return err
}
