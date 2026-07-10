package grpc

import (
	"context"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// NewUnaryServerProtoValidate проверяет входящие protobuf-сообщения по buf validate правилам.
func NewUnaryServerProtoValidate(validator protovalidate.Validator) grpc.UnaryServerInterceptor {
	if validator == nil {
		panic("protovalidate validator is required")
	}

	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if msg, ok := req.(proto.Message); ok && msg != nil {
			if err := validator.Validate(msg); err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		return handler(ctx, req)
	}
}

// MustNewProtoValidator создаёт protovalidate.Validator или паникует при ошибке инициализации.
func MustNewProtoValidator() protovalidate.Validator {
	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}
	return validator
}
