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

// Connect открывает соединение с Redis.
func Connect(ctx context.Context, url string) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)
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
