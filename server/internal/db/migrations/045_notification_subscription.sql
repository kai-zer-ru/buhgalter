-- +goose Up
ALTER TABLE notification_settings ADD COLUMN trigger_subscription INTEGER NOT NULL DEFAULT 1;
ALTER TABLE notification_settings ADD COLUMN subscription_days_before INTEGER NOT NULL DEFAULT 1;

-- +goose Down
-- Forward-only migration.
