-- Receipt chain starts at 057: 056_merchant_aliases shipped in v1.9.2.
-- +goose Up
ALTER TABLE system_settings ADD COLUMN receipt_api_token_encrypted TEXT NOT NULL DEFAULT '';

-- +goose Down
-- Forward-only migration.
