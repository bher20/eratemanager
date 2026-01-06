package storage

import "time"

// Provider holds metadata about a utility provider.
type Provider struct {
	Key            string `json:"key" gorm:"primaryKey;column:key"`
	Name           string `json:"name" gorm:"column:name"`
	LandingURL     string `json:"landingUrl" gorm:"column:landing_url"`
	DefaultPDFPath string `json:"defaultPdfPath" gorm:"column:default_pdf_path"`
	Notes          string `json:"notes,omitempty" gorm:"column:notes"`
}

// RatesSnapshot stores a previously computed rates response payload for a provider.
type RatesSnapshot struct {
	ID        uint      `json:"-" gorm:"primaryKey;column:id"`
	Provider  string    `json:"provider" gorm:"column:provider"`
	Payload   []byte    `json:"payload" gorm:"column:payload"`
	FetchedAt time.Time `json:"fetched_at" gorm:"column:fetched_at"`
}

// User represents a registered user in the system.
type User struct {
	ID           string    `json:"id" gorm:"primaryKey;column:id"`
	Username     string    `json:"username" gorm:"unique;column:username"`
	FirstName    string    `json:"first_name" gorm:"column:first_name"`
	LastName     string    `json:"last_name" gorm:"column:last_name"`
	Email        string    `json:"email" gorm:"column:email"`
	EmailVerified bool     `json:"email_verified" gorm:"column:email_verified"`
	SkipEmailVerification bool `json:"skip_email_verification" gorm:"column:skip_email_verification"`
	OnboardingCompleted bool `json:"onboarding_completed" gorm:"column:onboarding_completed"`
	PasswordHash string    `json:"-" gorm:"column:password_hash"`
	Role         string    `json:"role" gorm:"column:role"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// Token represents an API access token.
type Token struct {
	ID         string     `json:"id" gorm:"primaryKey;column:id"`
	UserID     string     `json:"user_id" gorm:"column:user_id"`
	Name       string     `json:"name" gorm:"column:name"`
	TokenHash  string     `json:"-" gorm:"column:token_hash"`
	Role       string     `json:"role" gorm:"column:role"`
	CreatedAt  time.Time  `json:"created_at" gorm:"column:created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" gorm:"column:expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" gorm:"column:last_used_at"`
}

// CasbinRule represents a policy rule for RBAC.
type CasbinRule struct {
	ID    uint   `gorm:"primaryKey"`
	PType string `json:"ptype" gorm:"column:ptype"`
	V0    string `json:"v0" gorm:"column:v0"`
	V1    string `json:"v1" gorm:"column:v1"`
	V2    string `json:"v2" gorm:"column:v2"`
	V3    string `json:"v3" gorm:"column:v3"`
	V4    string `json:"v4" gorm:"column:v4"`
	V5    string `json:"v5" gorm:"column:v5"`
}
// EmailConfig holds configuration for email notifications.
type EmailConfig struct {
	ID          string    `json:"id" gorm:"primaryKey;column:id"`
	Provider    string    `json:"provider" gorm:"column:provider"` // "smtp", "sendgrid", "gmail", "resend"
	Host        string    `json:"host,omitempty" gorm:"column:host"`
	Port        int       `json:"port,omitempty" gorm:"column:port"`
	Username    string    `json:"username,omitempty" gorm:"column:username"`
	Password    string    `json:"password,omitempty" gorm:"column:password"`
	FromAddress string    `json:"from_address" gorm:"column:from_address"`
	FromName    string    `json:"from_name" gorm:"column:from_name"`
	APIKey      string    `json:"api_key,omitempty" gorm:"column:api_key"` // For Sendgrid
	Encryption  string    `json:"encryption,omitempty" gorm:"column:encryption"` // "none", "ssl", "tls"
	Enabled     bool      `json:"enabled" gorm:"column:enabled"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`
}
