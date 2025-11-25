package storage

import "context"

// Storage abstracts persistence for providers and rate snapshots.
type Storage interface {
	// Providers
	ListProviders(ctx context.Context) ([]Provider, error)
	GetProvider(ctx context.Context, key string) (*Provider, error)
	UpsertProvider(ctx context.Context, p Provider) error

	// Rates snapshots
	GetRatesSnapshot(ctx context.Context, provider string) (*RatesSnapshot, error)
	SaveRatesSnapshot(ctx context.Context, snap RatesSnapshot) error

	// Close releases any resources (no-op for in-memory).
	Close() error
}
