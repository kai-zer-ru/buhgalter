-- name: ListDebtorsByUser :many
SELECT id, user_id, name, created_at
FROM debtors
WHERE user_id = ?
ORDER BY name COLLATE NOCASE ASC;

-- name: GetDebtorByID :one
SELECT id, user_id, name, created_at
FROM debtors
WHERE id = ? AND user_id = ?;

-- name: GetDebtorByName :one
SELECT id, user_id, name, created_at
FROM debtors
WHERE user_id = sqlc.arg(user_id) AND lower(name) = lower(sqlc.arg(name));

-- name: InsertDebtor :exec
INSERT INTO debtors (id, user_id, name, created_at)
VALUES (?, ?, ?, ?);

-- name: UpdateDebtorName :execrows
UPDATE debtors
SET name = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteDebtor :execrows
DELETE FROM debtors
WHERE id = ? AND user_id = ?;

-- name: CountActiveDebtsByDebtor :one
SELECT COUNT(*) FROM debts
WHERE debtor_id = ? AND user_id = ? AND is_settled = 0;

-- name: CountActiveDebtsByDebtorAndDirection :one
SELECT COUNT(*) FROM debts
WHERE debtor_id = ? AND user_id = ? AND is_settled = 0 AND direction = ?;

-- name: ListActiveDebtsByUser :many
SELECT
    d.id,
    d.user_id,
    d.debtor_id,
    dt.name AS debtor_name,
    d.direction,
    d.amount,
    d.affects_balance,
    d.debt_date,
    d.due_date,
    d.description,
    d.transaction_id,
    d.is_settled,
    d.settled_at,
    d.created_at,
    tx.account_id AS open_account_id,
    acc.name AS open_account_name
FROM debts d
JOIN debtors dt ON dt.id = d.debtor_id
LEFT JOIN debt_transactions dtx_open ON dtx_open.debt_id = d.id AND dtx_open.role = 'open'
LEFT JOIN transactions tx ON tx.id = COALESCE(dtx_open.transaction_id, d.transaction_id) AND tx.user_id = d.user_id
LEFT JOIN accounts acc ON acc.id = tx.account_id
WHERE d.user_id = ? AND d.is_settled = 0
ORDER BY d.debt_date DESC, d.created_at DESC;

-- name: ListSettledDebtsByUser :many
SELECT
    d.id,
    d.user_id,
    d.debtor_id,
    dt.name AS debtor_name,
    d.direction,
    d.amount,
    d.affects_balance,
    d.debt_date,
    d.due_date,
    d.description,
    d.transaction_id,
    d.is_settled,
    d.settled_at,
    d.created_at,
    tx.account_id AS open_account_id,
    acc.name AS open_account_name
FROM debts d
JOIN debtors dt ON dt.id = d.debtor_id
LEFT JOIN debt_transactions dtx_open ON dtx_open.debt_id = d.id AND dtx_open.role = 'open'
LEFT JOIN transactions tx ON tx.id = COALESCE(dtx_open.transaction_id, d.transaction_id) AND tx.user_id = d.user_id
LEFT JOIN accounts acc ON acc.id = tx.account_id
WHERE d.user_id = ? AND d.is_settled = 1
ORDER BY d.settled_at DESC, d.created_at DESC;

-- name: ListAllDebtsByUser :many
SELECT
    d.id,
    d.user_id,
    d.debtor_id,
    dt.name AS debtor_name,
    d.direction,
    d.amount,
    d.affects_balance,
    d.debt_date,
    d.due_date,
    d.description,
    d.transaction_id,
    d.is_settled,
    d.settled_at,
    d.created_at,
    tx.account_id AS open_account_id,
    acc.name AS open_account_name
FROM debts d
JOIN debtors dt ON dt.id = d.debtor_id
LEFT JOIN debt_transactions dtx_open ON dtx_open.debt_id = d.id AND dtx_open.role = 'open'
LEFT JOIN transactions tx ON tx.id = COALESCE(dtx_open.transaction_id, d.transaction_id) AND tx.user_id = d.user_id
LEFT JOIN accounts acc ON acc.id = tx.account_id
WHERE d.user_id = ?
ORDER BY d.is_settled ASC, d.debt_date DESC, d.created_at DESC;

