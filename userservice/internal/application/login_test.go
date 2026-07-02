package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
)

func TestLogin_success(t *testing.T) {
	repo := memory.NewUserRepository()
	tokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}

	register := application.NewRegister(repo)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "login@example.com",
		Password: "password123",
		Roles:    []string{"trader"},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, tokens)
	out, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "login@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.AccessToken == "" {
		t.Fatal("expected access token")
	}

	claims, err := tokens.Validate(out.AccessToken)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID == "" {
		t.Fatal("expected user id in claims")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "trader" {
		t.Fatalf("Roles = %v, want [trader]", claims.Roles)
	}
}

func TestLogin_invalidPassword(t *testing.T) {
	repo := memory.NewUserRepository()
	tokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}

	register := application.NewRegister(repo)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "login@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, tokens)
	_, err = login.Execute(context.Background(), application.LoginInput{
		Email:    "login@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
}
