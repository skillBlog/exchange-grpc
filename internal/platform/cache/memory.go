package cache

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	value     []byte
	expiresAt time.Time
}

// MemoryResponseCache — in-memory реализация ResponseCache для тестов.
type MemoryResponseCache struct {
	mu      sync.RWMutex
	entries map[string]memoryEntry
}

// NewMemoryResponseCache создаёт пустой in-memory кэш.
func NewMemoryResponseCache() *MemoryResponseCache {
	return &MemoryResponseCache{entries: make(map[string]memoryEntry)}
}

// Get возвращает значение из кэша, если оно есть и не истекло.
func (c *MemoryResponseCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false, nil
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, false, nil
	}

	return append([]byte(nil), entry.value...), true, nil
}

// Set сохраняет значение с TTL.
func (c *MemoryResponseCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := memoryEntry{value: append([]byte(nil), value...)}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	c.entries[key] = entry
	return nil
}
