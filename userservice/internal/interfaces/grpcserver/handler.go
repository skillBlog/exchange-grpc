package grpcserver

import (
	"context"

	userv1 "github.com/exchange-grpc/proto/pb/user/v1"
	"github.com/exchange-grpc/userservice/internal/application"
)

// Server реализует user.v1.UserService.
type Server struct {
	userv1.UnimplementedUserServiceServer
	register *application.Register
	login    *application.Login
}

// NewServer создаёт gRPC-сервер User.
func NewServer(register *application.Register, login *application.Login) *Server {
	return &Server{
		register: register,
		login:    login,
	}
}

// Register создаёт нового пользователя.
func (s *Server) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	out, err := s.register.Execute(ctx, application.RegisterInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &userv1.RegisterResponse{UserId: out.UserID}, nil
}

// Login аутентифицирует пользователя и возвращает JWT.
func (s *Server) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	out, err := s.login.Execute(ctx, application.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &userv1.LoginResponse{AccessToken: out.AccessToken}, nil
}
