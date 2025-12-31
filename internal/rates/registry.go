package rates

import (
	"fmt"
	"os"
	"sync"
)

// ParserFunc is a function that parses a PDF file and returns rates.
type ParserFunc func(path string) (*RatesResponse, error)

// TextParserFunc is a function that parses extracted PDF text and returns rates.
type TextParserFunc func(text string) (*RatesResponse, error)

// ParserConfig holds the configuration for a provider's parser.
type ParserConfig struct {
	// Key is the unique identifier for this provider (e.g., "cemc", "nes").
	Key string

	// Name is the human-readable name of the utility.
	Name string

	// ParsePDF parses a PDF file at the given path.
	ParsePDF ParserFunc

	// ParseText parses extracted text from a PDF (useful for testing).
	ParseText TextParserFunc
}

var (
	parsersMu sync.RWMutex
	parsers   = make(map[string]ParserConfig)
)

// RegisterParser registers a parser configuration for a provider.
// This is typically called from an init() function in each parser file.
func RegisterParser(cfg ParserConfig) {
	if cfg.Key == "" {
		panic("rates: RegisterParser called with empty key")
	}
	if cfg.ParsePDF == nil {
		panic(fmt.Sprintf("rates: RegisterParser(%q) called with nil ParsePDF", cfg.Key))
	}

	parsersMu.Lock()
	defer parsersMu.Unlock()

	if _, exists := parsers[cfg.Key]; exists {
		panic(fmt.Sprintf("rates: RegisterParser called twice for key %q", cfg.Key))
	}
	parsers[cfg.Key] = cfg
}

// GetParser returns the parser configuration for a provider key.
func GetParser(key string) (ParserConfig, bool) {
	parsersMu.RLock()
	defer parsersMu.RUnlock()

	cfg, ok := parsers[key]
	return cfg, ok
}

// ListParsers returns all registered parser keys.
func ListParsers() []string {
	parsersMu.RLock()
	defer parsersMu.RUnlock()

	keys := make([]string, 0, len(parsers))
	for k := range parsers {
		keys = append(keys, k)
	}
	return keys
}

// ParseProviderPDF is a convenience function that looks up the parser for a
// provider and parses the PDF at the configured path.
func ParseProviderPDF(providerKey string) (*RatesResponse, error) {
	parser, ok := GetParser(providerKey)
	if !ok {
		return nil, fmt.Errorf("no parser registered for provider: %s", providerKey)
	}

	// Get the PDF path from provider descriptor
	provider, ok := GetProvider(providerKey)
	if !ok {
		return nil, fmt.Errorf("no provider descriptor for: %s", providerKey)
	}

	path := provider.DefaultPDFPath
	if path == "" {
		return nil, fmt.Errorf("no PDF path configured for provider: %s", providerKey)
	}

	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("PDF not found at %s: %w", path, err)
	}

	return parser.ParsePDF(path)
}
