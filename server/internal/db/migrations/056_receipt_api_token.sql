-- +goose Up
ALTER TABLE system_settings ADD COLUMN receipt_api_token_encrypted TEXT NOT NULL DEFAULT '';

-- +goose Down
-- Forward-only migration.
