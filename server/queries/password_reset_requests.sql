-- name: UpsertPasswordResetRequest :exec
INSERT INTO password_reset_requests (id, user_id, created_at, dismissed_at)
VALUES (?, ?, ?, NULL)
ON CONFLICT(user_id) DO UPDATE SET
    created_at = excluded.created_at,
    dismissed_at = NULL;

-- name: ListPendingPasswordResetRequests :many
SELECT r.id, r.user_id, u.login, COALESCE(u.display_name, '') AS display_name, r.created_at
FROM password_reset_requests r
JOIN users u ON u.id = r.user_id
WHERE r.dismissed_at IS NULL
ORDER BY r.created_at ASC;

-- name: DismissPasswordResetRequest :execrows
UPDATE password_reset_requests
SET dismissed_at = ?
WHERE id = ? AND dismissed_at IS NULL;

-- name: DismissPasswordResetRequestsForUser :exec
UPDATE password_reset_requests
SET dismissed_at = ?
WHERE user_id = ? AND dismissed_at IS NULL;
