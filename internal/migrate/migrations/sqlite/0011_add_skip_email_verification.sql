-- +goose Up
ALTER TABLE users ADD COLUMN skip_email_verification BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN skip_email_verification;
