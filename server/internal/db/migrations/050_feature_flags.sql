-- +goose Up
CREATE TABLE feature_flags (
    key         TEXT PRIMARY KEY,
    enabled     INTEGER NOT NULL CHECK (enabled IN (0, 1)),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO feature_flags (key, enabled) VALUES
    ('registration', 0),
    ('debts', 1),
    ('credits', 1),
    ('budget', 1),
    ('subscriptions', 1),
    ('recurring', 1),
    ('balance_maintenance', 1),
    ('import_export', 1),
    ('stats', 1),
    ('notifications', 1),
    ('merchants_tags', 1),
    ('transaction_templates', 1);

INSERT INTO feature_flags (key, enabled, updated_at)
SELECT 'registration', registration_enabled, datetime('now')
FROM system_settings
WHERE id = 1
ON CONFLICT(key) DO UPDATE SET
    enabled = excluded.enabled,
    updated_at = excluded.updated_at;

ALTER TABLE system_settings DROP COLUMN registration_enabled;

-- +goose Down
ALTER TABLE system_settings ADD COLUMN registration_enabled INTEGER NOT NULL DEFAULT 0;

UPDATE system_settings
SET registration_enabled = COALESCE(
    (SELECT enabled FROM feature_flags WHERE key = 'registration'),
    0
)
WHERE id = 1;

DROP TABLE IF EXISTS feature_flags;
