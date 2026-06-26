-- name: GetUserTimezone :one
SELECT timezone FROM users WHERE id = ?;

-- name: GetTransactionByID :one
SELECT
    t.id,
    t.user_id,
    t.account_id,
    t.type,
    t.kind,
    t.amount,
    t.description,
    t.category_id,
    t.subcategory_id,
    t.transfer_group_id,
    t.transfer_account_id,
    t.transaction_date,
    t.created_at,
    t.updated_at,
    c.name AS category_name,
    c.icon AS category_icon,
    c.is_system AS category_is_system,
    s.name AS subcategory_name,
    a.name AS account_name,
    ta.name AS transfer_account_name,
    CASE
        WHEN t.transfer_group_id IS NULL THEN 0
        WHEN t.id = (
            SELECT x.id FROM transactions x
            WHERE x.transfer_group_id = t.transfer_group_id
            ORDER BY x.created_at ASC, x.id ASC
            LIMIT 1
        ) THEN 1
        ELSE 0
    END AS transfer_is_out,
    CASE WHEN cp.id IS NOT NULL THEN 1 ELSE 0 END AS credit_payment_linked
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN subcategories s ON s.id = t.subcategory_id
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN accounts ta ON ta.id = t.transfer_account_id
LEFT JOIN credit_payments cp ON cp.transaction_id = t.id
WHERE t.id = ? AND t.user_id = ?;

-- name: ListTransactionsByTransferGroup :many
SELECT
    t.id,
    t.user_id,
    t.account_id,
    t.type,
    t.kind,
    t.amount,
    t.description,
    t.category_id,
    t.subcategory_id,
    t.transfer_group_id,
    t.transfer_account_id,
    t.transaction_date,
    t.created_at,
    t.updated_at,
    c.name AS category_name,
    c.icon AS category_icon,
    c.is_system AS category_is_system,
    s.name AS subcategory_name,
    a.name AS account_name,
    ta.name AS transfer_account_name,
    CASE
        WHEN t.transfer_group_id IS NULL THEN 0
        WHEN t.id = (
            SELECT x.id FROM transactions x
            WHERE x.transfer_group_id = t.transfer_group_id
            ORDER BY x.created_at ASC, x.id ASC
            LIMIT 1
        ) THEN 1
        ELSE 0
    END AS transfer_is_out
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN subcategories s ON s.id = t.subcategory_id
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN accounts ta ON ta.id = t.transfer_account_id
WHERE t.transfer_group_id = ? AND t.user_id = ?
ORDER BY t.created_at ASC, t.id ASC;

