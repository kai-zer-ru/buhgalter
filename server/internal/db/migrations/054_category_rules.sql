-- +goose Up
CREATE TABLE category_rules (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind                TEXT NOT NULL CHECK (kind IN ('category_account', 'text_category')),
    pattern             TEXT,
    category_id         TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    subcategory_id      TEXT REFERENCES subcategories(id) ON DELETE SET NULL,
    default_account_id  TEXT REFERENCES accounts(id) ON DELETE SET NULL,
    priority            INTEGER NOT NULL DEFAULT 0,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_category_rules_user_kind ON category_rules(user_id, kind, priority, id);

-- +goose Down
DROP INDEX IF EXISTS idx_category_rules_user_kind;
DROP TABLE IF EXISTS category_rules;
