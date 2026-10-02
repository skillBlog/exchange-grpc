package grpc

import (
	"buf.build/go/protovalidate"
	"github.com/exchange-grpc/shared/metrics"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// UnaryServerInterceptors — канонический порядок unary-цепочки:
// request_id → recovery → metrics → validate → logging → jwt → span attrs.
// request_id стоит перед recovery, чтобы panic-лог содержал request_id.
func UnaryServerInterceptors(
	log *zap.Logger,
	validator protovalidate.Validator,
	tokens *sessionvalidation.TokenService,
	publicMethods ...string,
) grpc.UnaryServerInterceptor {
	return ChainUnaryServer(
		UnaryServerRequestID,
		UnaryServerRecovery(log),
		metrics.UnaryServerInterceptor(),
		NewUnaryServerProtoValidate(validator),
		UnaryServerLogging(log),
		NewUnaryServerJWTAuth(tokens, publicMethods...),
		UnaryServerSpanAttrs(),
	)
}

// StreamServerInterceptors — канонический порядок stream-цепочки:
// request_id → recovery → metrics → validate → logging → jwt → span attrs.
// request_id стоит перед recovery, чтобы panic-лог содержал request_id.
// JWT проверяется один раз при старте RPC из metadata, не на каждый RecvMsg.
func StreamServerInterceptors(
	log *zap.Logger,
	validator protovalidate.Validator,
	tokens *sessionvalidation.TokenService,
	publicMethods ...string,
) grpc.StreamServerInterceptor {
	return ChainStreamServer(
		StreamServerRequestID,
		StreamServerRecovery(log),
		metrics.StreamServerInterceptor(),
		NewStreamServerProtoValidate(validator),
		StreamServerLogging(log),
		NewStreamServerJWTAuth(tokens, publicMethods...),
		StreamServerSpanAttrs(),
	)
}
