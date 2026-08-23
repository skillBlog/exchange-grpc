package memory_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
)

func TestRefreshTokenRepository_RotateConcurrentOnlyOneSucceeds(t *testing.T) {
	repo := memory.NewRefreshTokenRepository()
	ctx := context.Background()
	now := time.Now()
	old := domain.RefreshToken{
		ID:        "old",
		UserID:    "user-1",
		TokenHash: "hash-old",
		ExpiresAt: now.Add(time.Hour),
	}
	if err := repo.Save(ctx, old); err != nil {
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
			next := domain.RefreshToken{
				ID:        fmt.Sprintf("new-%d", i),
				UserID:    "user-1",
				TokenHash: fmt.Sprintf("hash-new-%d", i),
				ExpiresAt: now.Add(time.Hour),
			}
			if err := repo.Rotate(ctx, "hash-old", now, next); err == nil {
				success.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if got := success.Load(); got != 1 {
		t.Fatalf("successful rotates = %d, want 1", got)
	}

	stored, err := repo.GetByTokenHash(ctx, "hash-old")
	if err != nil {
		t.Fatalf("GetByTokenHash(old) error = %v", err)
	}
	if stored.IsActive(now) {
		t.Fatal("old token must be revoked after rotate")
	}
}
