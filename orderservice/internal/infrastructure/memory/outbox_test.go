package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
)

func TestRecordAttempt_canceledContextDoesNotChangeState(t *testing.T) {
	store := NewOutboxStore()
	event := application.OutboxEvent{ID: "event-1", AggregateID: "order-1"}
	if err := store.Append(context.Background(), event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if _, err := store.ClaimBatch(context.Background(), 1); err != nil {
		t.Fatalf("ClaimBatch() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := store.RecordAttempt(ctx, event.ID)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RecordAttempt() error = %v, want context.Canceled", err)
	}
	if store.events[0].attempts != 0 || !store.events[0].claimed {
		t.Fatalf("row = %+v, want attempts 0 and still claimed", store.events[0])
	}

	if err := store.RecordAttempt(context.Background(), event.ID); err != nil {
		t.Fatalf("RecordAttempt() error = %v", err)
	}
	if store.events[0].attempts != 1 || store.events[0].claimed {
		t.Fatalf("row = %+v, want attempts 1 and claim released", store.events[0])
	}
}

func TestClaimBatch_secondCallerDoesNotTakeSameRow(t *testing.T) {
	store := NewOutboxStore()
	const n = 20
	for i := 0; i < n; i++ {
		event := application.OutboxEvent{ID: fmt.Sprintf("event-%d", i), AggregateID: "order"}
		if err := store.Append(context.Background(), event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := make(map[string]int, n)
	start := make(chan struct{})
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := store.ClaimBatch(context.Background(), n)
			if err != nil {
				t.Errorf("ClaimBatch() error = %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, event := range got {
				seen[event.ID]++
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(seen) != n {
		t.Fatalf("unique claimed = %d, want %d", len(seen), n)
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("event %s claimed %d times", id, count)
		}
	}
}

func TestReleaseStaleClaims_returnsExpiredRowOnly(t *testing.T) {
	store := NewOutboxStore()
	expired := application.OutboxEvent{ID: "expired", AggregateID: "order-1"}
	live := application.OutboxEvent{ID: "live", AggregateID: "order-2"}
	for _, event := range []application.OutboxEvent{expired, live} {
		if err := store.Append(context.Background(), event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}
	if _, err := store.ClaimBatch(context.Background(), 10); err != nil {
		t.Fatalf("ClaimBatch() error = %v", err)
	}
	store.ExpireClaim(expired.ID, time.Now().Add(-time.Hour))

	released, err := store.ReleaseStaleClaims(context.Background(), time.Now().Add(-30*time.Second))
	if err != nil {
		t.Fatalf("ReleaseStaleClaims() error = %v", err)
	}
	if released != 1 {
		t.Fatalf("released = %d, want 1", released)
	}

	got, err := store.ClaimBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("ClaimBatch() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != expired.ID {
		t.Fatalf("reclaimed = %+v, want only %s", got, expired.ID)
	}
}
