package storage

import (
	"context"
	"fmt"
	"log"
)

// Config controls how the storage backend is opened.
type Config struct {
	Driver    string
	DSN       string
	Providers []Provider
}

// Open constructs a Storage based on the given configuration.
func Open(ctx context.Context, cfg Config) (Storage, error) {
	drv := cfg.Driver
	if drv == "" {
		drv = "memory"
	}
	switch drv {
	case "memory":
		log.Printf("storage: using in-memory backend")
		if len(cfg.Providers) > 0 {
			return NewMemoryWithProviders(cfg.Providers), nil
		}
		return NewMemory(), nil

	case "sqlite", "postgres", "postgrespool": // All handled by GORM now
		log.Printf("storage: using gorm driver=%s", drv)
		st, err := NewGormStorage(drv, cfg.DSN)
		if err != nil {
			return nil, err
		}
		if err := st.Migrate(ctx); err != nil {
			st.Close()
			return nil, fmt.Errorf("storage migrate: %w", err)
		}
		return st, nil

	default:
		return nil, fmt.Errorf("unsupported storage driver %q", drv)
	}
}
