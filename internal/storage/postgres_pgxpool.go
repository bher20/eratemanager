
package storage

import (
    "context"
    "errors"
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
    if err != nil { return nil, err }

    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil { return nil, err }

    return &PostgresPoolStorage{pool: pool}, nil
}

func (s *PostgresPoolStorage) Close() error {
    s.pool.Close()
    return nil
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
    }
    for _, stmt := range stmts {
        _, err := s.pool.Exec(ctx, stmt)
        if err != nil { return err }
    }
    return nil
}

func (s *PostgresPoolStorage) ListProviders(ctx context.Context) ([]Provider, error) {
    rows, err := s.pool.Query(ctx, `SELECT key, name, landing_url, default_pdf_path, notes FROM providers`)
    if err != nil { return nil, err }
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
