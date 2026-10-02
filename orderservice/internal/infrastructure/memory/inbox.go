package memory

import (
	"context"
	"sync"

	"github.com/exchange-grpc/orderservice/internal/application"
)

// InboxStore хранит принятые command_id в памяти для тестов.
// Транзакций нет: при ошибке после InsertIfNew запись останется (в отличие от postgres).
type InboxStore struct {
	mu  sync.Mutex
	ids map[string]struct{}
}

// NewInboxStore создаёт пустой in-memory inbox.
func NewInboxStore() *InboxStore {
	return &InboxStore{ids: make(map[string]struct{})}
}

// InsertIfNew помечает event_id принятым. Дубль — inserted=false.
func (s *InboxStore) InsertIfNew(_ context.Context, eventID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ids == nil {
		s.ids = make(map[string]struct{})
	}
	if _, ok := s.ids[eventID]; ok {
		return false, nil
	}
	s.ids[eventID] = struct{}{}
	return true, nil
}

var _ application.InboxStore = (*InboxStore)(nil)
