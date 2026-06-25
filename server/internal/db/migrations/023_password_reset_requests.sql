-- +goose Up
CREATE TABLE password_reset_requests (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    dismissed_at TEXT
);

CREATE INDEX idx_password_reset_requests_pending ON password_reset_requests(user_id)
WHERE dismissed_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS password_reset_requests;
