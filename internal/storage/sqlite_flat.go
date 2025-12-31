package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStorage implements Storage using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

func OpenSQLite(dsn string) (*SQLiteStorage, error) {
	if dsn == "" {
		dsn = "eratemanager.db"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) Close() error { return s.db.Close() }

func (s *SQLiteStorage) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Migrate runs basic schema migrations for providers and rate snapshots.
func (s *SQLiteStorage) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS providers (
			key TEXT PRIMARY KEY,
			name TEXT,
			landing_url TEXT,
			default_pdf_path TEXT,
			notes TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS rates_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			payload BLOB NOT NULL,
			fetched_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS batch_progress (
			batch_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			PRIMARY KEY (batch_id, provider)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_batch_progress_status ON batch_progress(batch_id, status);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) ListProviders(ctx context.Context) ([]Provider, error) {
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

func (s *SQLiteStorage) GetProvider(ctx context.Context, key string) (*Provider, error) {
	row := s.db.QueryRowContext(ctx, `SELECT key, name, landing_url, default_pdf_path, notes FROM providers WHERE key = ?`, key)
	var p Provider
	if err := row.Scan(&p.Key, &p.Name, &p.LandingURL, &p.DefaultPDFPath, &p.Notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *SQLiteStorage) UpsertProvider(ctx context.Context, p Provider) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO providers (key, name, landing_url, default_pdf_path, notes)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			name = excluded.name,
			landing_url = excluded.landing_url,
			default_pdf_path = excluded.default_pdf_path,
			notes = excluded.notes
	`, p.Key, p.Name, p.LandingURL, p.DefaultPDFPath, p.Notes)
	return err
}

func (s *SQLiteStorage) GetRatesSnapshot(ctx context.Context, provider string) (*RatesSnapshot, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT payload, fetched_at
		FROM rates_snapshots
		WHERE provider = ?
		ORDER BY id DESC
		LIMIT 1
	`, provider)

	var payload []byte
	var fetched string
	if err := row.Scan(&payload, &fetched); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, fetched)
	if err != nil {
		// Fall back to now if parsing fails.
		t = time.Now()
	}
	return &RatesSnapshot{
		Provider:  provider,
		Payload:   payload,
		FetchedAt: t,
	}, nil
}

func (s *SQLiteStorage) SaveRatesSnapshot(ctx context.Context, snap RatesSnapshot) error {
	if snap.FetchedAt.IsZero() {
		snap.FetchedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rates_snapshots (provider, payload, fetched_at)
		VALUES (?, ?, ?)
	`, snap.Provider, snap.Payload, snap.FetchedAt.Format(time.RFC3339Nano))
	return err
}

func (s *SQLiteStorage) SaveBatchProgress(ctx context.Context, progress BatchProgress) error {
	var startedAt, completedAt sql.NullString
	if !progress.StartedAt.IsZero() {
		startedAt = sql.NullString{String: progress.StartedAt.Format(time.RFC3339Nano), Valid: true}
	}
	if !progress.CompletedAt.IsZero() {
		completedAt = sql.NullString{String: progress.CompletedAt.Format(time.RFC3339Nano), Valid: true}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO batch_progress (batch_id, provider, status, started_at, completed_at, error_message, retry_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(batch_id, provider) DO UPDATE SET
			status = excluded.status,
			started_at = excluded.started_at,
			completed_at = excluded.completed_at,
			error_message = excluded.error_message,
			retry_count = excluded.retry_count
	`, progress.BatchID, progress.Provider, progress.Status, startedAt, completedAt, progress.ErrorMessage, progress.RetryCount)
	return err
}

func (s *SQLiteStorage) GetBatchProgress(ctx context.Context, batchID, provider string) (*BatchProgress, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT batch_id, provider, status, started_at, completed_at, error_message, retry_count
		FROM batch_progress
		WHERE batch_id = ? AND provider = ?
	`, batchID, provider)

	var bp BatchProgress
	var startedAt, completedAt sql.NullString
	if err := row.Scan(&bp.BatchID, &bp.Provider, &bp.Status, &startedAt, &completedAt, &bp.ErrorMessage, &bp.RetryCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if startedAt.Valid {
		bp.StartedAt, _ = time.Parse(time.RFC3339Nano, startedAt.String)
	}
	if completedAt.Valid {
		bp.CompletedAt, _ = time.Parse(time.RFC3339Nano, completedAt.String)
	}

	return &bp, nil
}

func (s *SQLiteStorage) GetPendingBatchProviders(ctx context.Context, batchID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT provider FROM batch_progress
		WHERE batch_id = ? AND status IN ('pending', 'failed')
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
