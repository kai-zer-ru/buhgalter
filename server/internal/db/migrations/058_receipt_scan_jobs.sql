-- +goose Up
-- IF NOT EXISTS: scan_chek experiment databases may already have this table.
CREATE TABLE IF NOT EXISTS receipt_scan_jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'done', 'failed')),
    error_message TEXT,
    result_json TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_receipt_scan_jobs_user_created ON receipt_scan_jobs (user_id, created_at DESC);

-- +goose Down
DROP TABLE receipt_scan_jobs;
