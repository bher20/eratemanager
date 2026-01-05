-- +goose Up
ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN email_verified;
