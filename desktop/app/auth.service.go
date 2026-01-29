package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"crypto/rand"
	"crypto/subtle"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/crypto/argon2"
)

const USER_AUTH_FILE_NAME = "user_auth.toml"

type UserNotAuthenticatedError struct{}

func (e *UserNotAuthenticatedError) Error() string {
	return "USER_NOT_AUTHENTICATED"
}

type InsufficientPermissionsError struct{}

func (e *InsufficientPermissionsError) Error() string {
	return "INSUFFICIENT_PERMISSIONS"
}

type AuthService struct {
	ctx       context.Context
	logger    *slog.Logger
	db        *DbConnection
	app_dir   string
	app_state *AppState
}

func NewAuthService(
	logger *slog.Logger,
	db *DbConnection,
	app_dir string,
	app_state *AppState,
) *AuthService {
	return &AuthService{logger: logger, db: db, app_dir: app_dir, app_state: app_state}
}

func (s *AuthService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

type UserAuth struct {
	UserId    string
	AuthToken string
}

func (s *AuthService) UserFromLocal() (User, error) {
	user, err := s.user_from_local(s.db.conn, s.app_dir)
	if err != nil {
		return User{}, err
	}

	s.app_state._lock.Lock()
	defer s.app_state._lock.Unlock()
	s.app_state.user_id = user.Id
	return user, nil
}

func (s *AuthService) user_from_local(
	db *pgxpool.Pool,
	app_dir string,
) (User, error) {
	var user_auth UserAuth
	auth_file_path := filepath.Join(app_dir, USER_AUTH_FILE_NAME)
	_, err := toml.DecodeFile(auth_file_path, &user_auth)
	if err != nil {
		var parse_err toml.ParseError
		if errors.Is(err, os.ErrNotExist) {
			s.logger.With("error", err).Error("user auth file not found")
			return User{}, nil
		} else if errors.As(err, &parse_err) {
			s.logger.With("error", err).Error("user auth file invalid format")
			return User{}, err
		} else {
			s.logger.With("error", err).Error("user auth file invalid")
			return User{}, err
		}
	}

	var db_auth_tokens []string
	auth_row := db.QueryRow(s.ctx, "SELECT tokens FROM user_auth_ WHERE _id=$1", user_auth.UserId)
	err = auth_row.Scan(&db_auth_tokens)
	if err != nil {
		s.logger.With("error", err).Error("could not get user auth tokens")
		return User{}, err
	}

	if !slices.Contains(db_auth_tokens, user_auth.AuthToken) {
		return User{}, nil
	}

	var user User
	user_row := db.QueryRow(s.ctx, "SELECT _id, email, name, role FROM user_ WHERE _id=$1", user_auth.UserId)
	err = user_row.Scan(&user.Id, &user.Email, &user.Name, &user.Role)
	if err != nil {
		s.logger.With("error", err).Error("could not get user")
		return User{}, err
	}

	return user, nil
}

type UserCredentials struct {
	Email    string
	Password string
}

func (s *AuthService) AuthenticateAndGet(credentials UserCredentials, remember bool) (User, error) {
	user, err := s.authenticate_and_get(credentials, remember, s.db.conn, s.app_dir)
	if err != nil {
		return User{}, err
	}

	s.app_state._lock.Lock()
	defer s.app_state._lock.Unlock()
	s.app_state.user_id = user.Id
	return user, nil
}

func (s *AuthService) authenticate_and_get(
	credentials UserCredentials,
	remember bool,
	db *pgxpool.Pool,
	app_dir string,
) (User, error) {
	var user User
	user_row := db.QueryRow(
		s.ctx,
		"SELECT _id, email, name, role FROM user_ WHERE email=$1",
		credentials.Email,
	)
	err := user_row.Scan(&user.Id, &user.Email, &user.Name, &user.Role)
	if err != nil {
		s.logger.With("error", err).Error("could not get user")
		return User{}, nil
	}

	var auth_hash string
	auth_row := db.QueryRow(s.ctx, "SELECT auth FROM user_auth_ WHERE _id=$1", user.Id.String())
	err = auth_row.Scan(&auth_hash)
	if err != nil {
		s.logger.With("error", err).Error("could not get user auth")
		return User{}, nil
	}

	authorized, err := comparePasswordAndHash(credentials.Password, auth_hash)
	if err != nil {
		s.logger.With("error", err).Error("could not compare password to hash")
		return User{}, nil
	}

	if !authorized {
		return User{}, nil
	}

	user_auth_file_path := filepath.Join(app_dir, USER_AUTH_FILE_NAME)
	if remember {
		s.remember_user_token(user.Id, user_auth_file_path)
	} else {
		err = os.Remove(user_auth_file_path)
		if err != nil {
			s.logger.With("error", err).Error("could not remove user auth file")
		}
	}

	return user, nil
}

func (s *AuthService) remember_user_token(
	user uuid.UUID,
	auth_file string,
) error {
	auth_token_b := make([]byte, PASSWORD_HASH_SALT_LENGTH_BYTES)
	rand.Read(auth_token_b)
	auth_token := base64.RawStdEncoding.EncodeToString(auth_token_b)

	append_token_query := "UPDATE user_auth_ SET tokens=ARRAY_APPEND(tokens, $1) WHERE _id=$2"
	_, err := s.db.conn.Exec(s.ctx, append_token_query, auth_token, user)
	if err != nil {
		s.logger.With("error", err).Error("could not insert user auth token")
		return err
	}

	user_auth := UserAuth{UserId: user.String(), AuthToken: auth_token}
	user_auth_toml := new(bytes.Buffer)
	err = toml.NewEncoder(user_auth_toml).Encode(user_auth)
	if err != nil {
		s.logger.With("error", err).Error("could not save user auth token")
		return err
	}

	f, err := os.OpenFile(auth_file, os.O_CREATE|os.O_WRONLY, FILE_PERMISSIONS_WRR)
	if err != nil {
		s.logger.With("error", err).Error("could not save user auth token")
		return err
	}
	defer f.Close()
	_, err = f.Write(user_auth_toml.Bytes())
	if err != nil {
		s.logger.With("error", err).Error("could not save user auth token")
		return err
	}

	return nil
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

func (s *AuthService) Logout() (Ok, error) {
	s.app_state._lock.Lock()
	s.app_state.user_id = uuid.Nil
	s.app_state._lock.Unlock()

	err := s.logout(s.app_dir)
	return Ok{}, err
}

func (s *AuthService) logout(app_dir string) error {
	user_auth_file_path := filepath.Join(app_dir, USER_AUTH_FILE_NAME)
	err := os.Remove(user_auth_file_path)
	if err != nil {
		s.logger.With("error", err).Error("could not remove user auth file")
	}

	return err
}
