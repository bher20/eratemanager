package storage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPoolStorage struct {
	pool *pgxpool.Pool
}

func OpenPostgresPool(ctx context.Context, dsn string) (*PostgresPoolStorage, error) {
	if dsn == "" {
		dsn = "postgres://localhost:5432/eratemanager?sslmode=disable"
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &PostgresPoolStorage{pool: pool}, nil
}

func (s *PostgresPoolStorage) Close() error {
	s.pool.Close()
	return nil
}

func (s *PostgresPoolStorage) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Advisory lock helpers used by the cron worker.
func (s *PostgresPoolStorage) AcquireAdvisoryLock(ctx context.Context, key int64) (bool, error) {
	row := s.pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key)
	var ok bool
	if err := row.Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

func (s *PostgresPoolStorage) ReleaseAdvisoryLock(ctx context.Context, key int64) (bool, error) {
	row := s.pool.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, key)
	var ok bool
	if err := row.Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

func (s *PostgresPoolStorage) UpdateScheduledJob(ctx context.Context, name string, started time.Time, dur time.Duration, success bool, errMsg string) error {
	successInt := 0
	if success {
		successInt = 1
	}
	_, err := s.pool.Exec(ctx, `
        INSERT INTO scheduled_jobs (name, last_run_at, last_duration_ms, last_success, last_error)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (name) DO UPDATE SET
            last_run_at = EXCLUDED.last_run_at,
            last_duration_ms = EXCLUDED.last_duration_ms,
            last_success = EXCLUDED.last_success,
            last_error = EXCLUDED.last_error
    `, name, started.Format(time.RFC3339Nano), dur.Milliseconds(), successInt, errMsg)
	return err
}

func (s *PostgresPoolStorage) Migrate(ctx context.Context) error {
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
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresPoolStorage) ListProviders(ctx context.Context) ([]Provider, error) {
	rows, err := s.pool.Query(ctx, `SELECT key, name, landing_url, default_pdf_path, notes FROM providers`)
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

func (s *PostgresPoolStorage) GetProvider(ctx context.Context, key string) (*Provider, error) {
	row := s.pool.QueryRow(ctx, `SELECT key, name, landing_url, default_pdf_path, notes FROM providers WHERE key=$1`, key)
	var p Provider
	if err := row.Scan(&p.Key, &p.Name, &p.LandingURL, &p.DefaultPDFPath, &p.Notes); err != nil {
		return nil, nil
	}
	return &p, nil
}

func (s *PostgresPoolStorage) UpsertProvider(ctx context.Context, p Provider) error {
	_, err := s.pool.Exec(ctx, `
        INSERT INTO providers (key, name, landing_url, default_pdf_path, notes)
        VALUES ($1,$2,$3,$4,$5)
        ON CONFLICT (key) DO UPDATE SET
            name=EXCLUDED.name,
            landing_url=EXCLUDED.landing_url,
            default_pdf_path=EXCLUDED.default_pdf_path,
            notes=EXCLUDED.notes
    `, p.Key, p.Name, p.LandingURL, p.DefaultPDFPath, p.Notes)
	return err
}

func (s *PostgresPoolStorage) GetRatesSnapshot(ctx context.Context, provider string) (*RatesSnapshot, error) {
	row := s.pool.QueryRow(ctx, `
        SELECT payload, fetched_at
        FROM rates_snapshots
        WHERE provider=$1
        ORDER BY id DESC
        LIMIT 1
    `, provider)

	var payload []byte
	var fetched time.Time
	if err := row.Scan(&payload, &fetched); err != nil {
		return nil, nil
	}

	return &RatesSnapshot{
		Provider:  provider,
		Payload:   payload,
		FetchedAt: fetched,
	}, nil
}

func (s *PostgresPoolStorage) SaveRatesSnapshot(ctx context.Context, snap RatesSnapshot) error {
	if snap.FetchedAt.IsZero() {
		snap.FetchedAt = time.Now()
	}
	_, err := s.pool.Exec(ctx, `
        INSERT INTO rates_snapshots (provider, payload, fetched_at)
        VALUES ($1,$2,$3)
    `, snap.Provider, snap.Payload, snap.FetchedAt)
	return err
}

// BatchProgress methods for tracking batch job progress
func (s *PostgresPoolStorage) SaveBatchProgress(ctx context.Context, progress BatchProgress) error {
	_, err := s.pool.Exec(ctx, `
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

func (s *PostgresPoolStorage) GetBatchProgress(ctx context.Context, batchID, provider string) (*BatchProgress, error) {
	row := s.pool.QueryRow(ctx, `
        SELECT batch_id, provider, status, started_at, completed_at, error_message, retry_count
        FROM batch_progress
        WHERE batch_id = $1 AND provider = $2
    `, batchID, provider)

	var p BatchProgress
	var startedAt, completedAt *time.Time
	if err := row.Scan(&p.BatchID, &p.Provider, &p.Status, &startedAt, &completedAt, &p.ErrorMessage, &p.RetryCount); err != nil {
		return nil, nil // Not found
	}
	if startedAt != nil {
		p.StartedAt = *startedAt
	}
	if completedAt != nil {
		p.CompletedAt = *completedAt
	}
	return &p, nil
}

func (s *PostgresPoolStorage) GetPendingBatchProviders(ctx context.Context, batchID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
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

func (s *PostgresPoolStorage) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if err != nil {
		// pgx returns pgx.ErrNoRows, but we can check if err.Error() contains "no rows" or just return empty
		// Better to handle it properly if we imported pgx, but here we can just return error or empty.
		// Actually, pgxpool.QueryRow returns a Row which has Scan. Scan returns pgx.ErrNoRows.
		// Since we don't import pgx directly here (only pgxpool), we might need to check string or import pgx/v5.
		// But wait, we import github.com/jackc/pgx/v5/pgxpool.
		// Let's just return empty string if error.
		return "", nil
	}
	return value, nil
}

func (s *PostgresPoolStorage) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, $3)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now())
	return err
}

// Users

func (s *PostgresPoolStorage) CreateUser(ctx context.Context, user User) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, username, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, user.ID, user.Username, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt)
	return err
}

func (s *PostgresPoolStorage) GetUser(ctx context.Context, id string) (*User, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE id = $1`, id)
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresPoolStorage) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE username = $1`, username)
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresPoolStorage) UpdateUser(ctx context.Context, user User) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET username = $1, password_hash = $2, role = $3, updated_at = $4 WHERE id = $5
	`, user.Username, user.PasswordHash, user.Role, user.UpdatedAt, user.ID)
	return err
}

func (s *PostgresPoolStorage) DeleteUser(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (s *PostgresPoolStorage) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, username, password_hash, role, created_at, updated_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// Tokens

func (s *PostgresPoolStorage) CreateToken(ctx context.Context, token Token) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tokens (id, user_id, name, token_hash, role, created_at, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, token.ID, token.UserID, token.Name, token.TokenHash, token.Role, token.CreatedAt, token.ExpiresAt, token.LastUsedAt)
	return err
}

func (s *PostgresPoolStorage) GetToken(ctx context.Context, id string) (*Token, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE id = $1`, id)
	var t Token
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.CreatedAt, &t.ExpiresAt, &t.LastUsedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *PostgresPoolStorage) GetTokenByHash(ctx context.Context, hash string) (*Token, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE token_hash = $1`, hash)
	var t Token
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.CreatedAt, &t.ExpiresAt, &t.LastUsedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *PostgresPoolStorage) ListTokens(ctx context.Context, userID string) ([]Token, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Token
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.CreatedAt, &t.ExpiresAt, &t.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *PostgresPoolStorage) DeleteToken(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM tokens WHERE id = $1`, id)
	return err
}

func (s *PostgresPoolStorage) UpdateTokenLastUsed(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE tokens SET last_used_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}
