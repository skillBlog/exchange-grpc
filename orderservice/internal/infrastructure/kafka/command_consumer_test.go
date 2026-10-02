package kafka

import (
	"strings"
	"testing"

	"github.com/exchange-grpc/orderservice/internal/application"
)

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
