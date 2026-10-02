package domain_test

import (
	"testing"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/userservice/internal/domain"
)

func TestNormalizeEmail(t *testing.T) {
	got := domain.NormalizeEmail("  Foo@Bar.COM ")
	if got != "foo@bar.com" {
		t.Fatalf("NormalizeEmail() = %q, want foo@bar.com", got)
	}
}

func TestNewUser_normalizesEmail(t *testing.T) {
	user, err := domain.NewUser("id-1", "  Foo@Bar.COM ", "hash", []roles.Role{roles.RoleUser})
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	if user.Email != "foo@bar.com" {
		t.Fatalf("Email = %q, want foo@bar.com", user.Email)
	}
}
