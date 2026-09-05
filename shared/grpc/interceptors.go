package grpc

import (
	"buf.build/go/protovalidate"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// UnaryServerInterceptors — канонический порядок unary-цепочки:
// recovery → request_id → validate → logging → jwt → span attrs.
func UnaryServerInterceptors(
	log *zap.Logger,
	validator protovalidate.Validator,
	tokens *sessionvalidation.TokenService,
	publicMethods ...string,
) grpc.UnaryServerInterceptor {
	return ChainUnaryServer(
		UnaryServerRecovery(log),
		UnaryServerRequestID,
		NewUnaryServerProtoValidate(validator),
		UnaryServerLogging(log),
		NewUnaryServerJWTAuth(tokens, publicMethods...),
		UnaryServerSpanAttrs(),
	)
}

// StreamServerInterceptors — канонический порядок stream-цепочки:
// recovery → request_id → validate → logging → jwt → span attrs.
func StreamServerInterceptors(
	log *zap.Logger,
	validator protovalidate.Validator,
	tokens *sessionvalidation.TokenService,
	publicMethods ...string,
) grpc.StreamServerInterceptor {
	return ChainStreamServer(
		StreamServerRecovery(log),
		StreamServerRequestID,
		NewStreamServerProtoValidate(validator),
		StreamServerLogging(log),
		NewStreamServerJWTAuth(tokens, publicMethods...),
		StreamServerSpanAttrs(),
	)
}
