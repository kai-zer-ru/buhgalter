-- +goose Up
CREATE TABLE merchants (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL COLLATE NOCASE,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);

CREATE TABLE tags (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL COLLATE NOCASE,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);

CREATE TABLE transaction_tags (
    transaction_id  TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tag_id          TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (transaction_id, tag_id)
);
CREATE INDEX idx_transaction_tags_tag ON transaction_tags(tag_id);

ALTER TABLE transactions ADD COLUMN merchant_id TEXT REFERENCES merchants(id) ON DELETE SET NULL;
CREATE INDEX idx_tx_merchant ON transactions(merchant_id);

-- +goose Down
DROP INDEX IF EXISTS idx_tx_merchant;
-- SQLite cannot DROP COLUMN portably in older versions; leave merchant_id on down.
DROP TABLE IF EXISTS transaction_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS merchants;
