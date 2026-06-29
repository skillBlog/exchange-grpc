package interceptor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"time"

	exchangev1 "github.com/exchange-grpc/api/proto/exchange/v1"
	"github.com/exchange-grpc/internal/platform/cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var (
	cacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "grpc_cache_hits_total",
		Help: "Общее число попаданий в gRPC-кэш",
	})
	cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "grpc_cache_misses_total",
		Help: "Общее число промахов gRPC-кэша",
	})
)

// UnaryServerRedisCache кэширует ответы ViewMarkets с ключом по ролям пользователя.
func UnaryServerRedisCache(store cache.ResponseCache, ttl time.Duration, log *zap.Logger) grpc.UnaryServerInterceptor {
	if log == nil {
		log = zap.NewNop()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if store == nil || info.FullMethod != exchangev1.SpotInstrumentService_ViewMarkets_FullMethodName {
			return handler(ctx, req)
		}

		viewReq, ok := req.(*exchangev1.ViewMarketsRequest)
		if !ok {
			return handler(ctx, req)
		}

		key := viewMarketsCacheKey(viewReq.GetUserRoles())
		if payload, found, err := store.Get(ctx, key); err == nil && found {
			resp := &exchangev1.ViewMarketsResponse{}
			if unmarshalErr := proto.Unmarshal(payload, resp); unmarshalErr == nil {
				cacheHits.Inc()
				log.Info("grpc cache hit",
					zap.String("method", info.FullMethod),
					zap.String("cache_key", key),
					zap.String("request_id", RequestIDFromContext(ctx)),
				)
				return resp, nil
			}
		} else if err != nil {
			log.Warn("grpc cache get failed",
				zap.String("method", info.FullMethod),
				zap.String("cache_key", key),
				zap.Error(err),
			)
		}

		cacheMisses.Inc()
		log.Info("grpc cache miss",
			zap.String("method", info.FullMethod),
			zap.String("cache_key", key),
			zap.String("request_id", RequestIDFromContext(ctx)),
		)

		resp, err := handler(ctx, req)
		if err != nil {
			return nil, err
		}

		viewResp, ok := resp.(*exchangev1.ViewMarketsResponse)
		if !ok {
			return resp, nil
		}

		payload, marshalErr := proto.Marshal(viewResp)
		if marshalErr != nil {
			return resp, nil
		}
		if setErr := store.Set(ctx, key, payload, ttl); setErr != nil {
			log.Warn("grpc cache set failed",
				zap.String("method", info.FullMethod),
				zap.String("cache_key", key),
				zap.Error(setErr),
			)
		}

		return resp, nil
	}
}

func viewMarketsCacheKey(userRoles []string) string {
	roles := append([]string(nil), userRoles...)
	slices.Sort(roles)
	sum := sha256.Sum256([]byte(strings.Join(roles, ",")))
	return "view_markets:" + hex.EncodeToString(sum[:])
}
