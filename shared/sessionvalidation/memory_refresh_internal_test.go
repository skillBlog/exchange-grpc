package sessionvalidation

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	sharederrors "github.com/exchange-grpc/shared/errors"
)

func TestMemoryRefreshTokenStore_RotateDropsOldHashIndex(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	ctx := context.Background()
	now := time.Now()

	current := RefreshToken{
		ID:        "token-0",
		UserID:    "user-1",
		TokenHash: "hash-0",
		ExpiresAt: now.Add(time.Hour),
	}
	if err := store.Save(ctx, current); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	const rotations = 5
	for i := 1; i <= rotations; i++ {
		next := RefreshToken{
			ID:        fmt.Sprintf("token-%d", i),
			UserID:    "user-1",
			TokenHash: fmt.Sprintf("hash-%d", i),
			ExpiresAt: now.Add(time.Hour),
		}
		if err := store.Rotate(ctx, current.TokenHash, now, next); err != nil {
			t.Fatalf("Rotate(%d) error = %v", i, err)
		}
		current = next
	}

	if got := len(store.byHash); got != 1 {
		t.Fatalf("len(byHash) = %d, want 1", got)
	}
	if got := len(store.byID); got != rotations+1 {
		t.Fatalf("len(byID) = %d, want %d (revoked history kept)", got, rotations+1)
	}

	if _, err := store.GetByTokenHash(ctx, "hash-0"); !errors.Is(err, sharederrors.ErrUnauthorized) {
		t.Fatalf("old hash lookup error = %v, want ErrUnauthorized", err)
	}
	stored, err := store.GetByTokenHash(ctx, current.TokenHash)
	if err != nil {
		t.Fatalf("current hash lookup error = %v", err)
	}
	if stored.ID != current.ID {
		t.Fatalf("current id = %q, want %q", stored.ID, current.ID)
	}
}
