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

// User represents a registered user in the system.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Token represents an API access token.
type Token struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"`
	Role       string     `json:"role"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}
