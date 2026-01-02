package storage

import (
	"context"
	"time"
)

// Storage abstracts persistence for providers and rate snapshots.
type Storage interface {
	// Providers
	ListProviders(ctx context.Context) ([]Provider, error)
	GetProvider(ctx context.Context, key string) (*Provider, error)
	UpsertProvider(ctx context.Context, p Provider) error

	// Rates snapshots
	GetRatesSnapshot(ctx context.Context, provider string) (*RatesSnapshot, error)
	SaveRatesSnapshot(ctx context.Context, snap RatesSnapshot) error

	// Batch progress tracking
	SaveBatchProgress(ctx context.Context, progress BatchProgress) error
	GetBatchProgress(ctx context.Context, batchID, provider string) (*BatchProgress, error)
	GetPendingBatchProviders(ctx context.Context, batchID string) ([]string, error)

	// Settings
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error

	// Close releases any resources (no-op for in-memory).
	Close() error

	// Ping checks connection readiness where applicable.
	Ping(ctx context.Context) error
}

// BatchProgress tracks the state of a single provider within a batch job.
type BatchProgress struct {
	BatchID      string    `json:"batch_id"`
	Provider     string    `json:"provider"`
	Status       string    `json:"status"` // "pending", "in_progress", "completed", "failed"
	StartedAt    time.Time `json:"started_at,omitempty"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	RetryCount   int       `json:"retry_count"`
}
