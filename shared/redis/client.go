package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Client оборачивает go-redis клиент.
type Client struct {
	rdb *redis.Client
}

// Option настраивает redis.Options после ParseURL.
type Option func(*redis.Options)

// WithPoolSize задаёт размер connection pool.
func WithPoolSize(size int) Option {
	return func(o *redis.Options) {
		if size > 0 {
			o.PoolSize = size
		}
	}
}

// WithMinIdleConns задаёт минимальное число idle-соединений.
func WithMinIdleConns(n int) Option {
	return func(o *redis.Options) {
		if n > 0 {
			o.MinIdleConns = n
		}
	}
}

// WithMaxRetries задаёт число повторов команд при временных сбоях.
func WithMaxRetries(n int) Option {
	return func(o *redis.Options) {
		if n >= 0 {
			o.MaxRetries = n
		}
	}
}

// Connect открывает соединение с Redis.
func Connect(ctx context.Context, url string, opts ...Option) (*Client, error) {
	parsed, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	for _, opt := range opts {
		if opt != nil {
			opt(parsed)
		}
	}

	rdb := redis.NewClient(parsed)
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// Raw возвращает низкоуровневый клиент.
func (c *Client) Raw() *redis.Client {
	return c.rdb
}

// Ping проверяет доступность Redis.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return c.rdb.Ping(ctx).Err()
}

// Close закрывает соединение.
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}
