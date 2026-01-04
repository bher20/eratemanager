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

	// Users
	CreateUser(ctx context.Context, user User) error
	GetUser(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	UpdateUser(ctx context.Context, user User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context) ([]User, error)

	// Tokens
	CreateToken(ctx context.Context, token Token) error
	GetToken(ctx context.Context, id string) (*Token, error)
	GetTokenByHash(ctx context.Context, hash string) (*Token, error)
	ListTokens(ctx context.Context, userID string) ([]Token, error)
	DeleteToken(ctx context.Context, id string) error
	UpdateTokenLastUsed(ctx context.Context, id string) error

	// Casbin Rules
	LoadCasbinRules(ctx context.Context) ([]CasbinRule, error)
	AddCasbinRule(ctx context.Context, rule CasbinRule) error
	RemoveCasbinRule(ctx context.Context, rule CasbinRule) error

	// Email Configuration
	GetEmailConfig(ctx context.Context) (*EmailConfig, error)
	SaveEmailConfig(ctx context.Context, config EmailConfig) error

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
