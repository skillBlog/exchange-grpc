package domain

import "errors"

var (
	// ErrNotFound возвращается, когда запрошенная сущность не существует.
	ErrNotFound = errors.New("not found")

	// ErrInvalidArgument возвращается при ошибке валидации входных данных домена.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrMarketInactive возвращается, когда рынок существует, но недоступен для торговли.
	ErrMarketInactive = errors.New("market inactive")

	// ErrForbidden возвращается, когда ролей пользователя недостаточно для операции.
	ErrForbidden = errors.New("forbidden")
)
