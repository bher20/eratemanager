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
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TEXT NOT NULL,
			expires_at TEXT,
			last_used_at TEXT,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
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
			created_at TEXT,
			updated_at TEXT
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	// Add encryption column if not exists (SQLite doesn't support IF NOT EXISTS for ADD COLUMN)
	// We can ignore the error if it fails (column already exists)
	s.db.ExecContext(ctx, `ALTER TABLE email_config ADD COLUMN encryption TEXT`)

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

func (s *SQLiteStorage) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *SQLiteStorage) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now().Format(time.RFC3339))
	return err
}

// Users

func (s *SQLiteStorage) CreateUser(ctx context.Context, user User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, username, email, email_verified, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Username, user.Email, user.EmailVerified, user.PasswordHash, user.Role, user.CreatedAt.Format(time.RFC3339), user.UpdatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStorage) GetUser(ctx context.Context, id string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, username, email, email_verified, password_hash, role, created_at, updated_at FROM users WHERE id = ?`, id)
	var u User
	var email sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(&u.ID, &u.Username, &email, &u.EmailVerified, &u.PasswordHash, &u.Role, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.Email = email.String
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}

func (s *SQLiteStorage) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, username, email, email_verified, password_hash, role, created_at, updated_at FROM users WHERE username = ?`, username)
	var u User
	var email sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(&u.ID, &u.Username, &email, &u.EmailVerified, &u.PasswordHash, &u.Role, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.Email = email.String
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}

func (s *SQLiteStorage) GetUserByEmail(ctx context.Context, emailAddr string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, username, email, email_verified, password_hash, role, created_at, updated_at FROM users WHERE email = ?`, emailAddr)
	var u User
	var email sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(&u.ID, &u.Username, &email, &u.EmailVerified, &u.PasswordHash, &u.Role, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.Email = email.String
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}

func (s *SQLiteStorage) UpdateUser(ctx context.Context, user User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET username = ?, email = ?, email_verified = ?, password_hash = ?, role = ?, updated_at = ? WHERE id = ?
	`, user.Username, user.Email, user.EmailVerified, user.PasswordHash, user.Role, user.UpdatedAt.Format(time.RFC3339), user.ID)
	return err
}

func (s *SQLiteStorage) DeleteUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *SQLiteStorage) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, username, email, email_verified, password_hash, role, created_at, updated_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		var email sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&u.ID, &u.Username, &email, &u.EmailVerified, &u.PasswordHash, &u.Role, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		u.Email = email.String
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		out = append(out, u)
	}
	return out, rows.Err()
}

// Tokens

func (s *SQLiteStorage) CreateToken(ctx context.Context, token Token) error {
	var expiresAt, lastUsedAt *string
	if token.ExpiresAt != nil {
		t := token.ExpiresAt.Format(time.RFC3339)
		expiresAt = &t
	}
	if token.LastUsedAt != nil {
		t := token.LastUsedAt.Format(time.RFC3339)
		lastUsedAt = &t
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tokens (id, user_id, name, token_hash, role, created_at, expires_at, last_used_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, token.ID, token.UserID, token.Name, token.TokenHash, token.Role, token.CreatedAt.Format(time.RFC3339), expiresAt, lastUsedAt)
	return err
}

func (s *SQLiteStorage) GetToken(ctx context.Context, id string) (*Token, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE id = ?`, id)
	var t Token
	var createdAt string
	var expiresAt, lastUsedAt *string
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &createdAt, &expiresAt, &lastUsedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if expiresAt != nil {
		ts, _ := time.Parse(time.RFC3339, *expiresAt)
		t.ExpiresAt = &ts
	}
	if lastUsedAt != nil {
		ts, _ := time.Parse(time.RFC3339, *lastUsedAt)
		t.LastUsedAt = &ts
	}
	return &t, nil
}

func (s *SQLiteStorage) GetTokenByHash(ctx context.Context, hash string) (*Token, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE token_hash = ?`, hash)
	var t Token
	var createdAt string
	var expiresAt, lastUsedAt *string
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &createdAt, &expiresAt, &lastUsedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if expiresAt != nil {
		ts, _ := time.Parse(time.RFC3339, *expiresAt)
		t.ExpiresAt = &ts
	}
	if lastUsedAt != nil {
		ts, _ := time.Parse(time.RFC3339, *lastUsedAt)
		t.LastUsedAt = &ts
	}
	return &t, nil
}

