package providers

import "errors"

// ProviderType represents the type of utility provider (electric or water).
type ProviderType string

const (
	ProviderTypeElectric ProviderType = "electric"
	ProviderTypeWater    ProviderType = "water"
)

// Provider is the base interface for all utility providers.
type Provider interface {
	// Key returns the unique identifier for the provider (e.g., "cemc", "whud").
	Key() string
	// Name returns the human-readable name of the provider.
	Name() string
	// Type returns the type of the provider.
	Type() ProviderType
	// LandingURL returns the URL to the provider's rates page.
	LandingURL() string
	// DefaultPDFPath returns the default path to the PDF file (if applicable).
	DefaultPDFPath() string
}

// Common errors shared across providers.
var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrParseFailed      = errors.New("failed to parse rates")
	ErrNotImplemented   = errors.New("not implemented")
)
