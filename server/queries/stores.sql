-- name: ListStoresByUser :many
SELECT id, user_id, name, created_at
FROM stores
WHERE user_id IN (sqlc.slice('owner_ids'))
ORDER BY name COLLATE NOCASE ASC;

-- name: GetStoreByNormalizedName :one
SELECT id, user_id, name, name_normalized, created_at
FROM stores
WHERE user_id = ? AND name_normalized = ?;

-- name: InsertStore :exec
INSERT INTO stores (id, user_id, name, name_normalized, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetStoreUserID :one
SELECT user_id FROM stores WHERE id = ?;

-- name: GetStoreByID :one
SELECT id, user_id, name, name_normalized, created_at
FROM stores
WHERE id = sqlc.arg('id') AND user_id IN (sqlc.slice('owner_ids'));
