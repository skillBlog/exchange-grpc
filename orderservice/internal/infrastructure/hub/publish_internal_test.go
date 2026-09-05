package hub

import (
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestPublishOne_closedChannelDoesNotStarveNext(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	h := NewUpdateHub(4, zap.New(core), time.Millisecond)

	closed := make(chan application.UpdateEvent)
	close(closed)
	live := make(chan application.UpdateEvent, 1)
	event := application.UpdateEvent{OrderID: "order-1", Status: domain.OrderStatusFilled, UpdatedAt: time.Now().UTC()}

	h.publishOne(closed, event)
	h.publishOne(live, event)

	select {
	case got := <-live:
		if got.Status != domain.OrderStatusFilled {
			t.Fatalf("Status = %q, want filled", got.Status)
		}
	default:
		t.Fatal("live subscriber did not receive event after closed channel panic")
	}

	if logs.FilterMessage("order update publish panic recovered").Len() != 1 {
		t.Fatalf("recover logs = %d, want 1", logs.FilterMessage("order update publish panic recovered").Len())
	}
}
