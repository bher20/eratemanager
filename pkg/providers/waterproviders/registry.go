package waterproviders

import (
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]WaterProvider)
)

// Register registers a water provider.
func Register(p WaterProvider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if p == nil {
		panic("waterproviders: Register provider is nil")
	}
	if _, dup := registry[p.Key()]; dup {
		panic("waterproviders: Register called twice for provider " + p.Key())
	}
	registry[p.Key()] = p
}

// Get returns a water provider by key.
func Get(key string) (WaterProvider, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := registry[key]
	return p, ok
}

// List returns a sorted list of registered water provider keys.
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

// GetAll returns all registered water providers.
func GetAll() []WaterProvider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var providers []WaterProvider
	for _, p := range registry {
		providers = append(providers, p)
	}
	return providers
}
