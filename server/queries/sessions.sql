-- name: InsertSession :exec
INSERT INTO sessions (id, user_id, token_hash, last_activity, expires_at, ip_address, user_agent)
VALUES (?, ?, ?, datetime('now'), ?, ?, ?);

-- name: GetSessionByTokenHash :one
SELECT id, user_id, last_activity, expires_at
FROM sessions WHERE token_hash = ?;

-- name: GetSessionWithUser :one
SELECT s.id, s.user_id, s.last_activity, s.expires_at,
       u.login, COALESCE(u.display_name, '') AS display_name, u.is_admin, u.status,
       u.language, u.currency, u.timezone, u.theme
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = ?;

-- name: TouchSession :exec
UPDATE sessions
SET last_activity = datetime('now'), expires_at = ?
WHERE id = ?;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions WHERE token_hash = ?;

-- name: DeleteSessionByID :exec
DELETE FROM sessions WHERE id = ?;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions WHERE user_id = ?;
