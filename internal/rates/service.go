package rates

import (
    "context"
    "fmt"
    "os"
)

// Config controls how the rates service behaves.
type Config struct {
    // CEMCPDFPath is an optional filesystem path to a cached CEMC rates PDF.
    CEMCPDFPath string
    // NESPDFPath is an optional filesystem path to a cached NES rates PDF.
    NESPDFPath string
}

type Service struct {
    cfg Config
}

func NewService(cfg Config) *Service {
    return &Service{cfg: cfg}
}

// GetResidential returns the residential rate structure based on provider key.
func (s *Service) GetResidential(ctx context.Context, provider string) (*RatesResponse, error) {
    switch provider {
    case "cemc":
        return s.getCEMCRates()
    case "nes":
        return s.getNESRates()
    default:
        return nil, fmt.Errorf("unknown provider: %s", provider)
    }
}

func (s *Service) getCEMCRates() (*RatesResponse, error) {
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

func (s *Service) getNESRates() (*RatesResponse, error) {
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
