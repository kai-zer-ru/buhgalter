-- Note: on DBs that already applied legacy 043_receipt_scan_jobs, this file is skipped;
-- table is created by 046_subscriptions_ensure.sql instead.
-- +goose Up
CREATE TABLE subscriptions (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT,
    icon            TEXT,
    website_url     TEXT,
    amount          INTEGER NOT NULL CHECK (amount > 0),
    account_id      TEXT NOT NULL REFERENCES accounts(id),
    subcategory_id  TEXT REFERENCES subcategories(id),
    period          TEXT NOT NULL CHECK (period IN ('week', 'two_weeks', 'month', 'quarter', 'half_year', 'year')),
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

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_due ON subscriptions(user_id, active, next_run_at);
CREATE INDEX idx_subscriptions_subcategory ON subscriptions(subcategory_id);

-- +goose Down
-- Forward-only migration.
