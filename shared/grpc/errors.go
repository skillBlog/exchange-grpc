package grpc

import (
	"errors"

	sharederrors "github.com/exchange-grpc/shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Re-export общих sentinel-ошибок для обратной совместимости.
var (
	ErrInvalidArgument    = sharederrors.ErrInvalidArgument
	ErrNotFound           = sharederrors.ErrNotFound
	ErrForbidden          = sharederrors.ErrForbidden
	ErrUnauthorized       = sharederrors.ErrUnauthorized
	ErrAlreadyExists      = sharederrors.ErrAlreadyExists
	ErrFailedPrecondition = sharederrors.ErrMarketInactive
)

// ErrorMapping описывает соответствие domain-ошибки gRPC status code.
type ErrorMapping struct {
	Sentinel error
	Code     codes.Code
	Message  string
}

// StatusFromError преобразует ошибку в gRPC status.
// extra позволяет сервисам добавить свои domain sentinel-ошибки поверх общих.
func StatusFromError(err error, extra ...ErrorMapping) error {
	if err == nil {
		return nil
	}

	for _, mapping := range extra {
		if errors.Is(err, mapping.Sentinel) {
			message := mapping.Message
			if message == "" {
				message = mapping.Sentinel.Error()
			}
			return status.Error(mapping.Code, message)
		}
	}

	switch {
	case errors.Is(err, sharederrors.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, sharederrors.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, sharederrors.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, sharederrors.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, sharederrors.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, sharederrors.ErrMarketInactive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, sharederrors.ErrRateLimited):
		return status.Error(codes.ResourceExhausted, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
