package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrMarketInactive  = errors.New("market inactive")
	ErrForbidden       = errors.New("forbidden")
)
