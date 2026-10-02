package sessionvalidation_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sharederrors "github.com/exchange-grpc/shared/errors"
	"github.com/exchange-grpc/shared/sessionvalidation"
)

var refreshTokenContract = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestRefreshTokenService_IssueMatchesProtoCharset(t *testing.T) {
	svc := sessionvalidation.NewRefreshTokenService(sessionvalidation.NewMemoryRefreshTokenStore(), time.Hour)
	raw, err := svc.Issue(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if !refreshTokenContract.MatchString(raw) {
		t.Fatalf("issued refresh token %q does not match proto charset", raw)
	}
	if n := len(raw); n < 1 || n > 128 {
		t.Fatalf("issued refresh token length = %d, want 1..128", n)
	}
}

func TestRefreshTokenService_RotateRevokesOld(t *testing.T) {
	svc := sessionvalidation.NewRefreshTokenService(sessionvalidation.NewMemoryRefreshTokenStore(), time.Hour)
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

	if _, err := svc.Validate(ctx, oldRaw); !errors.Is(err, sharederrors.ErrUnauthorized) {
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
	svc := sessionvalidation.NewRefreshTokenService(sessionvalidation.NewMemoryRefreshTokenStore(), time.Hour)
	ctx := context.Background()

	raw, err := svc.Issue(ctx, "user-1")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := svc.Revoke(ctx, raw); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	if _, err := svc.Rotate(ctx, raw, "user-1"); !errors.Is(err, sharederrors.ErrUnauthorized) {
		t.Fatalf("Rotate() error = %v, want ErrUnauthorized", err)
	}
}

func TestRefreshTokenService_RotateRejectsWrongUser(t *testing.T) {
	svc := sessionvalidation.NewRefreshTokenService(sessionvalidation.NewMemoryRefreshTokenStore(), time.Hour)
	ctx := context.Background()

	raw, err := svc.Issue(ctx, "user-1")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := svc.Rotate(ctx, raw, "user-2"); !errors.Is(err, sharederrors.ErrUnauthorized) {
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

func TestMemoryRefreshTokenStore_RotateConcurrentOnlyOneSucceeds(t *testing.T) {
	store := sessionvalidation.NewMemoryRefreshTokenStore()
	ctx := context.Background()
	now := time.Now()
	old := sessionvalidation.RefreshToken{
		ID:        "old",
		UserID:    "user-1",
		TokenHash: "hash-old",
		ExpiresAt: now.Add(time.Hour),
	}
	if err := store.Save(ctx, old); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	const n = 8
	var (
		wg      sync.WaitGroup
		success atomic.Int32
	)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			next := sessionvalidation.RefreshToken{
				ID:        fmt.Sprintf("new-%d", i),
				UserID:    "user-1",
				TokenHash: fmt.Sprintf("hash-new-%d", i),
				ExpiresAt: now.Add(time.Hour),
			}
			if err := store.Rotate(ctx, "hash-old", now, next); err == nil {
				success.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if got := success.Load(); got != 1 {
		t.Fatalf("successful rotates = %d, want 1", got)
	}

	if _, err := store.GetByTokenHash(ctx, "hash-old"); !errors.Is(err, sharederrors.ErrUnauthorized) {
		t.Fatalf("GetByTokenHash(old) error = %v, want ErrUnauthorized", err)
	}

	active := 0
	for i := 0; i < n; i++ {
		stored, err := store.GetByTokenHash(ctx, fmt.Sprintf("hash-new-%d", i))
		if errors.Is(err, sharederrors.ErrUnauthorized) {
			continue
		}
		if err != nil {
			t.Fatalf("GetByTokenHash(new-%d) error = %v", i, err)
		}
		if stored.IsActive(now) {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("active rotated tokens = %d, want 1", active)
	}
}
