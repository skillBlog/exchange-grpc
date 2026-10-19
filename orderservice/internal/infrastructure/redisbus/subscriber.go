package redisbus

import (
	"context"
	"errors"
	"fmt"
	"time"

	orderkafka "github.com/exchange-grpc/orderservice/internal/infrastructure/kafka"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var errRedisNotConfigured = fmt.Errorf("redis order updates are not configured")

const subscribeRetryInterval = time.Second

// Subscriber читает order.updates и публикует события в локальный hub.
type Subscriber struct {
	log   *zap.Logger
	retry time.Duration
	serve func(ctx context.Context) error
}

// NewSubscriber создаёт подписчика. forwarder дедуплицирует event_id внутри процесса.
func NewSubscriber(client *redis.Client, channel string, forward *orderkafka.HubForwarder, log *zap.Logger) *Subscriber {
	if channel == "" {
		channel = OrderUpdatesChannel
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Subscriber{
		log:   log,
		retry: subscribeRetryInterval,
		serve: func(ctx context.Context) error {
			return serveRedis(ctx, client, channel, forward, log)
		},
	}
}

// Run держит подписку до отмены ctx. Обрыв соединения поднимает её снова.
func (s *Subscriber) Run(ctx context.Context) {
	if s == nil || s.serve == nil {
		return
	}
	for {
		if ctx.Err() != nil {
			return
		}
		err := s.serve(ctx)
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		s.log.Error("redis order updates subscription failed", zap.Error(err))
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.retry):
		}
	}
}

func serveRedis(ctx context.Context, client *redis.Client, channel string, forward *orderkafka.HubForwarder, log *zap.Logger) error {
	if client == nil {
		return errRedisNotConfigured
	}
	if forward == nil {
		return errRedisNotConfigured
	}
	pubsub := client.Subscribe(ctx, channel)
	defer pubsub.Close()
	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}
	return readMessages(ctx, func(ctx context.Context) (string, error) {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			return "", err
		}
		return msg.Payload, nil
	}, forward.Deliver, log)
}

// readMessages отдаёт payload в deliver. Ошибка deliver логируется и не рвёт подписку:
// pub/sub не повторяет сообщение, клиент добирает состояние из БД.
func readMessages(ctx context.Context, next func(context.Context) (string, error), deliver func([]byte) error, log *zap.Logger) error {
	if log == nil {
		log = zap.NewNop()
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		payload, err := next(ctx)
		if err != nil {
			return err
		}
		if err := deliver([]byte(payload)); err != nil {
			log.Error("redis order update was not applied", zap.Error(err))
		}
	}
}
