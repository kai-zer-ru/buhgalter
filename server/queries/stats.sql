-- name: StatsSummary :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS income_total,
    CAST(COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS expense_total,
    COUNT(*) AS transaction_count
FROM transactions t
WHERE t.user_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND (cp.exclude_from_stats = 1 OR cp.kind = 'retroactive')
  );

-- name: StatsByCategory :many
SELECT
    c.id AS category_id,
    c.name AS category_name,
    c.icon AS category_icon,
    c.type AS category_type,
    CAST(COALESCE(SUM(t.amount), 0) AS INTEGER) AS total,
    COUNT(*) AS tx_count
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND (cp.exclude_from_stats = 1 OR cp.kind = 'retroactive')
  )
GROUP BY c.id, c.name, c.icon, c.type
ORDER BY total DESC;

-- name: StatsBySubcategory :many
SELECT
    c.id AS category_id,
    c.name AS category_name,
    c.icon AS category_icon,
    s.id AS subcategory_id,
    s.name AS subcategory_name,
    CAST(COALESCE(SUM(t.amount), 0) AS INTEGER) AS total,
    COUNT(*) AS tx_count
FROM transactions t
JOIN categories c ON c.id = t.category_id
JOIN subcategories s ON s.id = t.subcategory_id
WHERE t.user_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND (cp.exclude_from_stats = 1 OR cp.kind = 'retroactive')
  )
GROUP BY c.id, c.name, c.icon, s.id, s.name
ORDER BY total DESC;

-- name: StatsPeriodRows :many
SELECT
    t.transaction_date,
    t.type,
    t.amount
FROM transactions t
WHERE t.user_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND (cp.exclude_from_stats = 1 OR cp.kind = 'retroactive')
  )
ORDER BY t.transaction_date ASC;

-- name: StatsContextDebtor :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS income_total,
    CAST(COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS expense_total,
    CAST(COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS lent_total,
    CAST(COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS borrowed_total,
    COUNT(DISTINCT t.id) AS transaction_count
FROM transactions t
JOIN debt_transactions dtx ON dtx.transaction_id = t.id
JOIN debts d ON d.id = dtx.debt_id
WHERE t.user_id = ?
  AND d.user_id = ?
  AND d.debtor_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?);

-- name: StatsContextCredit :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS income_total,
    CAST(COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS expense_total,
    COUNT(DISTINCT t.id) AS transaction_count
FROM transactions t
JOIN credit_payments cp ON cp.transaction_id = t.id
JOIN credits c ON c.id = cp.credit_id
WHERE t.user_id = ?
  AND c.user_id = ?
  AND c.id = ?
  AND t.type IN ('income', 'expense')
  AND cp.exclude_from_stats = 0
  AND cp.kind != 'retroactive'
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?);

-- name: StatsContextCreditPaid :one
SELECT
    CAST(COALESCE(SUM(cp.amount), 0) AS INTEGER) AS paid_total,
    COUNT(*) AS payment_count
FROM credit_payments cp
JOIN credits c ON c.id = cp.credit_id
WHERE c.user_id = ?
  AND c.id = ?
  AND cp.transaction_id IS NOT NULL
  AND cp.exclude_from_stats = 0
  AND cp.kind != 'retroactive'
  AND (? = '' OR cp.payment_date >= ?)
  AND (? = '' OR cp.payment_date <= ?);

-- name: StatsContextCreditRemaining :one
SELECT
    principal_amount,
    paid_amount
FROM credits
WHERE id = ? AND user_id = ?;

-- name: StatsContextDebts :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS income_total,
    CAST(COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS expense_total,
    COUNT(DISTINCT t.id) AS transaction_count
FROM transactions t
JOIN debt_transactions dtx ON dtx.transaction_id = t.id
JOIN debts d ON d.id = dtx.debt_id
WHERE t.user_id = ?
  AND d.user_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?);
