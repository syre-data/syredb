package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"syredb/database"
	"time"

	"crypto/rand"
	"crypto/subtle"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

const PasswordHashAlgoId = "argon2id"
const PasswordHashHashLengthBytes = 512
const PasswordHashSaltLengthBytes = 64
const PasswordHashIterations = 2
const PasswordHashMemory = 64 * 1024
const PasswordHashParallelism = 4

type AuthService struct {
	ctx    context.Context
	logger *slog.Logger
	db     *database.DBConnection
}

func NewAuthService(
	ctx context.Context,
	logger *slog.Logger,
	db *database.DBConnection,
) *AuthService {
	return &AuthService{ctx: ctx, logger: logger, db: db}
}

func (s *AuthService) EncodePassword(password string) string {
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

	if vals[1] != PasswordHashAlgoId {
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

func (s *AuthService) ComparePasswordAndHash(
	clear_text_password string,
	encoded_hash string,
) (match bool, err error) {
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

const JWTSessionIdKey = "session_id"

func (s *AuthService) CreateSession(user uuid.UUID, expires time.Time) (uuid.UUID, error) {
	tx, err := s.db.Conn.Begin(s.ctx)
	if err != nil {
		s.logger.With("error", err).Error("could not begin transaction")
		return uuid.Nil, err
	}
	defer tx.Rollback(s.ctx)

	var session uuid.UUID
	create_query :=
		`INSERT INTO _user_session_ (_user, _expires, active) 
		VALUES ($1, $2, true) RETURNING _token`
	err = tx.QueryRow(
		s.ctx,
		create_query,
		user,
		expires,
	).Scan(&session)
	if err != nil {
		s.logger.With(
			"error", err,
			"user", user,
		).Error("could not create new user session")
		return uuid.Nil, err
	}

	err = tx.Commit(s.ctx)
	return session, err
}

func (s *AuthService) UserFromToken(token uuid.UUID) (uuid.UUID, error) {
	var user uuid.UUID
	query := "SELECT _user FROM _user_session_ WHERE _token=$1 AND _expires>$2 AND active=true"
	err := s.db.Conn.QueryRow(s.ctx, query, token, time.Now()).Scan(&user)
	if err != nil {
		s.logger.With("error", err, "token", token).Error("could not get session user")
		return uuid.Nil, err
	}
	return user, nil
}

func (s *AuthService) DeactivateSession(token uuid.UUID) error {
	disable_query := "UPDATE _user_session_ SET active=false WHERE _token=$1"
	_, err := s.db.Conn.Exec(s.ctx, disable_query, token)
	if err != nil {
		s.logger.With(
			"error", err,
			"session token", token,
		).Error("could not deactivate previous user sessions")
		return err
	}

	return nil
}
