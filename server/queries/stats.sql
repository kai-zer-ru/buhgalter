-- name: StatsSummary :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS income_total,
    CAST(COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS expense_total,
    COUNT(*) AS transaction_count
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.currency = ?
WHERE t.user_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND (? = '' OR t.store_id = ?)
  AND (? = '' OR t.merchant_id = ?)
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND cp.exclude_from_stats = 1
  );

-- name: StatsSummaryAccount :one
-- Account-scoped totals: incoming transfers count as income, outgoing as expense (matches balance).
SELECT
    CAST(COALESCE(SUM(
        CASE
            WHEN t.type = 'income' THEN t.amount
            WHEN t.type = 'transfer' AND t.transfer_group_id IS NOT NULL AND t.id = (
                SELECT x.id FROM transactions x
                WHERE x.transfer_group_id = t.transfer_group_id
                ORDER BY x.created_at ASC, x.id ASC
                LIMIT 1 OFFSET 1
            ) THEN t.amount
            ELSE 0
        END
    ), 0) AS INTEGER) AS income_total,
    CAST(COALESCE(SUM(
        CASE
            WHEN t.type = 'expense' THEN t.amount
            WHEN t.type = 'transfer' AND t.transfer_group_id IS NOT NULL AND t.id = (
                SELECT x.id FROM transactions x
                WHERE x.transfer_group_id = t.transfer_group_id
                ORDER BY x.created_at ASC, x.id ASC
                LIMIT 1
            ) THEN t.amount
            ELSE 0
        END
    ), 0) AS INTEGER) AS expense_total,
    COUNT(*) AS transaction_count
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.currency = ?
WHERE t.user_id = ?
  AND t.account_id = ?
  AND t.type IN ('income', 'expense', 'transfer')
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND (? = '' OR t.store_id = ?)
  AND (? = '' OR t.merchant_id = ?)
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND cp.exclude_from_stats = 1
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
JOIN accounts a ON a.id = t.account_id AND a.currency = ?
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
  AND (? = '' OR t.store_id = ?)
  AND (? = '' OR t.merchant_id = ?)
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND cp.exclude_from_stats = 1
  )
GROUP BY c.id, c.name, c.icon, c.type
ORDER BY total DESC;

-- name: StatsByStore :many
SELECT
    m.id AS store_id,
    m.name AS store_name,
    CAST(COALESCE(SUM(t.amount), 0) AS INTEGER) AS total,
    COUNT(*) AS tx_count
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.currency = ?
JOIN merchants m ON m.id = t.merchant_id AND m.user_id = t.user_id
WHERE t.user_id = ?
  AND t.type = 'expense'
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND (? = '' OR t.store_id = ?)
  AND (? = '' OR t.merchant_id = ?)
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND cp.exclude_from_stats = 1
  )
GROUP BY m.id, m.name
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
JOIN accounts a ON a.id = t.account_id AND a.currency = ?
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
  AND (? = '' OR t.store_id = ?)
  AND (? = '' OR t.merchant_id = ?)
  AND NOT (
    t.receipt_import_id IS NOT NULL
    AND t.receipt_item_index IS NULL
  )
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND cp.exclude_from_stats = 1
  )
GROUP BY c.id, c.name, c.icon, s.id, s.name
ORDER BY total DESC;

-- name: StatsBySubcategoryReceiptLines :many
-- Attach: one transaction per receipt; subcategories per line in receipt_import_item_categories.
SELECT
    c.id AS category_id,
    c.name AS category_name,
    c.icon AS category_icon,
    s.id AS subcategory_id,
    s.name AS subcategory_name,
    CAST(COALESCE(SUM(
        CASE
            WHEN COALESCE(CAST(json_extract(ri.payload_json, '$.items[' || ric.item_index || '].sum') AS INTEGER), 0) > 0
            THEN CAST(json_extract(ri.payload_json, '$.items[' || ric.item_index || '].sum') AS INTEGER)
            ELSE COALESCE(CAST(json_extract(ri.payload_json, '$.items[' || ric.item_index || '].price') AS INTEGER), 0)
        END
    ), 0) AS INTEGER) AS total,
    COUNT(*) AS tx_count
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.currency = ?
JOIN receipt_imports ri ON ri.id = t.receipt_import_id AND ri.user_id = t.user_id
JOIN receipt_import_item_categories ric ON ric.receipt_import_id = t.receipt_import_id AND ric.user_id = t.user_id
JOIN categories c ON c.id = ric.category_id
JOIN subcategories s ON s.id = ric.subcategory_id
WHERE t.user_id = ?
  AND t.receipt_import_id IS NOT NULL
  AND t.receipt_item_index IS NULL
  AND ric.subcategory_id IS NOT NULL
  AND trim(ric.subcategory_id) != ''
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR ric.category_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND (? = '' OR t.store_id = ?)
  AND (? = '' OR t.merchant_id = ?)
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND cp.exclude_from_stats = 1
  )
GROUP BY c.id, c.name, c.icon, s.id, s.name
ORDER BY total DESC;

-- name: StatsPeriodRows :many
SELECT
    t.transaction_date,
    t.type,
    t.amount
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.currency = ?
WHERE t.user_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
  AND (? = '' OR t.store_id = ?)
  AND (? = '' OR t.merchant_id = ?)
  AND NOT EXISTS (
    SELECT 1
    FROM credit_payments cp
    WHERE cp.transaction_id = t.id
      AND cp.exclude_from_stats = 1
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
JOIN debts d ON d.id = dtx.debt_id AND d.currency = ?
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
JOIN accounts a ON a.id = c.debit_account_id AND a.currency = ?
WHERE t.user_id = ?
  AND c.user_id = ?
  AND c.id = ?
  AND t.type IN ('income', 'expense')
  AND cp.exclude_from_stats = 0
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?);

-- name: StatsContextCreditPaid :one
SELECT
    CAST(COALESCE(SUM(cp.amount), 0) AS INTEGER) AS paid_total,
    COUNT(*) AS payment_count
FROM credit_payments cp
JOIN credits c ON c.id = cp.credit_id
JOIN accounts a ON a.id = c.debit_account_id AND a.currency = ?
WHERE c.user_id = ?
  AND c.id = ?
  AND cp.transaction_id IS NOT NULL
  AND cp.exclude_from_stats = 0
  AND (? = '' OR cp.payment_date >= ?)
  AND (? = '' OR cp.payment_date <= ?);

-- name: StatsContextCreditRemaining :one
SELECT
    principal_amount,
    paid_amount
FROM credits c
JOIN accounts a ON a.id = c.debit_account_id AND a.currency = ?
WHERE c.id = ? AND c.user_id = ?;

-- name: StatsContextDebts :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS income_total,
    CAST(COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) AS INTEGER) AS expense_total,
    COUNT(DISTINCT t.id) AS transaction_count
FROM transactions t
JOIN debt_transactions dtx ON dtx.transaction_id = t.id
JOIN debts d ON d.id = dtx.debt_id AND d.currency = ?
WHERE t.user_id = ?
  AND d.user_id = ?
  AND t.type IN ('income', 'expense')
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?);
