-- +goose Up
ALTER TABLE transactions ADD COLUMN subscription_id TEXT REFERENCES subscriptions(id) ON DELETE SET NULL;
CREATE INDEX idx_tx_subscription ON transactions(subscription_id);

-- +goose Down
-- Forward-only migration.
