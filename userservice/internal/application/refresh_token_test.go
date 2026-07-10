package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/userservice/internal/infrastructure/ratelimit"
	"github.com/exchange-grpc/userservice/internal/infrastructure/tokens"
)

func TestRefreshToken_success(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := tokens.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher())
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "refresh@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute))
	loginOut, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "refresh@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login error = %v", err)
	}

	refreshUC := application.NewRefreshToken(repo, accessTokens, refreshTokens)
	out, err := refreshUC.Execute(context.Background(), application.RefreshTokenInput{
		RefreshToken: loginOut.RefreshToken,
	})
	if err != nil {
		t.Fatalf("refresh error = %v", err)
	}
	if out.AccessToken == "" {
		t.Fatal("expected new access token")
	}
}

func TestGetUser_success(t *testing.T) {
	repo := memory.NewUserRepository()
	register := application.NewRegister(repo, bcrypt.NewHasher())
	registerOut, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "profile@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register error = %v", err)
	}

	getUser := application.NewGetUser(repo)
	out, err := getUser.Execute(context.Background(), application.GetUserInput{UserID: registerOut.UserID})
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	if out.Email != "profile@example.com" {
		t.Fatalf("email = %q", out.Email)
	}
}

func TestLogout_revokesRefreshToken(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := tokens.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher())
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "logout@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute))
	loginOut, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "logout@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login error = %v", err)
	}

	logout := application.NewLogout(refreshTokens)
	if err := logout.Execute(context.Background(), application.LogoutInput{RefreshToken: loginOut.RefreshToken}); err != nil {
		t.Fatalf("logout error = %v", err)
	}

	refreshUC := application.NewRefreshToken(repo, accessTokens, refreshTokens)
	if _, err := refreshUC.Execute(context.Background(), application.RefreshTokenInput{RefreshToken: loginOut.RefreshToken}); err == nil {
		t.Fatal("expected refresh to fail after logout")
	}
}
