-- name: GetAPITokenByHash :one
SELECT id, user_id, expires_at FROM api_tokens WHERE token_hash = ?;

-- name: GetAPITokenExpiresAt :one
SELECT expires_at FROM api_tokens WHERE token_hash = ?;

-- name: TouchAPITokenByID :exec
UPDATE api_tokens SET last_used_at = datetime('now') WHERE id = ?;

-- name: TouchAPITokenByHash :exec
UPDATE api_tokens SET last_used_at = datetime('now') WHERE token_hash = ?;

-- name: ListAPITokensByUser :many
SELECT id, name, token_prefix, expires_at, last_used_at, created_at
FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC;

-- name: InsertAPIToken :exec
INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, expires_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetAPITokenCreatedAt :one
SELECT created_at FROM api_tokens WHERE id = ?;

-- name: GetAPITokenMeta :one
SELECT name, token_prefix FROM api_tokens WHERE id = ? AND user_id = ?;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = ? AND user_id = ?;
