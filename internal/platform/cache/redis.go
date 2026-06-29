package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisResponseCache хранит ответы в Redis.
type RedisResponseCache struct {
	client *redis.Client
}

// NewRedisResponseCache создаёт кэш на базе Redis.
func NewRedisResponseCache(addr string) (*RedisResponseCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &RedisResponseCache{client: client}, nil
}

// Close закрывает клиент Redis.
func (c *RedisResponseCache) Close() error {
	return c.client.Close()
}

// Get возвращает значение из кэша по ключу.
func (c *RedisResponseCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

// Set сохраняет значение с TTL.
func (c *RedisResponseCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}