func (s *SQLiteStorage) ListTokens(ctx context.Context, userID string) ([]Token, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, name, token_hash, role, created_at, expires_at, last_used_at FROM tokens WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Token
	for rows.Next() {
		var t Token
		var createdAt string
		var expiresAt, lastUsedAt *string
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &createdAt, &expiresAt, &lastUsedAt); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if expiresAt != nil {
			ts, _ := time.Parse(time.RFC3339, *expiresAt)
			t.ExpiresAt = &ts
		}
		if lastUsedAt != nil {
			ts, _ := time.Parse(time.RFC3339, *lastUsedAt)
			t.LastUsedAt = &ts
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLiteStorage) DeleteToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tokens WHERE id = ?`, id)
	return err
}

func (s *SQLiteStorage) UpdateTokenLastUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tokens SET last_used_at = ? WHERE id = ?`, time.Now().Format(time.RFC3339), id)
	return err
}

func (s *SQLiteStorage) LoadCasbinRules(ctx context.Context) ([]CasbinRule, error) {
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

func (s *SQLiteStorage) AddCasbinRule(ctx context.Context, rule CasbinRule) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO casbin_rules (ptype, v0, v1, v2, v3, v4, v5) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rule.PType, rule.V0, rule.V1, rule.V2, rule.V3, rule.V4, rule.V5)
	return err
}

func (s *SQLiteStorage) RemoveCasbinRule(ctx context.Context, rule CasbinRule) error {
	query := `DELETE FROM casbin_rules WHERE ptype = ?`
	args := []interface{}{rule.PType}

	if rule.V0 != "" {
		query += " AND v0 = ?"
		args = append(args, rule.V0)
	}
	if rule.V1 != "" {
		query += " AND v1 = ?"
		args = append(args, rule.V1)
	}
	if rule.V2 != "" {
		query += " AND v2 = ?"
		args = append(args, rule.V2)
	}
	if rule.V3 != "" {
		query += " AND v3 = ?"
		args = append(args, rule.V3)
	}
	if rule.V4 != "" {
		query += " AND v4 = ?"
		args = append(args, rule.V4)
	}
	if rule.V5 != "" {
		query += " AND v5 = ?"
		args = append(args, rule.V5)
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SQLiteStorage) GetEmailConfig(ctx context.Context) (*EmailConfig, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, provider, host, port, username, password, from_address, from_name, api_key, encryption, enabled, created_at, updated_at
		FROM email_config
		LIMIT 1
	`)
	var c EmailConfig
	var encryption sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(
		&c.ID, &c.Provider, &c.Host, &c.Port, &c.Username, &c.Password,
		&c.FromAddress, &c.FromName, &c.APIKey, &encryption, &c.Enabled, &createdAt, &updatedAt,
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
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &c, nil
}

func (s *SQLiteStorage) SaveEmailConfig(ctx context.Context, config EmailConfig) error {
	// Check if exists
	existing, err := s.GetEmailConfig(ctx)
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	if existing == nil {
		// Insert
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO email_config (id, provider, host, port, username, password, from_address, from_name, api_key, encryption, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, config.ID, config.Provider, config.Host, config.Port, config.Username, config.Password, config.FromAddress, config.FromName, config.APIKey, config.Encryption, config.Enabled, now, now)
	} else {
		// Update
		_, err = s.db.ExecContext(ctx, `
			UPDATE email_config
			SET provider=?, host=?, port=?, username=?, password=?, from_address=?, from_name=?, api_key=?, encryption=?, enabled=?, updated_at=?
			WHERE id=?
		`, config.Provider, config.Host, config.Port, config.Username, config.Password, config.FromAddress, config.FromName, config.APIKey, config.Encryption, config.Enabled, now, existing.ID)
	}
	return err
}
