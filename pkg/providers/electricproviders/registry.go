package electricproviders

import (
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]ElectricProvider)
)

// Register registers an electric provider.
func Register(p ElectricProvider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if p == nil {
		panic("electricproviders: Register provider is nil")
	}
	if _, dup := registry[p.Key()]; dup {
		panic("electricproviders: Register called twice for provider " + p.Key())
	}
	registry[p.Key()] = p
}

// Get returns an electric provider by key.
func Get(key string) (ElectricProvider, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := registry[key]
	return p, ok
}

// List returns a sorted list of registered electric provider keys.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var keys []string
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetAll returns all registered electric providers.
func GetAll() []ElectricProvider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var providers []ElectricProvider
	for _, p := range registry {
		providers = append(providers, p)
	}
	return providers
}
