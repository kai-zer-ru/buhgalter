-- name: ListUserChangeEventsAfter :many
SELECT id, user_id, entity_type, entity_id, action, occurred_at
FROM user_change_events
WHERE user_id = sqlc.arg(user_id)
  AND id > sqlc.arg(after_id)
  AND (sqlc.arg(since) = '' OR occurred_at >= sqlc.arg(since))
ORDER BY id ASC
LIMIT sqlc.arg(limit);