-- name: InsertTransaction :exec
INSERT INTO transactions (
    id, user_id, account_id, type, kind, amount, description,
    category_id, subcategory_id, transfer_group_id, transfer_account_id,
    transaction_date, affects_balance, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateTransaction :exec
UPDATE transactions
SET account_id = ?, type = ?, kind = ?, amount = ?, description = ?,
    category_id = ?, subcategory_id = ?, transaction_date = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: UpdateTransferLeg :exec
UPDATE transactions
SET account_id = ?, amount = ?, description = ?, transaction_date = ?,
    kind = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteTransaction :execrows
DELETE FROM transactions WHERE id = ? AND user_id = ?;

-- name: DeleteTransactionsByGroup :execrows
DELETE FROM transactions WHERE transfer_group_id = ? AND user_id = ?;

-- name: ActivateTransaction :execrows
UPDATE transactions SET kind = 'manual', updated_at = ? WHERE id = ? AND user_id = ? AND kind = 'future';

-- name: ListDueFutureTransactions :many
SELECT
    t.id,
    t.user_id,
    t.account_id,
    t.type,
    t.kind,
    t.amount,
    t.description,
    t.category_id,
    t.subcategory_id,
    t.transfer_group_id,
    t.transfer_account_id,
    t.transaction_date,
    t.affects_balance,
    t.created_at,
    t.updated_at
FROM transactions t
WHERE t.user_id = ?
  AND t.kind = 'future'
  AND t.transaction_date <= ?
ORDER BY t.transaction_date ASC, t.created_at ASC;

-- name: ActivateFutureTransactionsBefore :execrows
UPDATE transactions
SET kind = 'manual', updated_at = ?
WHERE user_id = ?
  AND kind = 'future'
  AND transaction_date <= ?;

-- name: ActivateAppliedCreditFutureTransactions :execrows
UPDATE transactions
SET kind = 'manual', updated_at = ?
WHERE user_id = ?
  AND kind = 'future'
  AND id IN (
    SELECT cp.transaction_id
    FROM credit_payments cp
    WHERE cp.transaction_id IS NOT NULL
      AND cp.is_applied = 1
      AND cp.exclude_from_stats = 0
  );

-- name: CountTransactionsFiltered :one
SELECT COUNT(*) AS count
FROM transactions t
WHERE t.user_id = ?
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%');

-- name: ListTransactionsFilteredDateDesc :many
SELECT
    t.id,
    t.user_id,
    t.account_id,
    t.type,
    t.kind,
    t.amount,
    t.description,
    t.category_id,
    t.subcategory_id,
    t.transfer_group_id,
    t.transfer_account_id,
    t.transaction_date,
    t.created_at,
    t.updated_at,
    c.name AS category_name,
    c.icon AS category_icon,
    c.is_system AS category_is_system,
    s.name AS subcategory_name,
    a.name AS account_name,
    ta.name AS transfer_account_name,
    CASE
        WHEN t.transfer_group_id IS NULL THEN 0
        WHEN t.id = (
            SELECT x.id FROM transactions x
            WHERE x.transfer_group_id = t.transfer_group_id
            ORDER BY x.created_at ASC, x.id ASC
            LIMIT 1
        ) THEN 1
        ELSE 0
    END AS transfer_is_out,
    CASE WHEN cp.id IS NOT NULL THEN 1 ELSE 0 END AS credit_payment_linked
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN subcategories s ON s.id = t.subcategory_id
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN accounts ta ON ta.id = t.transfer_account_id
LEFT JOIN credit_payments cp ON cp.transaction_id = t.id
WHERE t.user_id = ?
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
ORDER BY t.transaction_date DESC, t.created_at DESC
LIMIT ? OFFSET ?;

-- name: ListTransactionsFilteredDateAsc :many
SELECT
    t.id,
    t.user_id,
    t.account_id,
    t.type,
    t.kind,
    t.amount,
    t.description,
    t.category_id,
    t.subcategory_id,
    t.transfer_group_id,
    t.transfer_account_id,
    t.transaction_date,
    t.created_at,
    t.updated_at,
    c.name AS category_name,
    c.icon AS category_icon,
    c.is_system AS category_is_system,
    s.name AS subcategory_name,
    a.name AS account_name,
    ta.name AS transfer_account_name,
    CASE
        WHEN t.transfer_group_id IS NULL THEN 0
        WHEN t.id = (
            SELECT x.id FROM transactions x
            WHERE x.transfer_group_id = t.transfer_group_id
            ORDER BY x.created_at ASC, x.id ASC
            LIMIT 1
        ) THEN 1
        ELSE 0
    END AS transfer_is_out,
    CASE WHEN cp.id IS NOT NULL THEN 1 ELSE 0 END AS credit_payment_linked
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN subcategories s ON s.id = t.subcategory_id
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN accounts ta ON ta.id = t.transfer_account_id
LEFT JOIN credit_payments cp ON cp.transaction_id = t.id
WHERE t.user_id = ?
  AND (? = '' OR t.account_id = ?)
  AND (? = '' OR t.type = ?)
  AND (? = '' OR t.category_id = ?)
  AND (? = '' OR t.kind = ?)
  AND (? = '' OR t.transaction_date >= ?)
  AND (? = '' OR t.transaction_date <= ?)
  AND (? = '' OR t.description LIKE '%' || ? || '%')
ORDER BY t.transaction_date ASC, t.created_at ASC
LIMIT ? OFFSET ?;

-- name: ListRecentTransactions :many
SELECT
    t.id,
    t.user_id,
    t.account_id,
    t.type,
    t.kind,
    t.amount,
    t.description,
    t.category_id,
    t.subcategory_id,
    t.transfer_group_id,
    t.transfer_account_id,
    t.transaction_date,
    t.created_at,
    t.updated_at,
    c.name AS category_name,
    c.icon AS category_icon,
    c.is_system AS category_is_system,
    s.name AS subcategory_name,
    a.name AS account_name,
    ta.name AS transfer_account_name,
    CASE
        WHEN t.transfer_group_id IS NULL THEN 0
        WHEN t.id = (
            SELECT x.id FROM transactions x
            WHERE x.transfer_group_id = t.transfer_group_id
            ORDER BY x.created_at ASC, x.id ASC
            LIMIT 1
        ) THEN 1
        ELSE 0
    END AS transfer_is_out,
    CASE WHEN cp.id IS NOT NULL THEN 1 ELSE 0 END AS credit_payment_linked
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN subcategories s ON s.id = t.subcategory_id
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN accounts ta ON ta.id = t.transfer_account_id
LEFT JOIN credit_payments cp ON cp.transaction_id = t.id
WHERE t.user_id = ?
ORDER BY t.transaction_date DESC, t.created_at DESC
LIMIT ?;

-- name: SumIncomeManual :one
SELECT COALESCE(SUM(amount), 0) AS total
FROM transactions
WHERE user_id = ? AND account_id = ? AND type = 'income' AND kind = 'manual'
  AND affects_balance = 1 AND transaction_date <= ?;

-- name: SumExpenseManual :one
SELECT COALESCE(SUM(amount), 0) AS total
FROM transactions
WHERE user_id = ? AND account_id = ? AND type = 'expense' AND kind = 'manual'
  AND affects_balance = 1 AND transaction_date <= ?;

-- name: SumTransferOutManual :one
SELECT COALESCE(SUM(t.amount), 0) AS total
FROM transactions t
WHERE t.user_id = ? AND t.account_id = ? AND t.type = 'transfer' AND t.kind = 'manual'
  AND t.affects_balance = 1 AND t.transaction_date <= ?
  AND t.transfer_group_id IS NOT NULL
  AND t.id = (
    SELECT x.id FROM transactions x
    WHERE x.transfer_group_id = t.transfer_group_id
    ORDER BY x.created_at ASC, x.id ASC
    LIMIT 1
  );

-- name: SumTransferInManual :one
SELECT COALESCE(SUM(t.amount), 0) AS total
FROM transactions t
WHERE t.user_id = ? AND t.account_id = ? AND t.type = 'transfer' AND t.kind = 'manual'
  AND t.affects_balance = 1 AND t.transaction_date <= ?
  AND t.transfer_group_id IS NOT NULL
  AND t.id = (
    SELECT x.id FROM transactions x
    WHERE x.transfer_group_id = t.transfer_group_id
    ORDER BY x.created_at ASC, x.id ASC
    LIMIT 1 OFFSET 1
  );

-- name: SumFutureIncomeInRange :one
SELECT COALESCE(SUM(amount), 0) AS total
FROM transactions
WHERE user_id = ? AND account_id = ? AND type = 'income' AND kind = 'future'
  AND affects_balance = 1 AND transaction_date >= ? AND transaction_date <= ?;

-- name: SumFutureExpenseInRange :one
SELECT COALESCE(SUM(amount), 0) AS total
FROM transactions
WHERE user_id = ? AND account_id = ? AND type = 'expense' AND kind = 'future'
  AND affects_balance = 1 AND transaction_date >= ? AND transaction_date <= ?;

-- name: SumFutureTransferOutInRange :one
SELECT COALESCE(SUM(t.amount), 0) AS total
FROM transactions t
WHERE t.user_id = ? AND t.account_id = ? AND t.type = 'transfer' AND kind = 'future'
  AND t.affects_balance = 1 AND t.transaction_date >= ? AND t.transaction_date <= ?
  AND t.transfer_group_id IS NOT NULL
  AND t.id = (
    SELECT x.id FROM transactions x
    WHERE x.transfer_group_id = t.transfer_group_id
    ORDER BY x.created_at ASC, x.id ASC
    LIMIT 1
  );

-- name: SumFutureTransferInInRange :one
SELECT COALESCE(SUM(t.amount), 0) AS total
FROM transactions t
WHERE t.user_id = ? AND t.account_id = ? AND t.type = 'transfer' AND kind = 'future'
  AND t.affects_balance = 1 AND t.transaction_date >= ? AND t.transaction_date <= ?
  AND t.transfer_group_id IS NOT NULL
  AND t.id = (
    SELECT x.id FROM transactions x
    WHERE x.transfer_group_id = t.transfer_group_id
    ORDER BY x.created_at ASC, x.id ASC
    LIMIT 1 OFFSET 1
  );

-- name: HasFutureInMonth :one
SELECT COUNT(*) AS count
FROM transactions
WHERE user_id = ? AND account_id = ? AND kind = 'future'
  AND transaction_date >= ? AND transaction_date <= ?;

-- name: UpdateTransferCreatedAt :exec
UPDATE transactions SET created_at = ? WHERE id = ? AND user_id = ?;

-- name: UpdateTransferAccountID :exec
UPDATE transactions SET transfer_account_id = ?, updated_at = ? WHERE id = ? AND user_id = ?;

-- name: GetCategoryByNameAndType :one
SELECT id, name, type, icon, sort_order, is_primary, is_system, created_at
FROM categories
WHERE user_id = ? AND name = ? AND type = ?;
