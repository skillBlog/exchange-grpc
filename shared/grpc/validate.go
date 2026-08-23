package grpc

import (
	"context"
	"fmt"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// NewUnaryServerProtoValidate проверяет входящие protobuf-сообщения по buf validate правилам.
func NewUnaryServerProtoValidate(validator protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if validator == nil {
			return nil, status.Error(codes.Internal, "protovalidate validator is not configured")
		}
		if msg, ok := req.(proto.Message); ok && msg != nil {
			if err := validator.Validate(msg); err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		return handler(ctx, req)
	}
}

// NewStreamServerProtoValidate проверяет входящие protobuf-сообщения streaming RPC.
func NewStreamServerProtoValidate(validator protovalidate.Validator) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if validator == nil {
			return status.Error(codes.Internal, "protovalidate validator is not configured")
		}
		return handler(srv, &validatingServerStream{ServerStream: stream, validator: validator})
	}
}

type validatingServerStream struct {
	grpc.ServerStream
	validator protovalidate.Validator
}

func (s *validatingServerStream) RecvMsg(m any) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if msg, ok := m.(proto.Message); ok && msg != nil {
		if err := s.validator.Validate(msg); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
	}
	return nil
}

// NewProtoValidator создаёт protovalidate.Validator.
func NewProtoValidator() (protovalidate.Validator, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("create proto validator: %w", err)
	}
	return validator, nil
}
