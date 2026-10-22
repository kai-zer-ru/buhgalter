-- name: ListTagsByUser :many
SELECT id, user_id, name, created_at
FROM tags
WHERE user_id = ?
  AND (? = '' OR name LIKE '%' || ? || '%')
ORDER BY name COLLATE NOCASE ASC;

-- name: GetTagByID :one
SELECT id, user_id, name, created_at
FROM tags
WHERE id = ? AND user_id = ?;

-- name: GetTagByName :one
SELECT id, user_id, name, created_at
FROM tags
WHERE user_id = sqlc.arg(user_id) AND lower(name) = lower(sqlc.arg(name));

-- name: InsertTag :exec
INSERT INTO tags (id, user_id, name, created_at)
VALUES (?, ?, ?, ?);

-- name: UpdateTagName :execrows
UPDATE tags
SET name = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteTag :execrows
DELETE FROM tags
WHERE id = ? AND user_id = ?;

-- name: ListTagsByTransactionID :many
SELECT tg.id, tg.user_id, tg.name, tg.created_at
FROM transaction_tags tt
JOIN tags tg ON tg.id = tt.tag_id
WHERE tt.transaction_id = ?
ORDER BY tg.name COLLATE NOCASE ASC;

-- name: ListTagsForTransactions :many
SELECT tt.transaction_id, tg.id AS tag_id, tg.name AS tag_name
FROM transaction_tags tt
JOIN tags tg ON tg.id = tt.tag_id
WHERE tt.transaction_id IN (sqlc.slice('transaction_ids'))
ORDER BY tg.name COLLATE NOCASE ASC;

-- name: DeleteTransactionTags :exec
DELETE FROM transaction_tags WHERE transaction_id = ?;

-- name: InsertTransactionTag :exec
INSERT INTO transaction_tags (transaction_id, tag_id)
VALUES (?, ?);
