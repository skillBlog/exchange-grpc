package application

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

const (
	DefaultOutboxRelayInterval  = 100 * time.Millisecond
	DefaultOutboxRelayBatchSize = 100
)

// OutboxRelay забирает unprocessed события.
// С Kafka: только брокер, затем processed_at; hub кормит consumer (G5).
// Без Kafka: сразу hub (тесты и локальный режим).
type OutboxRelay struct {
	tx        TxManager
	queue     OutboxRelayStore
	notifier  OrderNotifier
	publisher OrderEventPublisher
	log       *zap.Logger
	interval  time.Duration
	batchSize int
}

// NewOutboxRelay создаёт relay. interval/batchSize <= 0 — дефолты. publisher может быть nil.
func NewOutboxRelay(
	tx TxManager,
	queue OutboxRelayStore,
	notifier OrderNotifier,
	publisher OrderEventPublisher,
	log *zap.Logger,
	interval time.Duration,
	batchSize int,
) *OutboxRelay {
	if log == nil {
		log = zap.NewNop()
	}
	if tx == nil {
		tx = nopTxManager{}
	}
	if interval <= 0 {
		interval = DefaultOutboxRelayInterval
	}
	if batchSize <= 0 {
		batchSize = DefaultOutboxRelayBatchSize
	}
	return &OutboxRelay{
		tx:        tx,
		queue:     queue,
		notifier:  notifier,
		publisher: publisher,
		log:       log,
		interval:  interval,
		batchSize: batchSize,
	}
}

// Run крутит Drain до отмены ctx.
func (r *OutboxRelay) Run(ctx context.Context) {
	if r == nil || r.queue == nil {
		return
	}
	if r.publisher == nil && r.notifier == nil {
		return
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		if err := r.Drain(ctx); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			r.log.Error("outbox relay drain failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Drain забирает пачки, пока есть успешно обработанные события.
// Битый payload или ошибка Kafka: processed=0, ждём следующий тик (без busy-loop).
func (r *OutboxRelay) Drain(ctx context.Context) error {
	for {
		processed, claimed, err := r.RunOnce(ctx)
		if err != nil {
			return err
		}
		if claimed == 0 || processed == 0 {
			return nil
		}
		if claimed < r.batchSize {
			return nil
		}
	}
}

// RunOnce берёт одну пачку под tx: Kafka или hub, затем processed_at.
func (r *OutboxRelay) RunOnce(ctx context.Context) (processed, claimed int, err error) {
	if r == nil || r.queue == nil {
		return 0, 0, nil
	}
	if r.publisher == nil && r.notifier == nil {
		return 0, 0, nil
	}

	err = r.tx.WithinTx(ctx, func(ctx context.Context) error {
		events, claimErr := r.queue.ClaimBatch(ctx, r.batchSize)
		if claimErr != nil {
			return claimErr
		}
		claimed = len(events)
		if claimed == 0 {
			return nil
		}

		done := make([]string, 0, claimed)
		for _, event := range events {
			update, parseErr := ParseOutboxUpdateEvent(event)
			if parseErr != nil {
				if recErr := r.queue.RecordAttempt(ctx, event.ID); recErr != nil {
					return recErr
				}
				r.log.Error("outbox relay skip invalid payload",
					zap.String("event_id", event.ID),
					zap.String("aggregate_id", event.AggregateID),
					zap.Error(parseErr),
				)
				continue
			}
			if r.publisher != nil {
				if pubErr := r.publisher.Publish(ctx, event); pubErr != nil {
					if recErr := r.queue.RecordAttempt(ctx, event.ID); recErr != nil {
						return recErr
					}
					r.log.Error("outbox relay kafka publish failed",
						zap.String("event_id", event.ID),
						zap.String("aggregate_id", event.AggregateID),
						zap.Error(pubErr),
					)
					continue
				}
			} else if r.notifier != nil {
				r.notifier.Publish(update)
			}
			done = append(done, event.ID)
		}
		processed = len(done)
		if processed == 0 {
			return nil
		}
		return r.queue.MarkProcessed(ctx, done)
	})
	return processed, claimed, err
}
