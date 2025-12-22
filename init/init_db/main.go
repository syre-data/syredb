package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"net/mail"
	"os"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"

	"golang.org/x/crypto/argon2"
)

const PASSWORD_HASH_ALGO_ID = "argon2id"
const PASSWORD_HASH_HASH_LENGTH_BYTES = 512
const PASSWORD_HASH_SALT_LENGTH_BYTES = 64
const PASSWORD_HASH_ITERATIONS = 2
const PASSWORD_HASH_MEMORY = 64 * 1024
const PASSWORD_HASH_PARALLELISM = 4

const APP_EMAIL_URL_KEY = "app:email:url"
const APP_EMAIL_USERNAME_KEY = "app:email:username"
const APP_EMAIL_PASSWORD_KEY = "app:email:password"
const APP_EMAIL_FROM_KEY = "app:email:from"

func main() {
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
	insert_user_query := "INSERT INTO user_ (email, name, role) VALUES ($1, $2, 'owner') RETURNING _id"
	err = tx.QueryRow(context.Background(), insert_user_query, email, name).Scan(&user_id)
	if err != nil {
		return err
	}

	insert_user_auth_query := "INSERT INTO user_auth_ (_id, auth) VALUES ($1, $2)"
	_, err = tx.Exec(context.Background(), insert_user_auth_query, user_id, encode_password(password))
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

func set_app_email(conn *pgx.Conn, url string, username string, password string, from string) error {
	insert_user_query := "INSERT INTO _app_data_ (key, value) VALUES ($1, $2), ($3, $4), ($5, $6), ($7, $8)"
	_, err := conn.Exec(
		context.Background(),
		insert_user_query,
		APP_EMAIL_URL_KEY,
		url,
		APP_EMAIL_USERNAME_KEY,
		username,
		APP_EMAIL_PASSWORD_KEY,
		password,
		APP_EMAIL_FROM_KEY,
		from,
	)
	return err
}