-- name: GetDebtByID :one
SELECT
    d.id,
    d.user_id,
    d.debtor_id,
    dt.name AS debtor_name,
    d.direction,
    d.amount,
    d.affects_balance,
    d.debt_date,
    d.due_date,
    d.description,
    d.transaction_id,
    d.is_settled,
    d.settled_at,
    d.created_at,
    tx.account_id AS open_account_id,
    acc.name AS open_account_name
FROM debts d
JOIN debtors dt ON dt.id = d.debtor_id
LEFT JOIN debt_transactions dtx_open ON dtx_open.debt_id = d.id AND dtx_open.role = 'open'
LEFT JOIN transactions tx ON tx.id = COALESCE(dtx_open.transaction_id, d.transaction_id) AND tx.user_id = d.user_id
LEFT JOIN accounts acc ON acc.id = tx.account_id
WHERE d.id = ? AND d.user_id = ?;

-- name: InsertDebt :exec
INSERT INTO debts (
    id, user_id, debtor_id, direction, amount, affects_balance,
    debt_date, due_date, description, transaction_id, is_settled, settled_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SettleDebt :execrows
UPDATE debts
SET is_settled = 1, settled_at = ?
WHERE id = ? AND user_id = ? AND is_settled = 0;

-- name: ReduceDebtAmount :execrows
UPDATE debts
SET amount = ?
WHERE id = ? AND user_id = ? AND is_settled = 0;

-- name: UpdateDebtTransactionID :exec
UPDATE debts SET transaction_id = ? WHERE id = ? AND user_id = ?;

-- name: ClearDebtTransactionLink :exec
UPDATE debts
SET transaction_id = NULL, affects_balance = 0
WHERE transaction_id = ? AND user_id = ?;

-- name: ClearDebtTransactionLinkByDebtID :exec
UPDATE debts
SET transaction_id = NULL, affects_balance = 0
WHERE id = ? AND user_id = ?;

-- name: InsertDebtTransactionLink :exec
INSERT INTO debt_transactions (debt_id, transaction_id, role)
VALUES (?, ?, ?);

-- name: DeleteDebtTransactionLink :exec
DELETE FROM debt_transactions
WHERE transaction_id = ?;

-- name: ListTransactionIDsByDebt :many
SELECT transaction_id
FROM debt_transactions
WHERE debt_id = ?;

-- name: ListTransactionIDsByDebtor :many
SELECT DISTINCT dtx.transaction_id
FROM debt_transactions dtx
INNER JOIN debts d ON d.id = dtx.debt_id
INNER JOIN transactions t ON t.id = dtx.transaction_id
WHERE d.user_id = ? AND d.debtor_id = ?
ORDER BY t.transaction_date DESC, t.created_at DESC;

-- name: CountSettleLinksByDebt :one
SELECT COUNT(*)
FROM debt_transactions
WHERE debt_id = ? AND role = 'settle';

-- name: GetDebtLinkByTransactionID :one
SELECT debt_id, role
FROM debt_transactions
WHERE transaction_id = ?;

-- name: DeleteDebt :execrows
DELETE FROM debts WHERE id = ? AND user_id = ?;

-- name: ListDebtsByDebtor :many
SELECT
    d.id,
    d.user_id,
    d.debtor_id,
    dt.name AS debtor_name,
    d.direction,
    d.amount,
    d.affects_balance,
    d.debt_date,
    d.due_date,
    d.description,
    d.transaction_id,
    d.is_settled,
    d.settled_at,
    d.created_at,
    tx.account_id AS open_account_id,
    acc.name AS open_account_name
FROM debts d
JOIN debtors dt ON dt.id = d.debtor_id
LEFT JOIN debt_transactions dtx_open ON dtx_open.debt_id = d.id AND dtx_open.role = 'open'
LEFT JOIN transactions tx ON tx.id = COALESCE(dtx_open.transaction_id, d.transaction_id) AND tx.user_id = d.user_id
LEFT JOIN accounts acc ON acc.id = tx.account_id
WHERE d.user_id = ? AND d.debtor_id = ?
ORDER BY d.is_settled ASC, d.debt_date DESC, d.created_at DESC;

-- name: ListActiveDebtsForSummary :many
SELECT direction, amount, due_date
FROM debts
WHERE user_id = ? AND is_settled = 0;
