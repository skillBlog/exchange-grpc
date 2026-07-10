package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
)

func TestRegister_success(t *testing.T) {
	repo := memory.NewUserRepository()
	uc := application.NewRegister(repo, bcrypt.NewHasher())

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

	user, err := repo.GetByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if len(user.Roles) != 1 || user.Roles[0] != roles.RoleUser {
		t.Fatalf("roles = %v, want [user]", user.Roles)
	}
}

func TestRegister_duplicateEmail(t *testing.T) {
	repo := memory.NewUserRepository()
	uc := application.NewRegister(repo, bcrypt.NewHasher())

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
	uc := application.NewRegister(memory.NewUserRepository(), bcrypt.NewHasher())

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "user@example.com",
		Password: "short",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestRegister_weakPassword(t *testing.T) {
	uc := application.NewRegister(memory.NewUserRepository(), bcrypt.NewHasher())

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "user@example.com",
		Password: "12345678",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}
