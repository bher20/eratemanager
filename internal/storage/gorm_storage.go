package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type GormStorage struct {
	db *gorm.DB
}

func NewGormStorage(driver, dsn string) (*GormStorage, error) {
	var gormDialector gorm.Dialector
	if driver == "postgres" || driver == "postgrespool" {
		gormDialector = postgres.Open(dsn)
	} else if driver == "sqlite" {
		gormDialector = sqlite.Open(dsn)
	} else {
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	db, err := gorm.Open(gormDialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	return &GormStorage{db: db}, nil
}

func (s *GormStorage) Migrate(ctx context.Context) error {
	// AutoMigrate all models
	// Note: We need to define structs that cover all tables.
	// We use the structs from models.go mostly.
	// For tables not in models.go like 'scheduled_jobs', we define them locally or add to models.
	
	err := s.db.AutoMigrate(
		&Provider{},
		&RatesSnapshot{},
		&BatchProgress{},
		&Setting{},
		&User{},
		&Token{},
		&CasbinRule{},
		&EmailConfig{},
		&ScheduledJob{},
	)
	return err
}

// Additional models needed for GORM that might not be in models.go
type Setting struct {
	Key       string    `gorm:"primaryKey;column:key"`
	Value     string    `gorm:"column:value"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

type ScheduledJob struct {
	Name           string    `gorm:"primaryKey;column:name"`
	LastRunAt      time.Time `gorm:"column:last_run_at"`
	LastDurationMs int64     `gorm:"column:last_duration_ms"`
	LastSuccess    int       `gorm:"column:last_success"`
	LastError      string    `gorm:"column:last_error"`
}

// Storage Interface Implementation

// Providers

func (s *GormStorage) ListProviders(ctx context.Context) ([]Provider, error) {
	var providers []Provider
	result := s.db.WithContext(ctx).Find(&providers)
	return providers, result.Error
}

func (s *GormStorage) GetProvider(ctx context.Context, key string) (*Provider, error) {
	var provider Provider
	result := s.db.WithContext(ctx).First(&provider, "key = ?", key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found, consistent with other implementations
		}
		return nil, result.Error
	}
	return &provider, nil
}

func (s *GormStorage) UpsertProvider(ctx context.Context, p Provider) error {
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		UpdateAll: true,
	}).Create(&p).Error
}

// RatesSnapshot

func (s *GormStorage) GetRatesSnapshot(ctx context.Context, provider string) (*RatesSnapshot, error) {
	var snap RatesSnapshot
	// Get latest
	result := s.db.WithContext(ctx).Order("fetched_at desc").First(&snap, "provider = ?", provider)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &snap, nil
}

func (s *GormStorage) SaveRatesSnapshot(ctx context.Context, snap RatesSnapshot) error {
	return s.db.WithContext(ctx).Create(&snap).Error
}

// BatchProgress

func (s *GormStorage) SaveBatchProgress(ctx context.Context, progress BatchProgress) error {
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "batch_id"}, {Name: "provider"}},
		UpdateAll: true,
	}).Create(&progress).Error
}

func (s *GormStorage) GetBatchProgress(ctx context.Context, batchID, provider string) (*BatchProgress, error) {
	var prog BatchProgress
	result := s.db.WithContext(ctx).First(&prog, "batch_id = ? AND provider = ?", batchID, provider)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &prog, nil
}

func (s *GormStorage) GetPendingBatchProviders(ctx context.Context, batchID string) ([]string, error) {
	var providers []string
	result := s.db.WithContext(ctx).Model(&BatchProgress{}).
		Where("batch_id = ? AND status IN ('pending', 'failed')", batchID).
		Pluck("provider", &providers)
	return providers, result.Error
}

// Settings

func (s *GormStorage) GetSetting(ctx context.Context, key string) (string, error) {
	var setting Setting
	result := s.db.WithContext(ctx).First(&setting, "key = ?", key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", result.Error
	}
	return setting.Value, nil
}

func (s *GormStorage) SetSetting(ctx context.Context, key, value string) error {
	setting := Setting{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		UpdateAll: true,
	}).Create(&setting).Error
}

// Users

func (s *GormStorage) CreateUser(ctx context.Context, user User) error {
	return s.db.WithContext(ctx).Create(&user).Error
}

func (s *GormStorage) GetUser(ctx context.Context, id string) (*User, error) {
	var user User
	result := s.db.WithContext(ctx).First(&user, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (s *GormStorage) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	result := s.db.WithContext(ctx).First(&user, "username = ?", username)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (s *GormStorage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	result := s.db.WithContext(ctx).First(&user, "email = ?", email)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (s *GormStorage) UpdateUser(ctx context.Context, user User) error {
	return s.db.WithContext(ctx).Save(&user).Error
}

func (s *GormStorage) DeleteUser(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&User{}, "id = ?", id).Error
}

func (s *GormStorage) ListUsers(ctx context.Context) ([]User, error) {
	var users []User
	result := s.db.WithContext(ctx).Find(&users)
	return users, result.Error
}

// Tokens

func (s *GormStorage) CreateToken(ctx context.Context, token Token) error {
	return s.db.WithContext(ctx).Create(&token).Error
}

func (s *GormStorage) GetToken(ctx context.Context, id string) (*Token, error) {
	var token Token
	result := s.db.WithContext(ctx).First(&token, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &token, nil
}

func (s *GormStorage) GetTokenByHash(ctx context.Context, hash string) (*Token, error) {
	var token Token
	result := s.db.WithContext(ctx).First(&token, "token_hash = ?", hash)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &token, nil
}

func (s *GormStorage) ListTokens(ctx context.Context, userID string) ([]Token, error) {
	var tokens []Token
	result := s.db.WithContext(ctx).Find(&tokens, "user_id = ?", userID)
	return tokens, result.Error
}

func (s *GormStorage) DeleteToken(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&Token{}, "id = ?", id).Error
}

func (s *GormStorage) UpdateTokenLastUsed(ctx context.Context, id string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Token{}).Where("id = ?", id).Update("last_used_at", now).Error
}

// Casbin Rules

func (s *GormStorage) LoadCasbinRules(ctx context.Context) ([]CasbinRule, error) {
	var rules []CasbinRule
	result := s.db.WithContext(ctx).Find(&rules)
	return rules, result.Error
}

func (s *GormStorage) AddCasbinRule(ctx context.Context, rule CasbinRule) error {
	return s.db.WithContext(ctx).Create(&rule).Error
}

func (s *GormStorage) RemoveCasbinRule(ctx context.Context, rule CasbinRule) error {
	return s.db.WithContext(ctx).Where(&rule).Delete(&CasbinRule{}).Error
}

// Email Config

func (s *GormStorage) GetEmailConfig(ctx context.Context) (*EmailConfig, error) {
	var config EmailConfig
	result := s.db.WithContext(ctx).First(&config)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &config, nil
}

func (s *GormStorage) SaveEmailConfig(ctx context.Context, config EmailConfig) error {
	// There should only be one config, so we can clean up others or just use ID if fixed
	if config.ID == "" {
		config.ID = "default" // Force single row if not specified
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&config).Error
}

// Close & Ping

func (s *GormStorage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *GormStorage) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Scheduled Jobs & Locking

func (s *GormStorage) AcquireAdvisoryLock(ctx context.Context, key int64) (bool, error) {
	if s.db.Dialector.Name() == "postgres" {
		var ok bool
		// GORM Raw for specific function
		err := s.db.WithContext(ctx).Raw("SELECT pg_try_advisory_lock(?)", key).Scan(&ok).Error
		return ok, err
	}
	// For SQLite, no advisory locks, assume always successful (single instance)
	return true, nil
}

func (s *GormStorage) ReleaseAdvisoryLock(ctx context.Context, key int64) (bool, error) {
	if s.db.Dialector.Name() == "postgres" {
		var ok bool
		err := s.db.WithContext(ctx).Raw("SELECT pg_advisory_unlock(?)", key).Scan(&ok).Error
		return ok, err
	}
	return true, nil
}

func (s *GormStorage) UpdateScheduledJob(ctx context.Context, name string, started time.Time, dur time.Duration, success bool, errMsg string) error {
	status := 0
	if success {
		status = 1
	}
	job := ScheduledJob{
		Name:           name,
		LastRunAt:      started,
		LastDurationMs: dur.Milliseconds(),
		LastSuccess:    status,
		LastError:      errMsg,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		UpdateAll: true,
	}).Create(&job).Error
}
