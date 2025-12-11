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

func main() {
	pg_user := flag.String("pg-user", "", "postgres username")
	pg_password := flag.String("pg-password", "", "postgres password")
	pg_url := flag.String("pg-url", "localhost:5432", "postgres database url")
	pg_db := flag.String("pg-db", "syredb", "postgres database name")
	user_email := flag.String("email", "", "user's email")
	user_name := flag.String("name", "", "user's name")
	user_password := flag.String("password", "", "user's password")
	flag.Parse()

	connectionString := fmt.Sprintf("postgresql://%s:%s@%s/%s", *pg_user, *pg_password, *pg_url, *pg_db)
	conn, err := pgx.Connect(context.Background(), connectionString)
	if err != nil {
		os.Exit(1)
	}

	_, err = mail.ParseAddress(*user_email)
	if err != nil {
		os.Exit(2)
	}

	err = create_user(conn, *user_email, *user_name, *user_password)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}
}

func create_user(conn *pgx.Conn, email string, name string, password string) error {
	tx, err := conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	var user_id uuid.UUID
	insert_user_query := "INSERT INTO user_ (email, name, permission_roles) VALUES ($1, $2, '{\"owner\"}') RETURNING _id"
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
