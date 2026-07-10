package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/userservice/internal/infrastructure/ratelimit"
	"github.com/exchange-grpc/userservice/internal/infrastructure/tokens"
)

func TestLogin_success(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := tokens.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher())
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "login@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute))
	out, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "login@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	claims, err := accessTokens.Validate(out.AccessToken)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID == "" {
		t.Fatal("expected user id in claims")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "user" {
		t.Fatalf("Roles = %v, want [user]", claims.Roles)
	}
}

func TestLogin_invalidPassword(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := tokens.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher())
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "login@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute))
	_, err = login.Execute(context.Background(), application.LoginInput{
		Email:    "login@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
}

func TestLogin_rateLimited(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := tokens.NewRefreshTokenService(refreshRepo, 24*time.Hour)
	limiter := ratelimit.NewLoginLimiter(1, time.Minute)

	login := application.NewLogin(repo, bcrypt.NewHasher(), accessTokens, refreshTokens, limiter)

	_, err = login.Execute(context.Background(), application.LoginInput{
		Email:    "missing@example.com",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("first error = %v, want ErrUnauthorized", err)
	}

	_, err = login.Execute(context.Background(), application.LoginInput{
		Email:    "missing@example.com",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second error = %v, want ErrRateLimited", err)
	}
}
