package grpc

import (
	"buf.build/go/protovalidate"
	"github.com/exchange-grpc/shared/metrics"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// UnaryServerInterceptors — канонический порядок unary-цепочки:
// recovery → request_id → metrics → validate → logging → jwt → span attrs.
func UnaryServerInterceptors(
	log *zap.Logger,
	validator protovalidate.Validator,
	tokens *sessionvalidation.TokenService,
	publicMethods ...string,
) grpc.UnaryServerInterceptor {
	return ChainUnaryServer(
		UnaryServerRecovery(log),
		UnaryServerRequestID,
		metrics.UnaryServerInterceptor(),
		NewUnaryServerProtoValidate(validator),
		UnaryServerLogging(log),
		NewUnaryServerJWTAuth(tokens, publicMethods...),
		UnaryServerSpanAttrs(),
	)
}

// StreamServerInterceptors — канонический порядок stream-цепочки:
// recovery → request_id → metrics → validate → logging → jwt → span attrs.
// JWT проверяется один раз при старте RPC из metadata, не на каждый RecvMsg.
func StreamServerInterceptors(
	log *zap.Logger,
	validator protovalidate.Validator,
	tokens *sessionvalidation.TokenService,
	publicMethods ...string,
) grpc.StreamServerInterceptor {
	return ChainStreamServer(
		StreamServerRecovery(log),
		StreamServerRequestID,
		metrics.StreamServerInterceptor(),
		NewStreamServerProtoValidate(validator),
		StreamServerLogging(log),
		NewStreamServerJWTAuth(tokens, publicMethods...),
		StreamServerSpanAttrs(),
	)
}
