package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ParseUUID проверяет и нормализует UUID-строку.
func ParseUUID(raw, field string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidArgument, field)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: invalid %s %q", ErrInvalidArgument, field, raw)
	}
	return id.String(), nil
}
