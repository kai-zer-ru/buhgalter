-- name: ListSubscriptionsByUser :many
SELECT
    s.id,
    s.user_id,
    s.name,
    s.description,
    s.icon,
    s.website_url,
    s.amount,
    s.account_id,
    a.name AS account_name,
    s.subcategory_id,
    sub.name AS subcategory_name,
    sub.icon AS subcategory_icon,
    s.period,
    s.weekday,
    s.day_of_month,
    s.start_date,
    s.time_local,
    s.next_run_at,
    s.last_run_at,
    s.active,
    s.created_at,
    s.updated_at
FROM subscriptions s
JOIN accounts a ON a.id = s.account_id
LEFT JOIN subcategories sub ON sub.id = s.subcategory_id
WHERE s.user_id = ?
ORDER BY s.next_run_at ASC, s.created_at DESC;

-- name: GetSubscriptionByID :one
SELECT
    s.id,
    s.user_id,
    s.name,
    s.description,
    s.icon,
    s.website_url,
    s.amount,
    s.account_id,
    a.name AS account_name,
    s.subcategory_id,
    sub.name AS subcategory_name,
    sub.icon AS subcategory_icon,
    s.period,
    s.weekday,
    s.day_of_month,
    s.start_date,
    s.time_local,
    s.next_run_at,
    s.last_run_at,
    s.active,
    s.created_at,
    s.updated_at
FROM subscriptions s
JOIN accounts a ON a.id = s.account_id
LEFT JOIN subcategories sub ON sub.id = s.subcategory_id
WHERE s.id = ? AND s.user_id = ?;

-- name: InsertSubscription :exec
INSERT INTO subscriptions (
    id, user_id, name, description, icon, website_url,
    amount, account_id, subcategory_id,
    period, weekday, day_of_month,
    start_date, time_local, next_run_at, last_run_at, active,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateSubscription :execrows
UPDATE subscriptions
SET name = ?, description = ?, icon = ?, website_url = ?,
    amount = ?, account_id = ?, subcategory_id = ?,
    period = ?, weekday = ?, day_of_month = ?,
    start_date = ?, time_local = ?, next_run_at = ?, active = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteSubscription :execrows
DELETE FROM subscriptions
WHERE id = ? AND user_id = ?;

-- name: ListDueSubscriptions :many
SELECT
    id,
    user_id,
    name,
    description,
    icon,
    website_url,
    amount,
    account_id,
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
FROM subscriptions
WHERE user_id = ? AND active = 1 AND next_run_at <= ?
ORDER BY next_run_at ASC;

-- name: MarkSubscriptionRan :execrows
UPDATE subscriptions
SET next_run_at = ?, last_run_at = ?, subcategory_id = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: SetSubscriptionSubcategory :execrows
UPDATE subscriptions
SET subcategory_id = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: CountSubscriptionsBySubcategory :one
SELECT COUNT(*)
FROM subscriptions
WHERE user_id = ? AND subcategory_id = ?;

-- name: ListActiveSubscriptionsForNotify :many
SELECT
    s.id,
    s.name,
    s.description,
    s.website_url,
    s.amount,
    s.account_id,
    a.name AS account_name,
    s.next_run_at
FROM subscriptions s
JOIN accounts a ON a.id = s.account_id
WHERE s.user_id = ? AND s.active = 1;

-- name: ListCandidateExpenseTransactions :many
SELECT
    t.id,
    t.user_id,
    t.account_id,
    a.name AS account_name,
    t.type,
    t.kind,
    t.amount,
    t.description,
    t.category_id,
    c.name AS category_name,
    c.is_system AS category_is_system,
    t.subcategory_id,
    s.name AS subcategory_name,
    t.transaction_date,
    t.created_at,
    t.updated_at
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN subcategories s ON s.id = t.subcategory_id
WHERE t.user_id = ?
  AND t.type = 'expense'
  AND t.subscription_id IS NULL
  AND (sqlc.narg(account_id) IS NULL OR t.account_id = sqlc.narg(account_id))
  AND (sqlc.narg(amount_min) IS NULL OR t.amount >= sqlc.narg(amount_min))
  AND (sqlc.narg(amount_max) IS NULL OR t.amount <= sqlc.narg(amount_max))
  AND (sqlc.narg(date_from) IS NULL OR t.transaction_date >= sqlc.narg(date_from))
  AND (sqlc.narg(date_to) IS NULL OR t.transaction_date <= sqlc.narg(date_to))
ORDER BY t.transaction_date DESC, t.created_at DESC;

-- name: AttachTransactionToSubscription :execrows
UPDATE transactions
SET category_id = ?, subcategory_id = ?, subscription_id = ?, description = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND type = 'expense' AND subscription_id IS NULL;

-- name: GetSubcategoryByCategoryAndName :one
SELECT s.id, s.category_id, s.name, s.icon, s.sort_order, s.created_at
FROM subcategories s
JOIN categories c ON c.id = s.category_id
WHERE s.category_id = ? AND s.name = ? AND c.user_id = ?;
