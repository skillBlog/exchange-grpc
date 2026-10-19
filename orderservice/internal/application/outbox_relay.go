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
	// DefaultOutboxClaimTTL — сколько claim считается живым.
	// Дольше одного Produce (5s), чтобы живую отправку не отобрали, и достаточно коротко,
	// чтобы упавший процесс не оставил строку в processing навсегда.
	DefaultOutboxClaimTTL = 30 * time.Second
)

// OutboxRelay забирает unprocessed события.
// С Kafka: claim коммитится до Produce, затем отдельные транзакции на успех и ошибку.
// Без Kafka: hub в одной транзакции, сетевого Produce нет.
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

// RunOnce берёт одну пачку.
// Kafka: claim, Publish и отметка результата — разные транзакции.
// Локальный hub: одна транзакция, как раньше.
func (r *OutboxRelay) RunOnce(ctx context.Context) (processed, claimed int, err error) {
	if r == nil || r.queue == nil {
		return 0, 0, nil
	}
	if r.publisher == nil && r.notifier == nil {
		return 0, 0, nil
	}
	if r.publisher == nil {
		return r.runLocal(ctx)
	}
	return r.runBroker(ctx)
}

// runLocal пишет в hub внутри одной транзакции. Produce здесь нет.
func (r *OutboxRelay) runLocal(ctx context.Context) (processed, claimed int, err error) {
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
			if r.notifier != nil {
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

// runBroker коммитит claim до Produce. Успех и ошибка фиксируются отдельными транзакциями.
func (r *OutboxRelay) runBroker(ctx context.Context) (processed, claimed int, err error) {
	if err = ctx.Err(); err != nil {
		return 0, 0, err
	}

	staleBefore := time.Now().Add(-DefaultOutboxClaimTTL)
	if err = r.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, relErr := r.queue.ReleaseStaleClaims(ctx, staleBefore)
		return relErr
	}); err != nil {
		return 0, 0, err
	}

	var events []OutboxEvent
	if err = r.tx.WithinTx(ctx, func(ctx context.Context) error {
		var claimErr error
		events, claimErr = r.queue.ClaimBatch(ctx, r.batchSize)
		return claimErr
	}); err != nil {
		return 0, 0, err
	}
	claimed = len(events)
	if claimed == 0 {
		return 0, 0, nil
	}

	done := make([]string, 0, claimed)
	failed := make([]string, 0)
	var stop error
	for _, event := range events {
		if stop = ctx.Err(); stop != nil {
			break
		}
		if _, parseErr := ParseOutboxUpdateEvent(event); parseErr != nil {
			failed = append(failed, event.ID)
			r.log.Error("outbox relay skip invalid payload",
				zap.String("event_id", event.ID),
				zap.String("aggregate_id", event.AggregateID),
				zap.Error(parseErr),
			)
			continue
		}
		if pubErr := r.publisher.Publish(ctx, event); pubErr != nil {
			failed = append(failed, event.ID)
			r.log.Error("outbox relay kafka publish failed",
				zap.String("event_id", event.ID),
				zap.String("aggregate_id", event.AggregateID),
				zap.Error(pubErr),
			)
			continue
		}
		done = append(done, event.ID)
	}

	// Отметка уже отправленных событий не должна отмениться вместе с остановкой relay.
	finishCtx := context.WithoutCancel(ctx)
	if err = r.markProcessed(finishCtx, done); err != nil {
		return 0, claimed, err
	}
	processed = len(done)
	if err = r.recordAttempts(finishCtx, failed); err != nil {
		return processed, claimed, err
	}
	if stop != nil {
		return processed, claimed, stop
	}
	return processed, claimed, nil
}

func (r *OutboxRelay) markProcessed(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.tx.WithinTx(ctx, func(ctx context.Context) error {
		return r.queue.MarkProcessed(ctx, ids)
	})
}

func (r *OutboxRelay) recordAttempts(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.tx.WithinTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if err := r.queue.RecordAttempt(ctx, id); err != nil {
				return err
			}
		}
		return nil
	})
}
