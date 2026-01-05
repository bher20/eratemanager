-- +goose Up
ALTER TABLE users ADD COLUMN skip_email_verification BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROP COLUMN skip_email_verification;
