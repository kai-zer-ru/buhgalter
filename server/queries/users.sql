-- name: GetUserByID :one
SELECT id, login, COALESCE(display_name, '') AS display_name, is_admin, status, language, currency, timezone, theme
FROM users WHERE id = ?;

-- name: GetUserByLogin :one
SELECT id, login, COALESCE(display_name, '') AS display_name, is_admin, status, language, currency, timezone, theme, password_hash
FROM users WHERE login = ?;

-- name: InsertUser :exec
INSERT INTO users (id, login, password_hash, display_name, is_admin, status, theme)
VALUES (?, ?, ?, ?, ?, ?, 'system');

-- name: UpdateUserProfile :exec
UPDATE users
SET display_name = ?, language = ?, currency = ?, timezone = ?, theme = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateUserPassword :execrows
UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?;

-- name: UpdateUserStatus :exec
UPDATE users SET status = ?, updated_at = datetime('now') WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT id, login, COALESCE(display_name, '') AS display_name, is_admin, status, created_at
FROM users ORDER BY created_at ASC;

-- name: GetUserLogin :one
SELECT login FROM users WHERE id = ?;

-- name: GetUserLoginAndStatus :one
SELECT login, status FROM users WHERE id = ?;

-- name: GetUserAdminItem :one
SELECT id, login, COALESCE(display_name, '') AS display_name, is_admin, status, created_at
FROM users WHERE id = ?;

-- name: GetUserCreatedAtAndStatus :one
SELECT created_at, status FROM users WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountAdminUsers :one
SELECT COUNT(*) FROM users WHERE is_admin = 1;

-- name: ListUserIDs :many
SELECT id FROM users;

-- name: ListAdminUserIDs :many
SELECT id FROM users WHERE is_admin = 1;

-- name: GetUserLanguage :one
SELECT language FROM users WHERE id = ?;

-- name: GetUserIsAdmin :one
SELECT is_admin FROM users WHERE id = ?;

-- name: GetUserFormatting :one
SELECT language, timezone, currency FROM users WHERE id = ?;

-- name: ListUsersWithTimezone :many
SELECT id, timezone FROM users;
