package grpc_test

import (
	"testing"

	sharederrors "github.com/exchange-grpc/shared/errors"
	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusFromError_commonMappings(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "not found", err: sharederrors.ErrNotFound, code: codes.NotFound},
		{name: "forbidden", err: sharederrors.ErrForbidden, code: codes.PermissionDenied},
		{name: "unauthorized", err: sharederrors.ErrUnauthorized, code: codes.Unauthenticated},
		{name: "rate limited", err: sharederrors.ErrRateLimited, code: codes.ResourceExhausted},
		{name: "market inactive", err: sharederrors.ErrMarketInactive, code: codes.FailedPrecondition},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st, ok := status.FromError(sharedgrpc.StatusFromError(tc.err))
			if !ok {
				t.Fatal("expected gRPC status")
			}
			if st.Code() != tc.code {
				t.Fatalf("code = %v, want %v", st.Code(), tc.code)
			}
		})
	}
}

func TestToStatusError_rateLimitedMessage(t *testing.T) {
	err := sharedgrpc.ToStatusError(sharederrors.ErrRateLimited, "too many requests")
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status")
	}
	if st.Code() != codes.ResourceExhausted || st.Message() != "too many requests" {
		t.Fatalf("unexpected status: %v", st)
	}
}
