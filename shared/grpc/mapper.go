package grpc

import (
	sharederrors "github.com/exchange-grpc/shared/errors"
	"google.golang.org/grpc/codes"
)

// DefaultErrorMappings возвращает стандартный набор маппингов domain → gRPC code.
func DefaultErrorMappings(rateLimitedMessage string) []ErrorMapping {
	if rateLimitedMessage == "" {
		rateLimitedMessage = sharederrors.ErrRateLimited.Error()
	}
	return []ErrorMapping{
		{Sentinel: sharederrors.ErrInvalidArgument, Code: codes.InvalidArgument},
		{Sentinel: sharederrors.ErrNotFound, Code: codes.NotFound},
		{Sentinel: sharederrors.ErrForbidden, Code: codes.PermissionDenied},
		{Sentinel: sharederrors.ErrUnauthorized, Code: codes.Unauthenticated},
		{Sentinel: sharederrors.ErrAlreadyExists, Code: codes.AlreadyExists},
		{Sentinel: sharederrors.ErrMarketInactive, Code: codes.FailedPrecondition},
		{Sentinel: sharederrors.ErrRateLimited, Code: codes.ResourceExhausted, Message: rateLimitedMessage},
	}
}

// ToStatusError преобразует domain-ошибку в gRPC status с общими маппингами.
func ToStatusError(err error, rateLimitedMessage string, extra ...ErrorMapping) error {
	mappings := DefaultErrorMappings(rateLimitedMessage)
	mappings = append(mappings, extra...)
	return StatusFromError(err, mappings...)
}
