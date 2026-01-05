package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			expires_at TIMESTAMPTZ,
			last_used_at TIMESTAMPTZ,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS casbin_rules (
			id SERIAL PRIMARY KEY,
			ptype TEXT NOT NULL,
			v0 TEXT,
			v1 TEXT,
			v2 TEXT,
			v3 TEXT,
			v4 TEXT,
			v5 TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS email_config (
			id TEXT PRIMARY KEY,
			provider TEXT,
			host TEXT,
			port INTEGER,
			username TEXT,
			password TEXT,
			from_address TEXT,
			from_name TEXT,
			api_key TEXT,
			encryption TEXT,
			enabled BOOLEAN,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ
		);`,
		`ALTER TABLE email_config ADD COLUMN IF NOT EXISTS encryption TEXT;`,
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

// Users

func (s *PostgresStorage) CreateUser(ctx context.Context, user User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, username, email, email_verified, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.ID, user.Username, user.Email, user.EmailVerified, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt)
	return err
}

func (s *PostgresStorage) GetUser(ctx context.Context, id string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, username, email, email_verified, password_hash, role, created_at, updated_at FROM users WHERE id = $1`, id)
	var u User
	var email sql.NullString
	if err := row.Scan(&u.ID, &u.Username, &email, &u.EmailVerified, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.Email = email.String
	return &u, nil
}

func (s *PostgresStorage) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, username, email, email_verified, password_hash, role, created_at, updated_at FROM users WHERE username = $1`, username)
	var u User
	var email sql.NullString
	if err := row.Scan(&u.ID, &u.Username, &email, &u.EmailVerified, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.Email = email.String
	return &u, nil
}

func (s *PostgresStorage) GetUserByEmail(ctx context.Context, emailAddr string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, username, email, email_verified, password_hash, role, created_at, updated_at FROM users WHERE email = $1`, emailAddr)
	var u User
	var email sql.NullString
	if err := row.Scan(&u.ID, &u.Username, &email, &u.EmailVerified, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.Email = email.String
	return &u, nil
}

func (s *PostgresStorage) UpdateUser(ctx context.Context, user User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET username = $1, email = $2, email_verified = $3, password_hash = $4, role = $5, updated_at = $6 WHERE id = $7
	`, user.Username, user.Email, user.EmailVerified, user.PasswordHash, user.Role, user.UpdatedAt, user.ID)
	return err
}

func (s *PostgresStorage) DeleteUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (s *PostgresStorage) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, username, email, email_verified, password_hash, role, created_at, updated_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		var email sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &email, &u.EmailVerified, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.Email = email.String
		out = append(out, u)
	}
	return out, rows.Err()
}

// Tokens

func (s *PostgresStorage) CreateToken(ctx context.Context, token Token) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tokens (id, user_id, name, token_hash, role, created_at, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, token.ID, token.UserID, token.Name, token.TokenHash, token.Role, token.CreatedAt, token.ExpiresAt, token.LastUsedAt)
	return err
}

func (s *PostgresStorage) GetToken(ctx context.Context, id string) (*Token, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE id = $1`, id)
	var t Token
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.CreatedAt, &t.ExpiresAt, &t.LastUsedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (s *PostgresStorage) GetTokenByHash(ctx context.Context, hash string) (*Token, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE token_hash = $1`, hash)
	var t Token
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.CreatedAt, &t.ExpiresAt, &t.LastUsedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (s *PostgresStorage) ListTokens(ctx context.Context, userID string) ([]Token, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE user_id = $1`, userID)
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

func (s *PostgresStorage) DeleteToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tokens WHERE id = $1`, id)
	return err
}

func (s *PostgresStorage) UpdateTokenLastUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tokens SET last_used_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (s *PostgresStorage) LoadCasbinRules(ctx context.Context) ([]CasbinRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ptype, v0, v1, v2, v3, v4, v5 FROM casbin_rules`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CasbinRule
	for rows.Next() {
		var r CasbinRule
		var v0, v1, v2, v3, v4, v5 sql.NullString
		if err := rows.Scan(&r.PType, &v0, &v1, &v2, &v3, &v4, &v5); err != nil {
			return nil, err
		}
		r.V0 = v0.String
		r.V1 = v1.String
		r.V2 = v2.String
		r.V3 = v3.String
		r.V4 = v4.String
		r.V5 = v5.String
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) AddCasbinRule(ctx context.Context, rule CasbinRule) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO casbin_rules (ptype, v0, v1, v2, v3, v4, v5) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rule.PType, rule.V0, rule.V1, rule.V2, rule.V3, rule.V4, rule.V5)
	return err
}

func (s *PostgresStorage) RemoveCasbinRule(ctx context.Context, rule CasbinRule) error {
	query := "DELETE FROM casbin_rules WHERE ptype = $1"
	args := []interface{}{rule.PType}
	idx := 2

	if rule.V0 != "" {
		query += " AND v0 = $" + fmt.Sprintf("%d", idx)
		args = append(args, rule.V0)
		idx++
	}
	if rule.V1 != "" {
		query += " AND v1 = $" + fmt.Sprintf("%d", idx)
		args = append(args, rule.V1)
		idx++
	}
	if rule.V2 != "" {
		query += " AND v2 = $" + fmt.Sprintf("%d", idx)
		args = append(args, rule.V2)
		idx++
	}
	if rule.V3 != "" {
		query += " AND v3 = $" + fmt.Sprintf("%d", idx)
		args = append(args, rule.V3)
		idx++
	}
	if rule.V4 != "" {
		query += " AND v4 = $" + fmt.Sprintf("%d", idx)
		args = append(args, rule.V4)
		idx++
	}
	if rule.V5 != "" {
		query += " AND v5 = $" + fmt.Sprintf("%d", idx)
		args = append(args, rule.V5)
		idx++
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *PostgresStorage) GetEmailConfig(ctx context.Context) (*EmailConfig, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, provider, host, port, username, password, from_address, from_name, api_key, encryption, enabled, created_at, updated_at
		FROM email_config
		LIMIT 1
	`)
	var c EmailConfig
	var encryption sql.NullString
	err := row.Scan(
		&c.ID, &c.Provider, &c.Host, &c.Port, &c.Username, &c.Password,
		&c.FromAddress, &c.FromName, &c.APIKey, &encryption, &c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if encryption.Valid {
		c.Encryption = encryption.String
	}
	return &c, nil
}

func (s *PostgresStorage) SaveEmailConfig(ctx context.Context, config EmailConfig) error {
	// Check if exists
	existing, err := s.GetEmailConfig(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	if existing == nil {
		// Insert
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO email_config (id, provider, host, port, username, password, from_address, from_name, api_key, encryption, enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, config.ID, config.Provider, config.Host, config.Port, config.Username, config.Password, config.FromAddress, config.FromName, config.APIKey, config.Encryption, config.Enabled, now, now)
	} else {
		// Update
		_, err = s.db.ExecContext(ctx, `
			UPDATE email_config
			SET provider=$1, host=$2, port=$3, username=$4, password=$5, from_address=$6, from_name=$7, api_key=$8, encryption=$9, enabled=$10, updated_at=$11
			WHERE id=$12
		`, config.Provider, config.Host, config.Port, config.Username, config.Password, config.FromAddress, config.FromName, config.APIKey, config.Encryption, config.Enabled, now, existing.ID)
	}
	return err
}
