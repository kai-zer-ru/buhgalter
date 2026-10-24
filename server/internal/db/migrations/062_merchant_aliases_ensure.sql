-- +goose Up
-- Repair: version 56 may already be recorded on a scan_chek database
-- (receipt migrations used 056–060 before v1.9.2 took 056 for merchant_aliases).
-- Those databases skip 056_merchant_aliases.sql.
CREATE TABLE IF NOT EXISTS merchant_aliases (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    merchant_id TEXT NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL COLLATE NOCASE,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_merchant_aliases_merchant ON merchant_aliases(merchant_id);

-- +goose Down
-- Forward-only migration.
