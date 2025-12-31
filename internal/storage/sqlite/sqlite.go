package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"

	"github.com/bher20/eratemanager/internal/storage"
)

// SQLiteStorage implements storage.Storage using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

// Open creates a new SQLiteStorage using the given DSN/path.
func Open(dsn string) (*SQLiteStorage, error) {
	if dsn == "" {
		// Default to a local file if not provided.
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
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// ListProviders returns all providers in the providers table.
func (s *SQLiteStorage) ListProviders(ctx context.Context) ([]storage.Provider, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, name, landing_url, default_pdf_path, notes FROM providers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []storage.Provider
	for rows.Next() {
		var p storage.Provider
		if err := rows.Scan(&p.Key, &p.Name, &p.LandingURL, &p.DefaultPDFPath, &p.Notes); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProvider looks up a provider by key.
func (s *SQLiteStorage) GetProvider(ctx context.Context, key string) (*storage.Provider, error) {
	row := s.db.QueryRowContext(ctx, `SELECT key, name, landing_url, default_pdf_path, notes FROM providers WHERE key = ?`, key)
	var p storage.Provider
	if err := row.Scan(&p.Key, &p.Name, &p.LandingURL, &p.DefaultPDFPath, &p.Notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// UpsertProvider inserts or updates a provider row.
func (s *SQLiteStorage) UpsertProvider(ctx context.Context, p storage.Provider) error {
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

// GetRatesSnapshot returns the most recent snapshot for a provider, if any.
func (s *SQLiteStorage) GetRatesSnapshot(ctx context.Context, provider string) (*storage.RatesSnapshot, error) {
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
	return &storage.RatesSnapshot{
		Provider:  provider,
		Payload:   payload,
		FetchedAt: t,
	}, nil
}

// SaveRatesSnapshot inserts a new snapshot row.
func (s *SQLiteStorage) SaveRatesSnapshot(ctx context.Context, snap storage.RatesSnapshot) error {
	if snap.FetchedAt.IsZero() {
		snap.FetchedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rates_snapshots (provider, payload, fetched_at)
		VALUES (?, ?, ?)
	`, snap.Provider, snap.Payload, snap.FetchedAt.Format(time.RFC3339Nano))
	return err
}
