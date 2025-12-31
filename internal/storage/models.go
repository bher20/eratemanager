package storage

import "time"

// Provider holds metadata about a utility provider.
type Provider struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	LandingURL     string `json:"landingUrl"`
	DefaultPDFPath string `json:"defaultPdfPath"`
	Notes          string `json:"notes,omitempty"`
}

// RatesSnapshot stores a previously computed rates response payload for a provider.
type RatesSnapshot struct {
	Provider  string    `json:"provider"`
	Payload   []byte    `json:"payload"`
	FetchedAt time.Time `json:"fetched_at"`
}
