package cache

import (
	"context"
	"time"
)

// ResponseCache хранит сериализованные полезные нагрузки gRPC-ответов.
type ResponseCache interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
