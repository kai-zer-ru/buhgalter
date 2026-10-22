-- name: ListMerchantsByUser :many
SELECT id, user_id, name, icon, created_at
FROM merchants
WHERE user_id = ?
ORDER BY name COLLATE NOCASE ASC;

-- name: GetMerchantByID :one
SELECT id, user_id, name, icon, created_at
FROM merchants
WHERE id = ? AND user_id = ?;

-- name: GetMerchantByName :one
SELECT id, user_id, name, icon, created_at
FROM merchants
WHERE user_id = sqlc.arg(user_id) AND lower(name) = lower(sqlc.arg(name));

-- name: InsertMerchant :exec
INSERT INTO merchants (id, user_id, name, icon, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateMerchant :execrows
UPDATE merchants
SET name = ?, icon = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteMerchant :execrows
DELETE FROM merchants
WHERE id = ? AND user_id = ?;
