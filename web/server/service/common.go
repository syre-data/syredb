package service

import (
	"strings"

	"github.com/google/uuid"
)

type InsufficientPermissionsError struct{}

func (e *InsufficientPermissionsError) Error() string {
	return "INSUFFICIENT_PERMISSIONS"
}

func uuid_to_sql_string(id uuid.UUID) string {
	return strings.ReplaceAll(id.String(), "-", "_")
}
