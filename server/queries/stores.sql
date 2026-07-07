-- name: ListStoresByUser :many
SELECT id, name, created_at
FROM stores
WHERE user_id = ?
ORDER BY name COLLATE NOCASE ASC;

-- name: GetStoreByNormalizedName :one
SELECT id, user_id, name, name_normalized, created_at
FROM stores
WHERE user_id = ? AND name_normalized = ?;

-- name: InsertStore :exec
INSERT INTO stores (id, user_id, name, name_normalized, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetStoreByID :one
SELECT id, user_id, name, name_normalized, created_at
FROM stores
WHERE id = ? AND user_id = ?;
