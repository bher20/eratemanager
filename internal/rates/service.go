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
type Config struct {
    // CEMCPDFPath is an optional filesystem path to a cached CEMC rates PDF.
    CEMCPDFPath string
    // NESPDFPath is an optional filesystem path to a cached NES rates PDF.
    NESPDFPath string
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
    switch provider {
    case "cemc":
        return s.getProviderRates(ctx, "cemc", s.getCEMCRatesFromPDF)
    case "nes":
        return s.getProviderRates(ctx, "nes", s.getNESRatesFromPDF)
    default:
        return nil, fmt.Errorf("unknown provider: %s", provider)
    }
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

// getCEMCRatesFromPDF maintains the existing PDF-loading behavior for CEMC.
func (s *Service) getCEMCRatesFromPDF() (*RatesResponse, error) {
    path := s.cfg.CEMCPDFPath
    if path == "" {
        if p, ok := GetProvider("cemc"); ok && p.DefaultPDFPath != "" {
            path = p.DefaultPDFPath
        }
    }
    if path == "" {
        return nil, fmt.Errorf("no PDF path configured for CEMC")
    }
    if _, err := os.Stat(path); err != nil {
        return nil, fmt.Errorf("CEMC PDF not found at %s: %w", path, err)
    }
    return ParseCEMCRatesFromPDF(path)
}

// getNESRatesFromPDF maintains the existing PDF-loading behavior for NES.
func (s *Service) getNESRatesFromPDF() (*RatesResponse, error) {
    path := s.cfg.NESPDFPath
    if path == "" {
        if p, ok := GetProvider("nes"); ok && p.DefaultPDFPath != "" {
            path = p.DefaultPDFPath
        }
    }
    if path == "" {
        return nil, fmt.Errorf("no PDF path configured for NES")
    }
    if _, err := os.Stat(path); err != nil {
        return nil, fmt.Errorf("NES PDF not found at %s: %w", path, err)
    }
    return ParseNESRatesFromPDF(path)
}
