-- +goose Up
CREATE TABLE budget_periods (
    id              TEXT PRIMARY KEY,
    budget_id       TEXT NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    period_start    TEXT NOT NULL,
    planned_amount  INTEGER NOT NULL CHECK (planned_amount > 0),
    rollover_amount INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (budget_id, period_start)
);

CREATE INDEX idx_budget_periods_budget ON budget_periods(budget_id);

-- +goose Down
-- Forward-only migration.
