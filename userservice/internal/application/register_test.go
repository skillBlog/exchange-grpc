package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
)

func newRegisterUC(t *testing.T, repo domain.UserRepository) *application.Register {
	t.Helper()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour)
	return application.NewRegister(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, nil)
}

func TestRegister_success(t *testing.T) {
	repo := memory.NewUserRepository()
	uc := newRegisterUC(t, repo)

	out, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "user@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.UserID == "" {
		t.Fatal("expected user id")
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
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
	uc := newRegisterUC(t, repo)

	input := application.RegisterInput{
		Email:    "dup@example.com",
		Password: "Password1!",
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
	uc := newRegisterUC(t, memory.NewUserRepository())

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "user@example.com",
		Password: "short",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestRegister_weakPassword(t *testing.T) {
	uc := newRegisterUC(t, memory.NewUserRepository())

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "user@example.com",
		Password: "12345678",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestRegister_passwordWithoutSpecial(t *testing.T) {
	uc := newRegisterUC(t, memory.NewUserRepository())

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "user@example.com",
		Password: "Password1",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestRegister_normalizesEmailForLookup(t *testing.T) {
	repo := memory.NewUserRepository()
	uc := newRegisterUC(t, repo)

	out, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "  Foo@Bar.COM ",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("register error = %v", err)
	}

	user, err := repo.GetByEmail(context.Background(), "FOO@bar.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if user.ID != out.UserID {
		t.Fatalf("user id = %q, want %q", user.ID, out.UserID)
	}
	if user.Email != "foo@bar.com" {
		t.Fatalf("stored email = %q, want foo@bar.com", user.Email)
	}

	login := application.NewLogin(
		repo,
		bcrypt.NewHasher(0),
		mustAccessTokens(t),
		sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour),
		nil,
		nil,
	)
	if _, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "Foo@Bar.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("login error = %v", err)
	}
}
