-- +goose Up
CREATE TABLE notification_templates (
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trigger_type    TEXT NOT NULL CHECK (trigger_type IN (
                        'debt_overdue', 'debt_due_soon', 'credit_payment', 'planned_operation', 'test'
                    )),
    template        TEXT NOT NULL,
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, trigger_type)
);

-- +goose Down
DROP TABLE notification_templates;
