package redisbus

import (
	"context"
	"fmt"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	orderkafka "github.com/exchange-grpc/orderservice/internal/infrastructure/kafka"
	"github.com/redis/go-redis/v9"
)

// OrderUpdatesChannel — канал Redis pub/sub с живыми обновлениями ордеров.
const OrderUpdatesChannel = "order.updates"

const defaultPublishTimeout = 5 * time.Second

// Publisher пишет конверт order.events в Redis. Подписчики каждой реплики кормят свой hub.
type Publisher struct {
	channel string
	publish func(ctx context.Context, channel string, payload []byte) error
}

// NewPublisher создаёт publisher. timeout <= 0 — 5s.
func NewPublisher(client *redis.Client, channel string, timeout time.Duration) *Publisher {
	if channel == "" {
		channel = OrderUpdatesChannel
	}
	if timeout <= 0 {
		timeout = defaultPublishTimeout
	}
	return &Publisher{
		channel: channel,
		publish: func(ctx context.Context, channel string, payload []byte) error {
			if client == nil {
				return fmt.Errorf("redis client is not configured")
			}
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return client.Publish(ctx, channel, payload).Err()
		},
	}
}

// Publish сериализует outbox-событие и публикует его в канал.
func (p *Publisher) Publish(ctx context.Context, event application.OutboxEvent) error {
	if p == nil || p.publish == nil {
		return fmt.Errorf("redis order update publisher is not configured")
	}
	body, err := orderkafka.MarshalEnvelope(event)
	if err != nil {
		return err
	}
	if err := p.publish(ctx, p.channel, body); err != nil {
		return fmt.Errorf("redis publish order update: %w", err)
	}
	return nil
}

var _ application.OrderEventPublisher = (*Publisher)(nil)
