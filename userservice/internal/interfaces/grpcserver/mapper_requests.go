package grpcserver

import (
	userv1 "github.com/exchange-grpc/proto/pb/user/v1"
	"github.com/exchange-grpc/userservice/internal/application"
)

// Mapper преобразует protobuf-запросы в application input/output.
type Mapper struct{}

func (Mapper) RegisterRequestToInput(req *userv1.RegisterRequest) application.RegisterInput {
	return application.RegisterInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}
}

func (Mapper) RegisterOutputToResponse(out application.RegisterOutput) *userv1.RegisterResponse {
	return &userv1.RegisterResponse{UserId: out.UserID}
}

func (Mapper) LoginRequestToInput(req *userv1.LoginRequest) application.LoginInput {
	return application.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}
}

func (Mapper) LoginOutputToResponse(out application.LoginOutput) *userv1.LoginResponse {
	return &userv1.LoginResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	}
}

func (Mapper) RefreshTokenRequestToInput(req *userv1.RefreshTokenRequest) application.RefreshTokenInput {
	return application.RefreshTokenInput{RefreshToken: req.GetRefreshToken()}
}

func (Mapper) RefreshTokenOutputToResponse(out application.RefreshTokenOutput) *userv1.RefreshTokenResponse {
	return &userv1.RefreshTokenResponse{AccessToken: out.AccessToken}
}

func (Mapper) GetUserOutputToResponse(out application.GetUserOutput) *userv1.GetUserResponse {
	return &userv1.GetUserResponse{
		UserId: out.UserID,
		Email:  out.Email,
		Roles:  append([]string(nil), out.Roles...),
	}
}

func (Mapper) LogoutRequestToInput(req *userv1.LogoutRequest) application.LogoutInput {
	return application.LogoutInput{RefreshToken: req.GetRefreshToken()}
}
