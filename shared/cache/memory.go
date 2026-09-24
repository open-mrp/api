package cache

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"
)

const defaultMemoryMaxEntries = 10_000

// MemoryStoreConfig configures an in-process MemoryStore.
type MemoryStoreConfig struct {
	// MaxEntries (optional; default: 10000) bounds the store; the least recently used entry is evicted past it.
	MaxEntries int

	// Now (optional; default: time.Now) is the clock used for expiry.
	Now func() time.Time
}

// WithDefaults returns a copy of the config with unset fields filled. It is safe to call on a nil receiver.
func (c *MemoryStoreConfig) WithDefaults() *MemoryStoreConfig {
	if c == nil {
		c = &MemoryStoreConfig{}
	}
	out := *c
	if out.MaxEntries == 0 {
		out.MaxEntries = defaultMemoryMaxEntries
	}
	if out.Now == nil {
		out.Now = time.Now
	}
	return &out
}

func (c *MemoryStoreConfig) validate() error {
	if c.MaxEntries <= 0 {
		return fmt.Errorf("cache: memory store max entries must be positive")
	}
	return nil
}

// MemoryStore is a bounded, per-process LRU Store. Each replica holds its own copy, so it suits small, hot values whose invalidation reaches every replica.
type MemoryStore struct {
	cfg MemoryStoreConfig

	mu      sync.Mutex
	entries map[string]*list.Element
	lru     *list.List
}

type memoryEntry struct {
	key       string
	value     []byte
	expiresAt time.Time
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore(cfg *MemoryStoreConfig) (*MemoryStore, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &MemoryStore{
		cfg:     *cfg,
		entries: make(map[string]*list.Element),
		lru:     list.New(),
	}, nil
}

func (s *MemoryStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	el, ok := s.entries[key]
	if !ok {
		return nil, false, nil
	}
	entry := el.Value.(*memoryEntry)
	if !s.cfg.Now().Before(entry.expiresAt) {
		s.remove(el)
		return nil, false, nil
	}
	s.lru.MoveToFront(el)
	return entry.value, true, nil
}

func (s *MemoryStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.set(key, value, ttl)
	return nil
}

func (s *MemoryStore) Add(_ context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if el, ok := s.entries[key]; ok && s.cfg.Now().Before(el.Value.(*memoryEntry).expiresAt) {
		return false, nil
	}
	s.set(key, value, ttl)
	return true, nil
}

func (s *MemoryStore) set(key string, value []byte, ttl time.Duration) {
	expiresAt := s.cfg.Now().Add(ttl)
	if el, ok := s.entries[key]; ok {
		entry := el.Value.(*memoryEntry)
		entry.value = value
		entry.expiresAt = expiresAt
		s.lru.MoveToFront(el)
		return
	}

	s.entries[key] = s.lru.PushFront(&memoryEntry{key: key, value: value, expiresAt: expiresAt})
	for s.lru.Len() > s.cfg.MaxEntries {
		s.remove(s.lru.Back())
	}
}

// Clear drops every entry, for when this replica may have missed invalidations.
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[string]*list.Element)
	s.lru.Init()
}

// Len returns the number of entries held, expired or not.
func (s *MemoryStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lru.Len()
}

func (s *MemoryStore) remove(el *list.Element) {
	s.lru.Remove(el)
	delete(s.entries, el.Value.(*memoryEntry).key)
}
