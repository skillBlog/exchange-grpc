package kafka

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type stubCommandHandler struct {
	err   error
	calls int
}

func (h *stubCommandHandler) Handle(ctx context.Context, _ application.OrderCommand) error {
	h.calls++
	if h.err != nil {
		return h.err
	}
	return ctx.Err()
}

type stubDLQ struct {
	err   error
	calls int
}

func (d *stubDLQ) Publish(context.Context, []byte, string) error {
	d.calls++
	return d.err
}

func TestNewCommandConsumer_usesSharedGroup(t *testing.T) {
	c := NewCommandConsumer([]string{"localhost:9092"}, "", "", nil, nil, 0, nil)
	if c == nil || c.reader == nil {
		t.Fatal("expected consumer")
	}
	t.Cleanup(func() { _ = c.Close() })
	if c.groupID != DefaultCommandGroupID {
		t.Fatalf("groupID = %q, want shared %q", c.groupID, DefaultCommandGroupID)
	}
	if strings.HasPrefix(c.groupID, "orderservice-hub-") {
		t.Fatal("command group must not be unique per process")
	}
	if c.topic != "order.commands" {
		t.Fatalf("topic = %q", c.topic)
	}
}

func TestDeadLetterPublisher_implementsSink(t *testing.T) {
	var _ application.DeadLetterSink = (*DeadLetterPublisher)(nil)
}

func TestProcess_dlqFailureReturnsNilSoOffsetCanCommit(t *testing.T) {
	dlq := &stubDLQ{err: errors.New("dlq down")}
	c := &CommandConsumer{
		topic:       "order.commands",
		handler:     &stubCommandHandler{},
		dlq:         dlq,
		maxAttempts: 1,
		log:         zap.NewNop(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := c.process(ctx, kafkago.Message{
		Topic:     "order.commands",
		Partition: 1,
		Offset:    7,
		Value:     []byte(`{`),
	})
	if err != nil {
		t.Fatalf("process() error = %v, want nil", err)
	}
	if dlq.calls != 1 {
		t.Fatalf("dlq calls = %d, want 1", dlq.calls)
	}
}

func TestProcess_dlqTimeoutStillReturnsNil(t *testing.T) {
	c := &CommandConsumer{
		topic:       "order.commands",
		handler:     &stubCommandHandler{},
		dlq:         &stubDLQ{err: context.DeadlineExceeded},
		maxAttempts: 1,
		log:         zap.NewNop(),
	}

	err := c.process(context.Background(), kafkago.Message{Value: []byte(`{`)})
	if err != nil {
		t.Fatalf("process() error = %v, want nil when only the DLQ write timed out", err)
	}
}

func TestProcess_canceledWorkDoesNotReturnNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &CommandConsumer{
		topic:       "order.commands",
		handler:     &stubCommandHandler{},
		dlq:         &stubDLQ{},
		maxAttempts: 1,
		log:         zap.NewNop(),
	}

	err := c.process(ctx, kafkago.Message{Value: []byte(`{`)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("process() error = %v, want context.Canceled", err)
	}
}

func TestContextWithShutdownGrace_outlivesParent(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	work, stop := contextWithShutdownGrace(parent, 80*time.Millisecond)
	defer stop()

	cancelParent()
	select {
	case <-work.Done():
		t.Fatal("work context ended together with parent")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case <-work.Done():
	case <-time.After(time.Second):
		t.Fatal("work context did not end after grace")
	}
}

func TestContextWithShutdownGrace_staysOpenWhileParentIsActive(t *testing.T) {
	work, stop := contextWithShutdownGrace(context.Background(), time.Second)
	select {
	case <-work.Done():
		t.Fatal("work context ended while parent is active")
	case <-time.After(20 * time.Millisecond):
	}
	stop()
	select {
	case <-work.Done():
	case <-time.After(time.Second):
		t.Fatal("stop did not cancel work context")
	}
}
