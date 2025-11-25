-- 001_create_providers.sql
CREATE TABLE IF NOT EXISTS providers (
    key TEXT PRIMARY KEY,
    name TEXT,
    landing_url TEXT,
    default_pdf_path TEXT,
    notes TEXT
);
