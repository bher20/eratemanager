-- +goose Up
ALTER TABLE users ADD COLUMN onboarding_completed BOOLEAN DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROP COLUMN onboarding_completed;
