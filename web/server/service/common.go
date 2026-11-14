package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type InsufficientPermissionsError struct{}

func (e *InsufficientPermissionsError) Error() string {
	return "INSUFFICIENT_PERMISSIONS"
}

type dbtx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func uuid_to_sql_string(id uuid.UUID) string {
	return strings.ReplaceAll(id.String(), "-", "_")
}

// SaveFormFile saves the content represented by file to the path dst.
// Creates parent directories if needed.
func SaveFormFile(file *multipart.FileHeader, dst string) error {
	parent := filepath.Dir(dst)
	err := os.MkdirAll(parent, os.ModePerm)
	if err != nil {
		return err
	}

	fdst, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer fdst.Close()

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(fdst, src)
	if err != nil {
		return err
	}

	return nil
}

// SqlArgsPlaceholderList creates an arguments placeholder list with len elements, starting from 1.
// If len <= 0, returns an empty string.
//
// # Example
//
// SqlArgsPlaceholderList(4) // "$1, $2, $3, $4"
func SqlArgsPlaceholderList(len int) string {
	if len <= 0 {
		return ""
	}

	args := make([]string, len)
	for idx := range len {
		args[idx] = fmt.Sprintf("$%d", idx+1)
	}

	return strings.Join(args, ", ")
}
