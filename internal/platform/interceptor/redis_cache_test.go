package interceptor_test

import (
	"context"
	"testing"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/platform/cache"
	"github.com/exchange-grpc/internal/platform/interceptor"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
)

func TestUnaryServerRedisCache_hitOnSecondCall(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	log := zap.New(core)
	store := cache.NewMemoryResponseCache()

	cachedInterceptor := interceptor.UnaryServerRedisCache(store, time.Minute, log)
	calls := 0
	handler := func(ctx context.Context, req any) (any, error) {
		calls++
		return &exchangev1.ViewMarketsResponse{
			Markets: []*exchangev1.Market{{Id: "BTC-USDT", Enabled: true}},
		}, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: exchangev1.SpotInstrumentService_ViewMarkets_FullMethodName}
	req := &exchangev1.ViewMarketsRequest{UserRoles: []string{"trader"}}

	if _, err := cachedInterceptor(context.Background(), req, info, handler); err != nil {
		t.Fatalf("first call error = %v", err)
	}
	if _, err := cachedInterceptor(context.Background(), req, info, handler); err != nil {
		t.Fatalf("second call error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("handler calls = %d, want 1", calls)
	}

	var hits, misses int
	for _, entry := range recorded.All() {
		switch entry.Message {
		case "grpc cache hit":
			hits++
		case "grpc cache miss":
			misses++
		}
	}
	if misses != 1 || hits != 1 {
		t.Fatalf("log hits=%d misses=%d, want 1/1", hits, misses)
	}
}

func TestUnaryServerRedisCache_differentRolesDifferentKeys(t *testing.T) {
	store := cache.NewMemoryResponseCache()
	cachedInterceptor := interceptor.UnaryServerRedisCache(store, time.Minute, zap.NewNop())

	calls := 0
	handler := func(ctx context.Context, req any) (any, error) {
		calls++
		return &exchangev1.ViewMarketsResponse{}, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: exchangev1.SpotInstrumentService_ViewMarkets_FullMethodName}

	_, _ = cachedInterceptor(context.Background(), &exchangev1.ViewMarketsRequest{UserRoles: []string{"trader"}}, info, handler)
	_, _ = cachedInterceptor(context.Background(), &exchangev1.ViewMarketsRequest{UserRoles: []string{"admin"}}, info, handler)

	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2", calls)
	}
}

func TestUnaryServerRedisCache_skipsOtherMethods(t *testing.T) {
	store := cache.NewMemoryResponseCache()
	cachedInterceptor := interceptor.UnaryServerRedisCache(store, time.Minute, zap.NewNop())

	calls := 0
	handler := func(ctx context.Context, req any) (any, error) {
		calls++
		return &exchangev1.GetMarketResponse{}, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: exchangev1.SpotInstrumentService_GetMarket_FullMethodName}

	_, _ = cachedInterceptor(context.Background(), &exchangev1.GetMarketRequest{MarketId: "BTC-USDT"}, info, handler)
	_, _ = cachedInterceptor(context.Background(), &exchangev1.GetMarketRequest{MarketId: "BTC-USDT"}, info, handler)

	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2 (no cache for GetMarket)", calls)
	}
}
