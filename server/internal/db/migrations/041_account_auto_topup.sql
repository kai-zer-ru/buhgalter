-- +goose Up
ALTER TABLE accounts ADD COLUMN auto_topup_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN auto_topup_threshold INTEGER;
ALTER TABLE accounts ADD COLUMN auto_topup_target INTEGER;
ALTER TABLE accounts ADD COLUMN auto_topup_source_account_id TEXT REFERENCES accounts(id);

ALTER TABLE notification_settings ADD COLUMN trigger_auto_topup_disabled INTEGER NOT NULL DEFAULT 1;

CREATE TABLE notification_templates_new (
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trigger_type    TEXT NOT NULL CHECK (trigger_type IN (
                        'debt_overdue', 'debt_due_soon', 'credit_payment', 'planned_operation',
                        'balance_shortfall', 'budget_threshold', 'auto_topup_disabled',
                        'user_registration', 'password_reset', 'test'
                    )),
    template        TEXT NOT NULL,
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, trigger_type)
);
INSERT INTO notification_templates_new (user_id, trigger_type, template, updated_at)
SELECT user_id, trigger_type, template, updated_at
FROM notification_templates;
DROP TABLE notification_templates;
ALTER TABLE notification_templates_new RENAME TO notification_templates;

-- +goose Down
-- Forward-only migration.
