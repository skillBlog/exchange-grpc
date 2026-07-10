package grpcserver

import (
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/userservice/internal/domain"
	"google.golang.org/grpc/codes"
)

func toGRPCError(err error) error {
	return grpc.ToStatusError(err, "too many login attempts",
		grpc.ErrorMapping{Sentinel: domain.ErrUnauthorized, Code: codes.Unauthenticated, Message: "invalid credentials"},
	)
}
