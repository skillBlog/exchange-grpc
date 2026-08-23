package tokens_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/userservice/internal/infrastructure/tokens"
)

func TestRefreshTokenService_RotateRevokesOld(t *testing.T) {
	svc := tokens.NewRefreshTokenService(memory.NewRefreshTokenRepository(), time.Hour)
	ctx := context.Background()

	oldRaw, err := svc.Issue(ctx, "user-1")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	newRaw, err := svc.Rotate(ctx, oldRaw, "user-1")
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if newRaw == "" || newRaw == oldRaw {
		t.Fatal("expected a new refresh token")
	}

	if _, err := svc.Validate(ctx, oldRaw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("old token error = %v, want ErrUnauthorized", err)
	}
	userID, err := svc.Validate(ctx, newRaw)
	if err != nil {
		t.Fatalf("new token Validate() error = %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("userID = %q, want user-1", userID)
	}
}

func TestRefreshTokenService_RotateRejectsAlreadyRevoked(t *testing.T) {
	svc := tokens.NewRefreshTokenService(memory.NewRefreshTokenRepository(), time.Hour)
	ctx := context.Background()

	raw, err := svc.Issue(ctx, "user-1")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := svc.Revoke(ctx, raw); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	if _, err := svc.Rotate(ctx, raw, "user-1"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Rotate() error = %v, want ErrUnauthorized", err)
	}
}

func TestRefreshTokenService_RotateRejectsWrongUser(t *testing.T) {
	svc := tokens.NewRefreshTokenService(memory.NewRefreshTokenRepository(), time.Hour)
	ctx := context.Background()

	raw, err := svc.Issue(ctx, "user-1")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := svc.Rotate(ctx, raw, "user-2"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Rotate() error = %v, want ErrUnauthorized", err)
	}

	userID, err := svc.Validate(ctx, raw)
	if err != nil {
		t.Fatalf("old token should stay valid: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("userID = %q, want user-1", userID)
	}
}
