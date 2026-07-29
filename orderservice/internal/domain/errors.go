package domain

import sharederrors "github.com/exchange-grpc/shared/errors"

var (
	ErrNotFound           = sharederrors.ErrNotFound
	ErrInvalidArgument    = sharederrors.ErrInvalidArgument
	ErrFailedPrecondition = sharederrors.ErrFailedPrecondition
	ErrForbidden          = sharederrors.ErrForbidden
	ErrAlreadyExists      = sharederrors.ErrAlreadyExists
	ErrRateLimited        = sharederrors.ErrRateLimited
)

// ErrMarketInactive — доменный алиас для неактивного рынка (failed precondition).
var ErrMarketInactive = ErrFailedPrecondition
