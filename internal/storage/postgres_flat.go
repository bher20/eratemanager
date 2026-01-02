package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStorage struct {
	db *sql.DB
}

func OpenPostgres(dsn string) (*PostgresStorage, error) {
	if dsn == "" {
		dsn = "postgres://localhost:5432/eratemanager?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Close() error { return s.db.Close() }

func (s *PostgresStorage) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *PostgresStorage) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS providers (
            key TEXT PRIMARY KEY,
            name TEXT,
            landing_url TEXT,
            default_pdf_path TEXT,
            notes TEXT
        );`,
		`CREATE TABLE IF NOT EXISTS rates_snapshots (
            id SERIAL PRIMARY KEY,
            provider TEXT NOT NULL,
            payload BYTEA NOT NULL,
            fetched_at TIMESTAMPTZ NOT NULL
        );`,
		`CREATE TABLE IF NOT EXISTS batch_progress (
            batch_id TEXT NOT NULL,
            provider TEXT NOT NULL,
            status TEXT NOT NULL,
            started_at TIMESTAMPTZ,
            completed_at TIMESTAMPTZ,
            error_message TEXT,
            retry_count INTEGER DEFAULT 0,
            PRIMARY KEY (batch_id, provider)
        );`,
		`CREATE INDEX IF NOT EXISTS idx_batch_progress_status ON batch_progress(batch_id, status);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStorage) ListProviders(ctx context.Context) ([]Provider, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, name, landing_url, default_pdf_path, notes FROM providers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.Key, &p.Name, &p.LandingURL, &p.DefaultPDFPath, &p.Notes); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) GetProvider(ctx context.Context, key string) (*Provider, error) {
	row := s.db.QueryRowContext(ctx, `SELECT key, name, landing_url, default_pdf_path, notes FROM providers WHERE key=$1`, key)
	var p Provider
	if err := row.Scan(&p.Key, &p.Name, &p.LandingURL, &p.DefaultPDFPath, &p.Notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *PostgresStorage) UpsertProvider(ctx context.Context, p Provider) error {
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO providers (key, name, landing_url, default_pdf_path, notes)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (key) DO UPDATE SET
            name = EXCLUDED.name,
            landing_url = EXCLUDED.landing_url,
            default_pdf_path = EXCLUDED.default_pdf_path,
            notes = EXCLUDED.notes
    `, p.Key, p.Name, p.LandingURL, p.DefaultPDFPath, p.Notes)
	return err
}

func (s *PostgresStorage) GetRatesSnapshot(ctx context.Context, provider string) (*RatesSnapshot, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT payload, fetched_at
        FROM rates_snapshots
        WHERE provider=$1
        ORDER BY id DESC
        LIMIT 1
    `, provider)

	var payload []byte
	var fetched time.Time
	if err := row.Scan(&payload, &fetched); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &RatesSnapshot{
		Provider:  provider,
		Payload:   payload,
		FetchedAt: fetched,
	}, nil
}

func (s *PostgresStorage) SaveRatesSnapshot(ctx context.Context, snap RatesSnapshot) error {
	if snap.FetchedAt.IsZero() {
		snap.FetchedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO rates_snapshots (provider, payload, fetched_at)
        VALUES ($1, $2, $3)
    `, snap.Provider, snap.Payload, snap.FetchedAt)
	return err
}

func (s *PostgresStorage) SaveBatchProgress(ctx context.Context, progress BatchProgress) error {
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO batch_progress (batch_id, provider, status, started_at, completed_at, error_message, retry_count)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (batch_id, provider) DO UPDATE SET
            status = EXCLUDED.status,
            started_at = EXCLUDED.started_at,
            completed_at = EXCLUDED.completed_at,
            error_message = EXCLUDED.error_message,
            retry_count = EXCLUDED.retry_count
    `, progress.BatchID, progress.Provider, progress.Status, progress.StartedAt, progress.CompletedAt, progress.ErrorMessage, progress.RetryCount)
	return err
}

func (s *PostgresStorage) GetBatchProgress(ctx context.Context, batchID, provider string) (*BatchProgress, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT batch_id, provider, status, started_at, completed_at, error_message, retry_count
        FROM batch_progress
        WHERE batch_id = $1 AND provider = $2
    `, batchID, provider)

	var p BatchProgress
	var startedAt, completedAt *time.Time
	if err := row.Scan(&p.BatchID, &p.Provider, &p.Status, &startedAt, &completedAt, &p.ErrorMessage, &p.RetryCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if startedAt != nil {
		p.StartedAt = *startedAt
	}
	if completedAt != nil {
		p.CompletedAt = *completedAt
	}
	return &p, nil
}

func (s *PostgresStorage) GetPendingBatchProviders(ctx context.Context, batchID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT provider FROM batch_progress
        WHERE batch_id = $1 AND status IN ('pending', 'failed')
        ORDER BY provider
    `, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (s *PostgresStorage) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *PostgresStorage) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, $3)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now())
	return err
}
