-- name: ListBudgetsByUser :many
SELECT
    b.id,
    b.user_id,
    b.name,
    b.scope,
    b.category_id,
    c.name AS category_name,
    c.icon AS category_icon,
    b.subcategory_id,
    s.name AS subcategory_name,
    b.amount,
    b.period,
    b.account_id,
    a.name AS account_name,
    b.month,
    b.copy_forward,
    b.rollover,
    b.alert_at_percent,
    b.is_active,
    b.created_at,
    b.updated_at
FROM budgets b
LEFT JOIN categories c ON c.id = b.category_id
LEFT JOIN subcategories s ON s.id = b.subcategory_id
LEFT JOIN accounts a ON a.id = b.account_id
WHERE b.user_id = ?
  AND (? = '' OR b.month = ?)
ORDER BY b.created_at DESC;

-- name: CountActiveBudgetsByUserMonth :one
SELECT COUNT(*) AS cnt
FROM budgets
WHERE user_id = ?
  AND month = ?
  AND is_active = 1;

-- name: GetBudgetByID :one
SELECT
    b.id,
    b.user_id,
    b.name,
    b.scope,
    b.category_id,
    c.name AS category_name,
    c.icon AS category_icon,
    b.subcategory_id,
    s.name AS subcategory_name,
    b.amount,
    b.period,
    b.account_id,
    a.name AS account_name,
    b.month,
    b.copy_forward,
    b.rollover,
    b.alert_at_percent,
    b.is_active,
    b.created_at,
    b.updated_at
FROM budgets b
LEFT JOIN categories c ON c.id = b.category_id
LEFT JOIN subcategories s ON s.id = b.subcategory_id
LEFT JOIN accounts a ON a.id = b.account_id
WHERE b.id = ? AND b.user_id = ?;

-- name: InsertBudget :exec
INSERT INTO budgets (
    id, user_id, name, scope,
    category_id, subcategory_id, amount, period, account_id,
    month, copy_forward,
    rollover, alert_at_percent, is_active,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateBudget :execrows
UPDATE budgets
SET name = ?, scope = ?,
    category_id = ?, subcategory_id = ?, amount = ?,
    account_id = ?, copy_forward = ?, alert_at_percent = ?, is_active = ?,
    updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteBudget :execrows
DELETE FROM budgets
WHERE id = ? AND user_id = ?;

-- name: CountActiveBudgetConflict :one
SELECT COUNT(*) AS cnt
FROM budgets
WHERE user_id = ?
  AND scope = ?
  AND IFNULL(category_id, '') = IFNULL(?, '')
  AND IFNULL(subcategory_id, '') = IFNULL(?, '')
  AND month = ?
  AND is_active = 1
  AND (sqlc.arg(exclude_id) = '' OR id != sqlc.arg(exclude_id));

-- name: GetBudgetPeriod :one
SELECT
    id,
    budget_id,
    period_start,
    planned_amount,
    rollover_amount,
    created_at,
    updated_at
FROM budget_periods
WHERE budget_id = ? AND period_start = ?;

-- name: InsertBudgetPeriod :exec
INSERT INTO budget_periods (
    id, budget_id, period_start, planned_amount, rollover_amount,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateBudgetPeriodPlannedAmount :execrows
UPDATE budget_periods
SET planned_amount = ?, updated_at = ?
WHERE budget_id = ? AND period_start = ?;

-- name: BudgetSpent :one
SELECT CAST(COALESCE(SUM(t.amount), 0) AS INTEGER) AS spent
FROM transactions t
WHERE t.user_id = ?
  AND t.type = 'expense'
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.subcategory_id = ?)
  AND t.kind = 'manual'
  AND t.transaction_date >= ?
  AND t.transaction_date < ?
  AND t.transfer_group_id IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND cp.exclude_from_stats = 1
  );

-- name: HasBudgetAlertSent :one
SELECT COUNT(*) AS cnt
FROM budget_alert_sent
WHERE budget_id = ? AND period_start = ? AND threshold_percent = ?;

-- name: InsertBudgetAlertSent :exec
INSERT INTO budget_alert_sent (budget_id, period_start, threshold_percent, sent_at)
VALUES (?, ?, ?, ?);

-- name: ListActiveBudgetsByUser :many
SELECT
    b.id,
    b.user_id,
    b.name,
    b.scope,
    b.category_id,
    b.subcategory_id,
    b.amount,
    b.account_id,
    b.month,
    b.alert_at_percent
FROM budgets b
WHERE b.user_id = ? AND b.is_active = 1
  AND (? = '' OR b.month = ?);
