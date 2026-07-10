package domain

import sharederrors "github.com/exchange-grpc/shared/errors"

var (
	ErrAlreadyExists   = sharederrors.ErrAlreadyExists
	ErrUnauthorized    = sharederrors.ErrUnauthorized
	ErrInvalidArgument = sharederrors.ErrInvalidArgument
	ErrNotFound        = sharederrors.ErrNotFound
	ErrRateLimited     = sharederrors.ErrRateLimited
)
