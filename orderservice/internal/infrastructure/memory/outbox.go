package memory

import (
	"context"
	"sync"

	"github.com/exchange-grpc/orderservice/internal/application"
)

type outboxRow struct {
	event     application.OutboxEvent
	processed bool
	claimed   bool
	attempts  int
}

// OutboxStore хранит outbox-события в памяти для тестов.
type OutboxStore struct {
	mu     sync.Mutex
	events []outboxRow
}

// NewOutboxStore создаёт пустой in-memory outbox.
func NewOutboxStore() *OutboxStore {
	return &OutboxStore{}
}

// Append добавляет событие.
func (s *OutboxStore) Append(_ context.Context, event application.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, outboxRow{event: event})
	return nil
}

// Events возвращает копию записанных событий (включая processed).
func (s *OutboxStore) Events() []application.OutboxEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]application.OutboxEvent, len(s.events))
	for i, row := range s.events {
		out[i] = row.event
	}
	return out
}

// ClaimBatch помечает до limit unprocessed строк как claimed (аналог SKIP LOCKED).
func (s *OutboxStore) ClaimBatch(_ context.Context, limit int) ([]application.OutboxEvent, error) {
	if limit <= 0 {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]application.OutboxEvent, 0, limit)
	for i := range s.events {
		row := &s.events[i]
		if row.processed || row.claimed {
			continue
		}
		row.claimed = true
		out = append(out, row.event)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

// MarkProcessed помечает события обработанными и снимает claim.
func (s *OutboxStore) MarkProcessed(_ context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.events {
		if _, ok := want[s.events[i].event.ID]; !ok {
			continue
		}
		s.events[i].processed = true
		s.events[i].claimed = false
	}
	return nil
}

// RecordAttempt увеличивает attempts и снимает claim, чтобы событие можно было взять снова.
func (s *OutboxStore) RecordAttempt(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.events {
		if s.events[i].event.ID != id {
			continue
		}
		s.events[i].attempts++
		s.events[i].claimed = false
		return nil
	}
	return nil
}

var (
	_ application.OutboxStore      = (*OutboxStore)(nil)
	_ application.OutboxRelayStore = (*OutboxStore)(nil)
)
