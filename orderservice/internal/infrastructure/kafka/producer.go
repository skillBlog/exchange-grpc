package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	kafkago "github.com/segmentio/kafka-go"
)

const defaultProduceTimeout = 5 * time.Second

// Event — контракт сообщения в order.events (event_id для идемпотентных консьюмеров, G5).
type Event struct {
	EventID     string          `json:"event_id"`
	EventType   string          `json:"event_type"`
	AggregateID string          `json:"aggregate_id"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Producer пишет outbox-события в Kafka. Key = aggregate_id (order_id), чтобы события одного ордера шли в одну партицию.
type Producer struct {
	writer  *kafkago.Writer
	timeout time.Duration
}

// NewProducer создаёт writer. brokers не должны быть пустыми.
func NewProducer(brokers []string, topic string, timeout time.Duration) *Producer {
	if timeout <= 0 {
		timeout = defaultProduceTimeout
	}
	if topic == "" {
		topic = "order.events"
	}
	return &Producer{
		timeout: timeout,
		writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafkago.Hash{},
			RequiredAcks:           kafkago.RequireAll,
			AllowAutoTopicCreation: true,
			BatchTimeout:           10 * time.Millisecond,
		},
	}
}

// Publish сериализует outbox-событие и пишет в Kafka.
func (p *Producer) Publish(ctx context.Context, event application.OutboxEvent) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka producer is not configured")
	}
	msg, err := messageFromEvent(event)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}
	return nil
}

// Ping проверяет, что брокер отвечает (optional health).
func (p *Producer) Ping(ctx context.Context) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka producer is not configured")
	}
	addr := p.writer.Addr.String()
	if addr == "" {
		return fmt.Errorf("kafka brokers are empty")
	}
	d := net.Dialer{}
	host := firstHost(addr)
	conn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return fmt.Errorf("kafka dial: %w", err)
	}
	return conn.Close()
}

// Close закрывает writer.
func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

func messageFromEvent(event application.OutboxEvent) (kafkago.Message, error) {
	body, err := json.Marshal(Event{
		EventID:     event.ID,
		EventType:   event.EventType,
		AggregateID: event.AggregateID,
		Payload:     event.Payload,
		CreatedAt:   event.CreatedAt,
	})
	if err != nil {
		return kafkago.Message{}, fmt.Errorf("marshal kafka event: %w", err)
	}
	return kafkago.Message{
		Key:   []byte(event.AggregateID),
		Value: body,
		Headers: []kafkago.Header{
			{Key: "event_id", Value: []byte(event.ID)},
			{Key: "event_type", Value: []byte(event.EventType)},
		},
	}, nil
}

func firstHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil && host != "" {
		return addr
	}
	// Addr.String() у kafka.TCP может быть "host:9092,host2:9092".
	for i := 0; i < len(addr); i++ {
		if addr[i] == ',' {
			return addr[:i]
		}
	}
	return addr
}

var _ application.OrderEventPublisher = (*Producer)(nil)
