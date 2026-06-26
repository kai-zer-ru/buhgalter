-- +goose Up
CREATE TABLE recurring_operations (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type            TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    amount          INTEGER NOT NULL CHECK (amount > 0),
    description     TEXT,
    account_id      TEXT NOT NULL REFERENCES accounts(id),
    category_id     TEXT NOT NULL REFERENCES categories(id),
    subcategory_id  TEXT REFERENCES subcategories(id),
    period          TEXT NOT NULL CHECK (period IN ('week', 'two_weeks', 'month', 'year')),
    weekday         INTEGER CHECK (weekday BETWEEN 1 AND 7),
    day_of_month    INTEGER CHECK (day_of_month BETWEEN 1 AND 31),
    start_date      TEXT NOT NULL,
    time_local      TEXT NOT NULL DEFAULT '00:00',
    next_run_at     TEXT NOT NULL,
    last_run_at     TEXT,
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_recurring_ops_user ON recurring_operations(user_id);
CREATE INDEX idx_recurring_ops_due ON recurring_operations(user_id, active, next_run_at);

-- +goose Down
-- Forward-only migration.
