-- +goose Up
ALTER TABLE notification_settings ADD COLUMN keyboard_telegram TEXT NOT NULL DEFAULT '{"rows":[]}';
ALTER TABLE notification_settings ADD COLUMN keyboard_max TEXT NOT NULL DEFAULT '{"rows":[]}';
ALTER TABLE notification_templates ADD COLUMN keyboard_telegram TEXT NOT NULL DEFAULT '{"rows":[]}';
ALTER TABLE notification_templates ADD COLUMN keyboard_max TEXT NOT NULL DEFAULT '{"rows":[]}';

-- +goose Down
-- Forward-only migration.
