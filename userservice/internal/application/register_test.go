package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
)

func TestRegister_success(t *testing.T) {
	repo := memory.NewUserRepository()
	uc := application.NewRegister(repo)

	out, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "user@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.UserID == "" {
		t.Fatal("expected user id")
	}

	_, err = repo.GetByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
}

func TestRegister_duplicateEmail(t *testing.T) {
	repo := memory.NewUserRepository()
	uc := application.NewRegister(repo)

	input := application.RegisterInput{
		Email:    "dup@example.com",
		Password: "password123",
	}
	if _, err := uc.Execute(context.Background(), input); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}

	_, err := uc.Execute(context.Background(), input)
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("error = %v, want ErrAlreadyExists", err)
	}
}

func TestRegister_shortPassword(t *testing.T) {
	uc := application.NewRegister(memory.NewUserRepository())

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "user@example.com",
		Password: "short",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
