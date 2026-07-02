package sessionvalidation_test

import (
	"testing"
	"time"

	"github.com/exchange-grpc/shared/sessionvalidation"
)

func TestTokenService_issueAndValidate(t *testing.T) {
	svc, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}

	token, err := svc.Issue("user-1", []string{"trader"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", claims.UserID)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "trader" {
		t.Fatalf("Roles = %v, want [trader]", claims.Roles)
	}
}

func TestTokenService_rejectsInvalidToken(t *testing.T) {
	svc, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}

	if _, err := svc.Validate("not-a-jwt"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}
