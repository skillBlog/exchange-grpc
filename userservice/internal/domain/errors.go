package domain

import "errors"

var (
	// ErrAlreadyExists возвращается при регистрации существующего email.
	ErrAlreadyExists = errors.New("already exists")

	// ErrUnauthorized возвращается при неверных учётных данных.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidArgument возвращается при ошибке валидации.
	ErrInvalidArgument = errors.New("invalid argument")
)
