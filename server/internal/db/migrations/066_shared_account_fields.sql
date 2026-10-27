-- +goose Up
ALTER TABLE transactions ADD COLUMN created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL;
UPDATE transactions SET created_by_user_id = user_id;
ALTER TABLE accounts ADD COLUMN original_owner_user_id TEXT REFERENCES users(id) ON DELETE SET NULL;

-- +goose Down
-- Forward-only migration.
