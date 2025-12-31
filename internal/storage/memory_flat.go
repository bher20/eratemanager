package storage

import (
	"context"
	"sync"
	"time"
)

// MemoryStorage is an in-memory Storage implementation, useful for tests and
// simple single-process deployments.
type MemoryStorage struct {
	mu            sync.RWMutex
	providers     map[string]Provider
	snaps         map[string]RatesSnapshot
	batchProgress map[string]BatchProgress
}

// NewMemory returns a MemoryStorage initialized with default providers.
func NewMemory() *MemoryStorage {
	m := &MemoryStorage{
		providers:     make(map[string]Provider),
		snaps:         make(map[string]RatesSnapshot),
		batchProgress: make(map[string]BatchProgress),
	}
	return m
}

// NewMemoryWithProviders returns a MemoryStorage initialized with the given
// provider list. This avoids importing the `rates` package into `storage`
// and thus prevents import cycles; conversion should be done by callers.
func NewMemoryWithProviders(list []Provider) *MemoryStorage {
	m := &MemoryStorage{
		providers:     make(map[string]Provider),
		snaps:         make(map[string]RatesSnapshot),
		batchProgress: make(map[string]BatchProgress),
	}
	for _, p := range list {
		m.providers[p.Key] = p
	}
	return m
}

func (m *MemoryStorage) Close() error { return nil }

func (m *MemoryStorage) Ping(ctx context.Context) error { return nil }

func (m *MemoryStorage) ListProviders(ctx context.Context) ([]Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Provider, 0, len(m.providers))
	for _, p := range m.providers {
		cp := p
		out = append(out, cp)
	}
	return out, nil
}

func (m *MemoryStorage) GetProvider(ctx context.Context, key string) (*Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[key]
	if !ok {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

func (m *MemoryStorage) UpsertProvider(ctx context.Context, p Provider) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.Key] = p
	return nil
}

func (m *MemoryStorage) GetRatesSnapshot(ctx context.Context, provider string) (*RatesSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.snaps[provider]
	if !ok {
		return nil, nil
	}
	cp := s
	return &cp, nil
}

func (m *MemoryStorage) SaveRatesSnapshot(ctx context.Context, snap RatesSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if snap.FetchedAt.IsZero() {
		snap.FetchedAt = time.Now()
	}
	m.snaps[snap.Provider] = snap
	return nil
}

func (m *MemoryStorage) SaveBatchProgress(ctx context.Context, progress BatchProgress) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := progress.BatchID + ":" + progress.Provider
	m.batchProgress[key] = progress
	return nil
}

func (m *MemoryStorage) GetBatchProgress(ctx context.Context, batchID, provider string) (*BatchProgress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := batchID + ":" + provider
	p, ok := m.batchProgress[key]
	if !ok {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

func (m *MemoryStorage) GetPendingBatchProviders(ctx context.Context, batchID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var providers []string
	prefix := batchID + ":"
	for key, p := range m.batchProgress {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			if p.Status == "pending" || p.Status == "failed" {
				providers = append(providers, p.Provider)
			}
		}
	}
	return providers, nil
}
