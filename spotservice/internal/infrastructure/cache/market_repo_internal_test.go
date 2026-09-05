package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/exchange-grpc/spotservice/internal/domain"
	"github.com/exchange-grpc/spotservice/internal/infrastructure/memory"
)

type countingMarketRepo struct {
	inner   domain.MarketRepository
	calls   atomic.Int32
	blockID string
	started chan struct{}
	release chan struct{}
}

func (r *countingMarketRepo) GetByID(ctx context.Context, id string) (domain.Market, error) {
	r.calls.Add(1)
	if r.blockID != "" && id == r.blockID {
		if r.started != nil {
			select {
			case r.started <- struct{}{}:
			default:
			}
		}
		if r.release != nil {
			select {
			case <-r.release:
			case <-ctx.Done():
				return domain.Market{}, ctx.Err()
			}
		}
	}
	return r.inner.GetByID(ctx, id)
}

func (r *countingMarketRepo) ListActivePage(ctx context.Context, userRoles []string, limit int, afterID string) ([]domain.Market, error) {
	return r.inner.ListActivePage(ctx, userRoles, limit, afterID)
}

func (r *countingMarketRepo) Ping(ctx context.Context) error {
	return r.inner.Ping(ctx)
}

type delayClockRepo struct {
	inner   domain.MarketRepository
	advance func()
}

func (r delayClockRepo) GetByID(ctx context.Context, id string) (domain.Market, error) {
	market, err := r.inner.GetByID(ctx, id)
	if r.advance != nil {
		r.advance()
	}
	return market, err
}

func (r delayClockRepo) ListActivePage(ctx context.Context, userRoles []string, limit int, afterID string) ([]domain.Market, error) {
	return r.inner.ListActivePage(ctx, userRoles, limit, afterID)
}

func (r delayClockRepo) Ping(ctx context.Context) error {
	return r.inner.Ping(ctx)
}

func TestMarketRepository_singleflightCoalescesMisses(t *testing.T) {
	inner := &countingMarketRepo{inner: memory.NewSeededMarketRepository()}
	repo := NewMarketRepository(inner, time.Minute)

	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := repo.GetByID(context.Background(), "BTC-USDT"); err != nil {
				t.Errorf("GetByID() error = %v", err)
			}
		}()
	}
	wg.Wait()

	if got := inner.calls.Load(); got != 1 {
		t.Fatalf("inner GetByID calls = %d, want 1", got)
	}
}

func TestMarketRepository_ttlStartsAfterSuccessfulRead(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	inner := &countingMarketRepo{
		inner: delayClockRepo{
			inner: memory.NewSeededMarketRepository(),
			advance: func() {
				clock = clock.Add(5 * time.Second)
			},
		},
	}
	repo := NewMarketRepository(inner, 30*time.Second)
	repo.now = func() time.Time { return clock }

	if _, err := repo.GetByID(context.Background(), "BTC-USDT"); err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if inner.calls.Load() != 1 {
		t.Fatalf("calls after first read = %d, want 1", inner.calls.Load())
	}

	clock = time.Date(2026, 1, 1, 12, 0, 32, 0, time.UTC)
	if _, err := repo.GetByID(context.Background(), "BTC-USDT"); err != nil {
		t.Fatalf("GetByID() at T+32s error = %v", err)
	}
	if inner.calls.Load() != 1 {
		t.Fatalf("calls at T+32s = %d, want 1 (TTL started after DB read)", inner.calls.Load())
	}

	clock = time.Date(2026, 1, 1, 12, 0, 36, 0, time.UTC)
	if _, err := repo.GetByID(context.Background(), "BTC-USDT"); err != nil {
		t.Fatalf("GetByID() at T+36s error = %v", err)
	}
	if inner.calls.Load() != 2 {
		t.Fatalf("calls at T+36s = %d, want 2", inner.calls.Load())
	}
}

func TestMarketRepository_doesNotHoldLockDuringInnerFetch(t *testing.T) {
	inner := &countingMarketRepo{
		inner:   memory.NewSeededMarketRepository(),
		blockID: "BTC-USDT",
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	repo := NewMarketRepository(inner, time.Minute)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := repo.GetByID(context.Background(), "BTC-USDT"); err != nil {
			t.Errorf("blocked GetByID() error = %v", err)
		}
	}()

	select {
	case <-inner.started:
	case <-time.After(time.Second):
		t.Fatal("inner GetByID was not called")
	}

	ethDone := make(chan struct{})
	go func() {
		defer close(ethDone)
		if _, err := repo.GetByID(context.Background(), "ETH-USDT"); err != nil {
			t.Errorf("ETH GetByID() error = %v", err)
		}
	}()

	select {
	case <-ethDone:
	case <-time.After(time.Second):
		t.Fatal("GetByID(ETH-USDT) blocked while another fetch held the cache lock")
	}

	close(inner.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("blocked GetByID did not finish")
	}
}

func TestMarketRepository_evictsExpiredEntries(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := NewMarketRepository(memory.NewSeededMarketRepository(), 30*time.Second)
	repo.now = func() time.Time { return clock }

	if _, err := repo.GetByID(context.Background(), "BTC-USDT"); err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if repo.cachedCount() != 1 {
		t.Fatalf("cachedCount = %d, want 1", repo.cachedCount())
	}

	clock = clock.Add(31 * time.Second)
	if _, err := repo.GetByID(context.Background(), "BTC-USDT"); err != nil {
		t.Fatalf("GetByID() after TTL error = %v", err)
	}
	if repo.cachedCount() != 1 {
		t.Fatalf("cachedCount after refresh = %d, want 1 (expired key deleted, new entry stored)", repo.cachedCount())
	}
}

func TestMarketRepository_canceledCallerDoesNotWaitForInner(t *testing.T) {
	inner := &countingMarketRepo{
		inner:   memory.NewSeededMarketRepository(),
		blockID: "BTC-USDT",
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	repo := NewMarketRepository(inner, time.Minute)
	repo.fetchTimeout = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := repo.GetByID(ctx, "BTC-USDT")
		errCh <- err
	}()

	select {
	case <-inner.started:
	case <-time.After(time.Second):
		t.Fatal("inner GetByID was not called")
	}

	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled GetByID still waited for inner fetch")
	}

	close(inner.release)
}
