-- +goose Up
CREATE TABLE budget_alert_sent (
    budget_id           TEXT NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    period_start        TEXT NOT NULL,
    threshold_percent   INTEGER NOT NULL,
    sent_at             TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (budget_id, period_start, threshold_percent)
);

-- +goose Down
-- Forward-only migration.
