-- +goose Up
CREATE TABLE import_idempotency (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key TEXT NOT NULL,
    response_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (user_id, idempotency_key)
);
CREATE INDEX idx_import_idempotency_user_key ON import_idempotency (user_id, idempotency_key);

-- +goose Down
DROP TABLE import_idempotency;
