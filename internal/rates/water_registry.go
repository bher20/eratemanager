package rates

import (
	"fmt"
	"sync"
)

// WaterParserFunc is a function that fetches and parses water rates from a URL.
type WaterParserFunc func(url string) (*WaterRatesResponse, error)

// WaterParserConfig holds the configuration for a water provider's parser.
type WaterParserConfig struct {
	// Key is the unique identifier for this provider (e.g., "whud").
	Key string

	// Name is the human-readable name of the utility.
	Name string

	// ParseHTML fetches and parses an HTML page at the given URL.
	ParseHTML WaterParserFunc
}

var (
	waterParsersMu sync.RWMutex
	waterParsers   = make(map[string]WaterParserConfig)
)

// RegisterWaterParser registers a parser configuration for a water provider.
// This is typically called from an init() function in each parser file.
func RegisterWaterParser(cfg WaterParserConfig) {
	if cfg.Key == "" {
		panic("rates: RegisterWaterParser called with empty key")
	}
	if cfg.ParseHTML == nil {
		panic(fmt.Sprintf("rates: RegisterWaterParser(%q) called with nil ParseHTML", cfg.Key))
	}

	waterParsersMu.Lock()
	defer waterParsersMu.Unlock()

	if _, exists := waterParsers[cfg.Key]; exists {
		panic(fmt.Sprintf("rates: RegisterWaterParser called twice for key %q", cfg.Key))
	}
	waterParsers[cfg.Key] = cfg
}

// GetWaterParser returns the parser configuration for a water provider key.
func GetWaterParser(key string) (WaterParserConfig, bool) {
	waterParsersMu.RLock()
	defer waterParsersMu.RUnlock()

	cfg, ok := waterParsers[key]
	return cfg, ok
}

// ListWaterParsers returns all registered water parser keys.
func ListWaterParsers() []string {
	waterParsersMu.RLock()
	defer waterParsersMu.RUnlock()

	keys := make([]string, 0, len(waterParsers))
	for k := range waterParsers {
		keys = append(keys, k)
	}
	return keys
}
