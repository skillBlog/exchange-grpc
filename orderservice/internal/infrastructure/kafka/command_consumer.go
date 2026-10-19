package kafka

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/shared/tracing"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// DefaultCommandShutdownGrace — сколько текущая команда может дописаться после отмены Run.
// Новые FetchMessage после отмены не стартуют.
const DefaultCommandShutdownGrace = 5 * time.Second

// DefaultCommandGroupID шарится между репликами: каждая команда обрабатывается один раз.
const DefaultCommandGroupID = "orderservice-commands"

// CommandConsumer читает order.commands, применяет через inbox и при яде пишет DLQ.
type CommandConsumer struct {
	reader      *kafkago.Reader
	topic       string
	groupID     string
	handler     application.OrderCommandHandler
	dlq         application.DeadLetterSink
	maxAttempts int
	log         *zap.Logger
}

// NewCommandConsumer создаёт reader с shared consumer group.
// Пустой groupID → DefaultCommandGroupID (не уникальный на процесс, в отличие от G5 hub).
func NewCommandConsumer(
	brokers []string,
	topic, groupID string,
	handler application.OrderCommandHandler,
	dlq application.DeadLetterSink,
	maxAttempts int,
	log *zap.Logger,
) *CommandConsumer {
	if topic == "" {
		topic = "order.commands"
	}
	if groupID == "" {
		groupID = DefaultCommandGroupID
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &CommandConsumer{
		topic:       topic,
		groupID:     groupID,
		handler:     handler,
		dlq:         dlq,
		maxAttempts: maxAttempts,
		log:         log,
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:               brokers,
			Topic:                 topic,
			GroupID:               groupID,
			StartOffset:           kafkago.FirstOffset,
			MinBytes:              1,
			MaxBytes:              10 << 20,
			WatchPartitionChanges: true,
		}),
	}
}

// Run читает команды до отмены ctx.
// Offset коммитится после успешной обработки и после залогированной ошибки DLQ.
// До успешного результата offset не коммитится. При отмене новые fetch не стартуют,
// уже взятому сообщению даётся DefaultCommandShutdownGrace дописать обработку и commit.
func (c *CommandConsumer) Run(ctx context.Context) {
	if c == nil || c.reader == nil {
		return
	}
	defer func() { _ = c.reader.Close() }()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			c.log.Error("kafka command consumer fetch failed", zap.Error(err))
			continue
		}

		if err := c.finishMessage(ctx, msg); err != nil {
			return
		}
	}
}

// finishMessage обрабатывает уже полученное сообщение.
// Отмена ctx не обрывает его сразу: есть короткое окно на обработку и commit.
func (c *CommandConsumer) finishMessage(ctx context.Context, msg kafkago.Message) error {
	workCtx, stop := contextWithShutdownGrace(ctx, DefaultCommandShutdownGrace)
	defer stop()

	if err := c.process(workCtx, msg); err != nil {
		return err
	}
	if err := c.reader.CommitMessages(workCtx, msg); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		c.log.Error("kafka command consumer commit failed", zap.Error(err))
	}
	return nil
}

func (c *CommandConsumer) process(ctx context.Context, msg kafkago.Message) error {
	ctx = extractTrace(ctx, msg.Headers)
	ctx, span := tracing.Start(ctx, "kafka.consume order.commands",
		tracing.Attr("messaging.destination", msg.Topic),
	)
	var spanErr error
	defer tracing.End(span, &spanErr)

	for {
		if err := ctx.Err(); err != nil {
			spanErr = err
			return err
		}
		last := application.ProcessCommandMessage(ctx, msg.Value, c.handler, c.dlq, c.maxAttempts)
		if last == nil {
			spanErr = nil
			return nil
		}
		// Окно остановки истекло: offset не коммитим, доставка повторится.
		if ctx.Err() != nil {
			spanErr = last
			return last
		}
		if errors.Is(last, application.ErrDLQPublish) {
			spanErr = last
			c.log.Error("kafka command dlq publish failed, committing offset",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.Error(last),
			)
			return nil
		}
		if errors.Is(last, context.Canceled) || errors.Is(last, context.DeadlineExceeded) {
			spanErr = last
			return last
		}
		spanErr = last
		c.log.Error("kafka command processing failed, retrying same message",
			zap.String("topic", msg.Topic),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
			zap.Error(last),
		)
	}
}

// contextWithShutdownGrace отменяет результат через grace после отмены parent.
// Пока parent жив, работа не ограничена этим таймером. stop завершает её сразу.
func contextWithShutdownGrace(parent context.Context, grace time.Duration) (context.Context, func()) {
	if grace <= 0 {
		grace = DefaultCommandShutdownGrace
	}
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	stopped := make(chan struct{})
	go func() {
		select {
		case <-parent.Done():
			timer := time.NewTimer(grace)
			select {
			case <-timer.C:
				cancel()
			case <-stopped:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			}
		case <-stopped:
		}
	}()
	var once sync.Once
	return ctx, func() {
		once.Do(func() {
			close(stopped)
			cancel()
		})
	}
}

// Close закрывает reader.
func (c *CommandConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}
