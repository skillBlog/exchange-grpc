package grpcserver

import (
	"strings"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	userv1 "github.com/exchange-grpc/proto/pb/user/v1"
	"github.com/exchange-grpc/shared/roles"
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
	return &userv1.RegisterResponse{
		UserId:       out.UserID,
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	}
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
	return &userv1.RefreshTokenResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	}
}

func (Mapper) GetUserOutputToResponse(out application.GetUserOutput) *userv1.GetUserResponse {
	return &userv1.GetUserResponse{
		UserId: out.UserID,
		Email:  out.Email,
		Roles:  rolesToProto(out.Roles),
	}
}

func (Mapper) LogoutRequestToInput(req *userv1.LogoutRequest) application.LogoutInput {
	return application.LogoutInput{RefreshToken: req.GetRefreshToken()}
}

func rolesToProto(values []string) []commonv1.Role {
	result := make([]commonv1.Role, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		role, ok := roles.Parse(value)
		if !ok {
			result = append(result, commonv1.Role_ROLE_UNSPECIFIED)
			continue
		}
		switch role {
		case roles.RoleUser:
			result = append(result, commonv1.Role_ROLE_USER)
		case roles.RoleTrader:
			result = append(result, commonv1.Role_ROLE_TRADER)
		case roles.RoleAdmin:
			result = append(result, commonv1.Role_ROLE_ADMIN)
		default:
			result = append(result, commonv1.Role_ROLE_UNSPECIFIED)
		}
	}
	return result
}
