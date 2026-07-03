-- name: ListRecurringOperationsByUser :many
SELECT
    ro.id,
    ro.user_id,
    ro.type,
    ro.amount,
    ro.description,
    ro.account_id,
    a.name AS account_name,
    ro.category_id,
    c.name AS category_name,
    ro.subcategory_id,
    s.name AS subcategory_name,
    ro.period,
    ro.weekday,
    ro.day_of_month,
    ro.start_date,
    ro.time_local,
    ro.next_run_at,
    ro.last_run_at,
    ro.active,
    ro.created_at,
    ro.updated_at
FROM recurring_operations ro
JOIN accounts a ON a.id = ro.account_id
JOIN categories c ON c.id = ro.category_id
LEFT JOIN subcategories s ON s.id = ro.subcategory_id
WHERE ro.user_id = ?
ORDER BY ro.created_at DESC;

-- name: GetRecurringOperationByID :one
SELECT
    ro.id,
    ro.user_id,
    ro.type,
    ro.amount,
    ro.description,
    ro.account_id,
    a.name AS account_name,
    ro.category_id,
    c.name AS category_name,
    ro.subcategory_id,
    s.name AS subcategory_name,
    ro.period,
    ro.weekday,
    ro.day_of_month,
    ro.start_date,
    ro.time_local,
    ro.next_run_at,
    ro.last_run_at,
    ro.active,
    ro.created_at,
    ro.updated_at
FROM recurring_operations ro
JOIN accounts a ON a.id = ro.account_id
JOIN categories c ON c.id = ro.category_id
LEFT JOIN subcategories s ON s.id = ro.subcategory_id
WHERE ro.id = ? AND ro.user_id = ?;

-- name: InsertRecurringOperation :exec
INSERT INTO recurring_operations (
    id, user_id, type, amount, description,
    account_id, category_id, subcategory_id,
    period, weekday, day_of_month,
    start_date, time_local, next_run_at, last_run_at, active,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRecurringOperation :execrows
UPDATE recurring_operations
SET type = ?, amount = ?, description = ?,
    account_id = ?, category_id = ?, subcategory_id = ?,
    period = ?, weekday = ?, day_of_month = ?,
    start_date = ?, time_local = ?, next_run_at = ?, active = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteRecurringOperation :execrows
DELETE FROM recurring_operations
WHERE id = ? AND user_id = ?;

-- name: ReassignRecurringOperationsCategory :exec
UPDATE recurring_operations
SET category_id = ?, updated_at = ?
WHERE user_id = ? AND category_id = ?;

-- name: ReassignRecurringSubcategory :exec
UPDATE recurring_operations
SET subcategory_id = ?, updated_at = ?
WHERE user_id = ? AND subcategory_id = ?;

-- name: ListDueRecurringOperations :many
SELECT
    id,
    user_id,
    type,
    amount,
    description,
    account_id,
    category_id,
    subcategory_id,
    period,
    weekday,
    day_of_month,
    start_date,
    time_local,
    next_run_at,
    last_run_at,
    active,
    created_at,
    updated_at
FROM recurring_operations
WHERE user_id = ? AND active = 1 AND next_run_at <= ?
ORDER BY next_run_at ASC;

-- name: MarkRecurringOperationRan :execrows
UPDATE recurring_operations
SET next_run_at = ?, last_run_at = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: SetRecurringOperationNextRunAt :execrows
UPDATE recurring_operations SET next_run_at = ?
WHERE id = ? AND user_id = ?;
