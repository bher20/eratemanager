package storage

import (
	"context"
	"sync"
	"time"

	"github.com/bher20/eratemanager/internal/rates"
)

// MemoryStorage is an in-memory Storage implementation, useful for tests and
// simple single-process deployments.
type MemoryStorage struct {
	mu        sync.RWMutex
	providers map[string]Provider
	snaps     map[string]RatesSnapshot
}

// NewMemory returns a MemoryStorage initialized with default providers.
func NewMemory() *MemoryStorage {
	m := &MemoryStorage{
		providers: make(map[string]Provider),
		snaps:     make(map[string]RatesSnapshot),
	}
	for _, p := range rates.Providers() {
		m.providers[p.Key] = Provider{
			Key:            p.Key,
			Name:           p.Name,
			LandingURL:     p.LandingURL,
			DefaultPDFPath: p.DefaultPDFPath,
			Notes:          p.Notes,
		}
	}
	return m
}

func (m *MemoryStorage) Close() error { return nil }

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
