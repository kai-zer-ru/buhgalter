-- +goose Up
CREATE TABLE merchant_aliases (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    merchant_id TEXT NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL COLLATE NOCASE,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);
CREATE INDEX idx_merchant_aliases_merchant ON merchant_aliases(merchant_id);

-- +goose Down
DROP INDEX IF EXISTS idx_merchant_aliases_merchant;
DROP TABLE IF EXISTS merchant_aliases;
