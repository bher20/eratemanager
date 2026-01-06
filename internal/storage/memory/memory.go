package memory

import (
	"context"
	"sync"
	"time"

	"github.com/bher20/eratemanager/internal/storage"
	"github.com/bher20/eratemanager/pkg/providers/electricproviders"
	"github.com/bher20/eratemanager/pkg/providers/waterproviders"
)

// MemoryStorage is an in-memory Storage implementation, useful for tests and
// simple single-process deployments.
type MemoryStorage struct {
	mu        sync.RWMutex
	providers map[string]storage.Provider
	snaps     map[string]storage.RatesSnapshot
}

// New seeds the in-memory store from registered providers so that default
// providers are available without needing a database.
func New() *MemoryStorage {
	m := &MemoryStorage{
		providers: make(map[string]storage.Provider),
		snaps:     make(map[string]storage.RatesSnapshot),
	}

	for _, p := range electricproviders.GetAll() {
		m.providers[p.Key()] = storage.Provider{
			Key:            p.Key(),
			Name:           p.Name(),
			LandingURL:     p.LandingURL(),
			DefaultPDFPath: p.DefaultPDFPath(),
		}
	}
	for _, p := range waterproviders.GetAll() {
		m.providers[p.Key()] = storage.Provider{
			Key:            p.Key(),
			Name:           p.Name(),
			LandingURL:     p.LandingURL(),
			DefaultPDFPath: p.DefaultPDFPath(),
		}
	}

	return m
}

func (m *MemoryStorage) Close() error { return nil }

// ListProviders returns all known providers.
func (m *MemoryStorage) ListProviders(ctx context.Context) ([]storage.Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]storage.Provider, 0, len(m.providers))
	for _, p := range m.providers {
		out = append(out, p)
	}
	return out, nil
}

// GetProvider looks up a provider by key.
func (m *MemoryStorage) GetProvider(ctx context.Context, key string) (*storage.Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.providers[key]
	if !ok {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

// UpsertProvider inserts or updates a provider.
func (m *MemoryStorage) UpsertProvider(ctx context.Context, p storage.Provider) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.Key] = p
	return nil
}

// GetRatesSnapshot returns the most recent snapshot for a provider, if any.
func (m *MemoryStorage) GetRatesSnapshot(ctx context.Context, provider string) (*storage.RatesSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.snaps[provider]
	if !ok {
		return nil, nil
	}
	cp := s
	return &cp, nil
}

// SaveRatesSnapshot stores a rates snapshot for a provider.
func (m *MemoryStorage) SaveRatesSnapshot(ctx context.Context, snap storage.RatesSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if snap.FetchedAt.IsZero() {
		snap.FetchedAt = time.Now()
	}
	m.snaps[snap.Provider] = snap
	return nil
}
