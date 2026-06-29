package memory_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/exchange-grpc/internal/adapter/repository/memory"
	"github.com/exchange-grpc/internal/domain"
)

func TestMarketRepository_ListActive(t *testing.T) {
	repo := memory.NewSeededMarketRepository()

	markets, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive() error = %v", err)
	}

	ids := make([]string, 0, len(markets))
	for _, market := range markets {
		if !market.IsActive() {
			t.Fatalf("ListActive returned inactive market %q", market.ID)
		}
		ids = append(ids, market.ID)
	}

	want := []string{"BTC-USDT", "ETH-USDT", "BNB-USDT"}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("active market ids = %v, want %v", ids, want)
	}
}

func TestMarketRepository_GetByID(t *testing.T) {
	repo := memory.NewSeededMarketRepository()

	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{name: "active market", id: "BTC-USDT"},
		{name: "disabled market", id: "SOL-USDT"},
		{name: "deleted market", id: "XRP-USDT"},
		{name: "missing market", id: "DOGE-USDT", wantErr: domain.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.GetByID(context.Background(), tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetByID() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}
		})
	}
}

func TestOrderRepository_saveAndGet(t *testing.T) {
	repo := memory.NewOrderRepository()
	ctx := context.Background()

	order, err := domain.NewOrder("order-1", "user-1", "BTC-USDT", domain.OrderTypeLimit, "100", "0.1")
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}

	if err := repo.Save(ctx, order); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repo.GetByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.UserID != order.UserID {
		t.Fatalf("UserID = %q, want %q", got.UserID, order.UserID)
	}
}

func TestOrderRepository_GetByIDAndUserID(t *testing.T) {
	repo := memory.NewOrderRepository()
	ctx := context.Background()

	order, err := domain.NewOrder("order-1", "user-1", "BTC-USDT", domain.OrderTypeMarket, "", "1")
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Save(ctx, order); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err = repo.GetByIDAndUserID(ctx, order.ID, "user-1")
	if err != nil {
		t.Fatalf("GetByIDAndUserID() error = %v", err)
	}

	_, err = repo.GetByIDAndUserID(ctx, order.ID, "user-2")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByIDAndUserID() error = %v, want ErrNotFound", err)
	}
}

func TestMarketRepository_concurrentAccess(t *testing.T) {
	repo := memory.NewSeededMarketRepository()
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = repo.ListActive(ctx)
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		_, _ = repo.GetByID(ctx, "BTC-USDT")
	}

	<-done
}
