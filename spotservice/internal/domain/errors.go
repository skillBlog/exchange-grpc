package domain

import sharederrors "github.com/exchange-grpc/shared/errors"

var (
	ErrNotFound        = sharederrors.ErrNotFound
	ErrInvalidArgument = sharederrors.ErrInvalidArgument
	ErrMarketInactive  = sharederrors.ErrMarketInactive
	ErrForbidden       = sharederrors.ErrForbidden
	ErrRateLimited     = sharederrors.ErrRateLimited
)
