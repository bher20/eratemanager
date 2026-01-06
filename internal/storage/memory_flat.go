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
	settings      map[string]string
	users         map[string]User
	tokens        map[string]Token
	emailConfig   *EmailConfig
}

// NewMemory returns a MemoryStorage initialized with default providers.
func NewMemory() *MemoryStorage {
	m := &MemoryStorage{
		providers:     make(map[string]Provider),
		snaps:         make(map[string]RatesSnapshot),
		batchProgress: make(map[string]BatchProgress),
		settings:      make(map[string]string),
		users:         make(map[string]User),
		tokens:        make(map[string]Token),
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
		settings:      make(map[string]string),
		users:         make(map[string]User),
		tokens:        make(map[string]Token),
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

func (m *MemoryStorage) GetSetting(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings[key], nil
}

func (m *MemoryStorage) SetSetting(ctx context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[key] = value
	return nil
}

// Users

func (m *MemoryStorage) CreateUser(ctx context.Context, user User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
	return nil
}

func (m *MemoryStorage) GetUser(ctx context.Context, id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return &u, nil
}

func (m *MemoryStorage) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *MemoryStorage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *MemoryStorage) UpdateUser(ctx context.Context, user User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[user.ID]; !ok {
		return nil // or error
	}
	m.users[user.ID] = user
	return nil
}

func (m *MemoryStorage) DeleteUser(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, id)
	return nil
}

func (m *MemoryStorage) ListUsers(ctx context.Context) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []User
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, nil
}

// Tokens

func (m *MemoryStorage) CreateToken(ctx context.Context, token Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token.ID] = token
	return nil
}

func (m *MemoryStorage) GetToken(ctx context.Context, id string) (*Token, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tokens[id]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

func (m *MemoryStorage) GetTokenByHash(ctx context.Context, hash string) (*Token, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tokens {
		if t.TokenHash == hash {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *MemoryStorage) ListTokens(ctx context.Context, userID string) ([]Token, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Token
	for _, t := range m.tokens {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *MemoryStorage) DeleteToken(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, id)
	return nil
}

func (m *MemoryStorage) UpdateTokenLastUsed(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tokens[id]; ok {
		now := time.Now()
		t.LastUsedAt = &now
		m.tokens[id] = t
	}
	return nil
}

func (m *MemoryStorage) LoadCasbinRules(ctx context.Context) ([]CasbinRule, error) {
	// In-memory storage doesn't persist rules, so we return empty.
	// The Enforcer will start with default policies.
	return nil, nil
}

func (m *MemoryStorage) AddCasbinRule(ctx context.Context, rule CasbinRule) error {
	// No-op for memory storage as Casbin handles in-memory state itself
	return nil
}

func (m *MemoryStorage) RemoveCasbinRule(ctx context.Context, rule CasbinRule) error {
	// No-op
	return nil
}

func (m *MemoryStorage) GetEmailConfig(ctx context.Context) (*EmailConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.emailConfig == nil {
		return nil, nil
	}
	// Return a copy
	cfg := *m.emailConfig
	return &cfg, nil
}

func (m *MemoryStorage) SaveEmailConfig(ctx context.Context, config EmailConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emailConfig = &config
	return nil
}

func (m *MemoryStorage) AcquireAdvisoryLock(ctx context.Context, key int64) (bool, error) {
// In-memory single instance always acquires lock
return true, nil
}

func (m *MemoryStorage) ReleaseAdvisoryLock(ctx context.Context, key int64) (bool, error) {
return true, nil
}

func (m *MemoryStorage) UpdateScheduledJob(ctx context.Context, name string, started time.Time, dur time.Duration, success bool, errMsg string) error {
// No-op for memory storage
return nil
}
