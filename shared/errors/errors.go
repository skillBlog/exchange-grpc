package errors

import "errors"

// Общие sentinel-ошибки домена для всех сервисов.
var (
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrNotFound           = errors.New("not found")
	ErrForbidden          = errors.New("forbidden")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrAlreadyExists      = errors.New("already exists")
	ErrFailedPrecondition = errors.New("failed precondition")
	ErrConflict           = errors.New("conflict")
	ErrRateLimited        = errors.New("rate limited")
)
