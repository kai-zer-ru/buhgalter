-- +goose Up
CREATE TABLE budgets (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    scope           TEXT NOT NULL CHECK (scope IN ('category', 'subcategory', 'all_expense', 'all_income')),
    category_id     TEXT REFERENCES categories(id),
    subcategory_id  TEXT REFERENCES subcategories(id),
    amount          INTEGER NOT NULL CHECK (amount > 0),
    period          TEXT NOT NULL DEFAULT 'month' CHECK (period = 'month'),
    account_id      TEXT REFERENCES accounts(id),
    rollover        INTEGER NOT NULL DEFAULT 0,
    alert_at_percent INTEGER NOT NULL DEFAULT 90,
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_budgets_user ON budgets(user_id);

CREATE UNIQUE INDEX idx_budgets_active_unique ON budgets(
    user_id,
    scope,
    IFNULL(category_id, ''),
    IFNULL(subcategory_id, ''),
    IFNULL(account_id, '')
) WHERE is_active = 1;

-- +goose Down
-- Forward-only migration.
