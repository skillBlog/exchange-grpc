package grpc

import (
	"errors"

	sharederrors "github.com/exchange-grpc/shared/errors"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorMapping описывает соответствие domain-ошибки gRPC status code.
type ErrorMapping struct {
	Sentinel error
	Code     codes.Code
	Message  string
}

func defaultErrorMappings(rateLimitedMessage string) []ErrorMapping {
	if rateLimitedMessage == "" {
		rateLimitedMessage = sharederrors.ErrRateLimited.Error()
	}
	return []ErrorMapping{
		{Sentinel: sharederrors.ErrInvalidArgument, Code: codes.InvalidArgument},
		{Sentinel: sharederrors.ErrNotFound, Code: codes.NotFound},
		{Sentinel: sharederrors.ErrForbidden, Code: codes.PermissionDenied},
		{Sentinel: sharederrors.ErrUnauthorized, Code: codes.Unauthenticated},
		{Sentinel: sessionvalidation.ErrInvalidToken, Code: codes.Unauthenticated},
		{Sentinel: sharederrors.ErrAlreadyExists, Code: codes.AlreadyExists},
		{Sentinel: sharederrors.ErrFailedPrecondition, Code: codes.FailedPrecondition},
		{Sentinel: sharederrors.ErrRateLimited, Code: codes.ResourceExhausted, Message: rateLimitedMessage},
	}
}

// StatusFromError преобразует ошибку в gRPC status.
// extra позволяет сервисам переопределить общие маппинги или добавить свои sentinel-ошибки.
func StatusFromError(err error, extra ...ErrorMapping) error {
	return mapStatus(err, extra, defaultErrorMappings(""))
}

// ToStatusError преобразует domain-ошибку в gRPC status с общим маппингом.
// extra проверяется раньше defaults, поэтому сервисы могут задать свой message/code.
func ToStatusError(err error, rateLimitedMessage string, extra ...ErrorMapping) error {
	return mapStatus(err, extra, defaultErrorMappings(rateLimitedMessage))
}

func mapStatus(err error, extra, defaults []ErrorMapping) error {
	if err == nil {
		return nil
	}
	for _, mapping := range extra {
		if errors.Is(err, mapping.Sentinel) {
			return status.Error(mapping.Code, mappingMessage(err, mapping))
		}
	}
	for _, mapping := range defaults {
		if errors.Is(err, mapping.Sentinel) {
			return status.Error(mapping.Code, mappingMessage(err, mapping))
		}
	}
	return status.Error(codes.Internal, "internal error")
}

func mappingMessage(err error, mapping ErrorMapping) string {
	if mapping.Message != "" {
		return mapping.Message
	}
	return err.Error()
}
