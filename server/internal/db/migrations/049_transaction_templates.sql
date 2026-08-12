-- +goose Up
CREATE TABLE transaction_templates (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL CHECK (type IN ('income', 'expense', 'transfer')),
    account_id      TEXT REFERENCES accounts(id) ON DELETE SET NULL,
    to_account_id   TEXT REFERENCES accounts(id) ON DELETE SET NULL,
    category_id     TEXT REFERENCES categories(id) ON DELETE SET NULL,
    subcategory_id  TEXT REFERENCES subcategories(id) ON DELETE SET NULL,
    amount          INTEGER CHECK (amount IS NULL OR amount > 0),
    description     TEXT,
    merchant_id     TEXT REFERENCES merchants(id) ON DELETE SET NULL,
    icon            TEXT,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_transaction_templates_user ON transaction_templates(user_id, sort_order, name);

CREATE TABLE transaction_template_tags (
    template_id  TEXT NOT NULL REFERENCES transaction_templates(id) ON DELETE CASCADE,
    tag_id       TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (template_id, tag_id)
);
CREATE INDEX idx_transaction_template_tags_tag ON transaction_template_tags(tag_id);

-- +goose Down
DROP INDEX IF EXISTS idx_transaction_template_tags_tag;
DROP TABLE IF EXISTS transaction_template_tags;
DROP INDEX IF EXISTS idx_transaction_templates_user;
DROP TABLE IF EXISTS transaction_templates;
