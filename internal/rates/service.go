package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bher20/eratemanager/internal/storage"
)

// Config controls how the rates service behaves.
// Provider-specific PDF paths can be set via environment variables or
// the provider descriptor's DefaultPDFPath.
type Config struct {
	// PDFPaths allows overriding PDF paths per provider key.
	// If empty, uses the provider's DefaultPDFPath from the registry.
	PDFPaths map[string]string
}

// Service coordinates fetching and caching of rates.
type Service struct {
	cfg   Config
	store storage.Storage // may be nil for PDF-only mode
}

// NewService preserves the original API: PDF-only, no storage caching.
func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// NewServiceWithStorage returns a Service that uses the provided storage
// backend to cache and read rates snapshots.
func NewServiceWithStorage(cfg Config, st storage.Storage) *Service {
	return &Service{cfg: cfg, store: st}
}

// GetResidential returns the residential rate structure based on provider key.
// It consults persistent storage first; on cache miss it parses PDFs and
// writes a new snapshot.
func (s *Service) GetResidential(ctx context.Context, provider string) (*RatesResponse, error) {
	// Use the registry to find the parser
	parser, ok := GetParser(provider)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (no parser registered)", provider)
	}

	loader := func() (*RatesResponse, error) {
		return s.parseProviderPDF(provider, parser)
	}

	return s.getProviderRates(ctx, provider, loader)
}

// getProviderRates is a small helper that tries storage first, then falls back
// to the provided PDF loader and writes the result back to storage.
func (s *Service) getProviderRates(
	ctx context.Context,
	key string,
	loader func() (*RatesResponse, error),
) (*RatesResponse, error) {
	// If we have a storage backend, try a cached snapshot first.
	if s.store != nil {
		snap, err := s.store.GetRatesSnapshot(ctx, key)
		if err == nil && snap != nil && len(snap.Payload) > 0 {
			var resp RatesResponse
			if err := json.Unmarshal(snap.Payload, &resp); err == nil {
				return &resp, nil
			}
			// If unmarshal fails, fall through to re-parse from PDF.
		}
	}

	// Cache miss or decode failure: compute from PDFs.
	resp, err := loader()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("nil response from loader for provider %s", key)
	}
	if resp.FetchedAt.IsZero() {
		resp.FetchedAt = time.Now()
	}

	// Best-effort write-back to storage.
	if s.store != nil {
		if payload, err := json.Marshal(resp); err == nil {
			_ = s.store.SaveRatesSnapshot(ctx, storage.RatesSnapshot{
				Provider:  key,
				Payload:   payload,
				FetchedAt: resp.FetchedAt,
			})
		}
	}

	return resp, nil
}

// parseProviderPDF is a generic PDF loader that uses the registry.
func (s *Service) parseProviderPDF(providerKey string, parser ParserConfig) (*RatesResponse, error) {
	// Check for override in config
	path := ""
	if s.cfg.PDFPaths != nil {
		path = s.cfg.PDFPaths[providerKey]
	}

	// Fall back to provider descriptor
	if path == "" {
		if p, ok := GetProvider(providerKey); ok && p.DefaultPDFPath != "" {
			path = p.DefaultPDFPath
		}
	}

	if path == "" {
		return nil, fmt.Errorf("no PDF path configured for %s", providerKey)
	}

	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("%s PDF not found at %s: %w", providerKey, path, err)
	}

	return parser.ParsePDF(path)
}

// ForceRefresh bypasses the cache and forces a fresh PDF parse for a provider.
// The result is saved to storage.
func (s *Service) ForceRefresh(ctx context.Context, provider string) (*RatesResponse, error) {
	// Use the registry to find the parser
	parser, ok := GetParser(provider)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (no parser registered)", provider)
	}

	resp, err := s.parseProviderPDF(provider, parser)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("nil response from parser for provider %s", provider)
	}
	if resp.FetchedAt.IsZero() {
		resp.FetchedAt = time.Now()
	}

	// Write-back to storage (best-effort)
	if s.store != nil {
		if payload, err := json.Marshal(resp); err == nil {
			_ = s.store.SaveRatesSnapshot(ctx, storage.RatesSnapshot{
				Provider:  provider,
				Payload:   payload,
				FetchedAt: resp.FetchedAt,
			})
		}
	}

	return resp, nil
}
