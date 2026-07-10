package grpcserver

import (
	"context"

	userv1 "github.com/exchange-grpc/proto/pb/user/v1"
	"github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/userservice/internal/application"
)

// Server реализует user.v1.UserService.
type Server struct {
	userv1.UnimplementedUserServiceServer
	mapper       Mapper
	register     *application.Register
	login        *application.Login
	refreshToken *application.RefreshToken
	getUser      *application.GetUser
	logout       *application.Logout
}

// NewServer создаёт gRPC-сервер User.
func NewServer(
	register *application.Register,
	login *application.Login,
	refreshToken *application.RefreshToken,
	getUser *application.GetUser,
	logout *application.Logout,
) *Server {
	return &Server{
		register:     register,
		login:        login,
		refreshToken: refreshToken,
		getUser:      getUser,
		logout:       logout,
	}
}

// Register создаёт нового пользователя.
func (s *Server) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	out, err := s.register.Execute(ctx, s.mapper.RegisterRequestToInput(req))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return s.mapper.RegisterOutputToResponse(out), nil
}

// Login аутентифицирует пользователя и возвращает JWT.
func (s *Server) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	out, err := s.login.Execute(ctx, s.mapper.LoginRequestToInput(req))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return s.mapper.LoginOutputToResponse(out), nil
}

// RefreshToken обновляет access token по refresh token.
func (s *Server) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	out, err := s.refreshToken.Execute(ctx, s.mapper.RefreshTokenRequestToInput(req))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return s.mapper.RefreshTokenOutputToResponse(out), nil
}

// GetUser возвращает профиль текущего пользователя.
func (s *Server) GetUser(ctx context.Context, _ *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	userID, ok := grpc.UserIDFromContext(ctx)
	if !ok {
		return nil, grpc.ErrMissingUserID()
	}

	out, err := s.getUser.Execute(ctx, application.GetUserInput{UserID: userID})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return s.mapper.GetUserOutputToResponse(out), nil
}

// Logout инвалидирует refresh token.
func (s *Server) Logout(ctx context.Context, req *userv1.LogoutRequest) (*userv1.LogoutResponse, error) {
	if err := s.logout.Execute(ctx, s.mapper.LogoutRequestToInput(req)); err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.LogoutResponse{}, nil
}
