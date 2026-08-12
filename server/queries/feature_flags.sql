-- name: ListFeatureFlags :many
SELECT key, enabled, updated_at FROM feature_flags;

-- name: GetFeatureFlag :one
SELECT key, enabled, updated_at FROM feature_flags WHERE key = ?;

-- name: UpsertFeatureFlag :exec
INSERT INTO feature_flags (key, enabled, updated_at)
VALUES (?, ?, datetime('now'))
ON CONFLICT(key) DO UPDATE SET
    enabled = excluded.enabled,
    updated_at = excluded.updated_at;
