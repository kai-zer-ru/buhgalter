-- name: ListCreditsByUser :many
SELECT
    c.id,
    c.user_id,
    c.name,
    c.principal_amount,
    c.issue_date,
    c.term_months,
    c.interest_rate,
    c.payment_interval,
    c.paid_amount,
    c.monthly_payment,
    c.debit_account_id,
    a.name AS debit_account_name,
    c.added_retroactively,
    c.recorded_at,
    c.status,
    c.closed_at,
    c.created_at,
    c.updated_at
FROM credits c
JOIN accounts a ON a.id = c.debit_account_id
WHERE c.user_id = ?
  AND (? = '' OR c.status = ?)
ORDER BY c.status ASC, c.issue_date DESC, c.created_at DESC;

-- name: GetCreditByID :one
SELECT
    c.id,
    c.user_id,
    c.name,
    c.principal_amount,
    c.issue_date,
    c.term_months,
    c.interest_rate,
    c.payment_interval,
    c.paid_amount,
    c.monthly_payment,
    c.debit_account_id,
    a.name AS debit_account_name,
    c.added_retroactively,
    c.recorded_at,
    c.status,
    c.closed_at,
    c.created_at,
    c.updated_at
FROM credits c
JOIN accounts a ON a.id = c.debit_account_id
WHERE c.id = ? AND c.user_id = ?;

-- name: InsertCredit :exec
INSERT INTO credits (
    id, user_id, name, principal_amount, issue_date, term_months, interest_rate,
    payment_interval, paid_amount, monthly_payment, debit_account_id,
    added_retroactively, recorded_at, status, closed_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateCredit :execrows
UPDATE credits
SET name = ?, monthly_payment = ?, debit_account_id = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND status = 'active';

-- name: UpdateCreditPaidAmount :exec
UPDATE credits
SET paid_amount = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: CloseCredit :execrows
UPDATE credits
SET status = 'closed', closed_at = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND status = 'active';

-- name: DeleteCredit :execrows
DELETE FROM credits WHERE id = ? AND user_id = ?;

-- name: ListCreditPayments :many
SELECT
    cp.id,
    cp.credit_id,
    cp.transaction_id,
    cp.amount,
    cp.payment_date,
    cp.kind,
    cp.is_applied,
    cp.exclude_from_stats,
    cp.created_at,
    t.kind AS transaction_kind
FROM credit_payments cp
LEFT JOIN transactions t ON t.id = cp.transaction_id
WHERE cp.credit_id = ?
ORDER BY cp.payment_date ASC, cp.created_at ASC;

-- name: GetCreditPaymentByID :one
SELECT
    id, credit_id, transaction_id, amount, payment_date, kind,
    is_applied, exclude_from_stats, created_at
FROM credit_payments
WHERE id = ? AND credit_id = ?;

-- name: InsertCreditPayment :exec
INSERT INTO credit_payments (
    id, credit_id, transaction_id, amount, payment_date, kind,
    is_applied, exclude_from_stats, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ApplyCreditPayment :execrows
UPDATE credit_payments
SET is_applied = 1, kind = ?, transaction_id = ?
WHERE id = ? AND credit_id = ? AND is_applied = 0;

-- name: ApplyScheduledPayment :execrows
UPDATE credit_payments
SET transaction_id = ?, is_applied = 1, payment_date = ?, amount = ?
WHERE id = ? AND credit_id = ? AND kind = 'scheduled' AND is_applied = 0;

-- name: MarkCreditPaymentApplied :execrows
UPDATE credit_payments
SET is_applied = 1
WHERE id = ? AND credit_id = ? AND is_applied = 0 AND transaction_id IS NOT NULL;

-- name: DeleteUnappliedScheduledOnDate :execrows
DELETE FROM credit_payments
WHERE credit_id = ?
  AND kind = 'scheduled'
  AND is_applied = 0
  AND payment_date = ?;

-- name: DeleteUnappliedCreditPayments :execrows
DELETE FROM credit_payments
WHERE credit_id = ? AND is_applied = 0;

-- name: DeleteCreditPaymentsByCredit :exec
DELETE FROM credit_payments WHERE credit_id = ?;

-- name: CountAppliedCreditPayments :one
SELECT COUNT(*) FROM credit_payments
WHERE credit_id = ? AND is_applied = 1;

-- name: CountCreditPayments :one
SELECT COUNT(*) FROM credit_payments WHERE credit_id = ?;

-- name: ListDueCreditPayments :many
SELECT
    cp.id,
    cp.credit_id,
    cp.transaction_id,
    cp.amount,
    cp.payment_date,
    cp.kind,
    cp.is_applied,
    cp.exclude_from_stats,
    cp.created_at,
    c.user_id,
    c.name AS credit_name,
    c.debit_account_id,
    c.status AS credit_status
FROM credit_payments cp
JOIN credits c ON c.id = cp.credit_id
WHERE cp.is_applied = 0
  AND cp.kind = 'scheduled'
  AND c.status = 'active'
  AND cp.payment_date <= ?
ORDER BY cp.payment_date ASC;

-- name: HasPaymentOnDate :one
SELECT COUNT(*) FROM credit_payments
WHERE credit_id = ?
  AND payment_date = ?
  AND is_applied = 1
  AND kind IN ('early', 'auto', 'retroactive');

-- name: ListCreditPaymentTransactionIDs :many
SELECT transaction_id FROM credit_payments
WHERE credit_id = ? AND transaction_id IS NOT NULL;

-- name: UnlinkCreditPaymentTransactions :exec
UPDATE credit_payments SET transaction_id = NULL WHERE credit_id = ?;

-- name: GetCreditPaymentLinkByTransactionID :one
SELECT
    cp.id AS payment_id,
    cp.credit_id,
    cp.amount AS payment_amount,
    cp.kind AS payment_kind,
    cp.is_applied AS payment_is_applied,
    c.paid_amount,
    c.status AS credit_status
FROM credit_payments cp
JOIN credits c ON c.id = cp.credit_id
WHERE cp.transaction_id = ? AND c.user_id = ?;

-- name: GetCreditPaymentForUser :one
SELECT
    cp.id,
    cp.credit_id,
    cp.transaction_id,
    cp.amount,
    cp.payment_date,
    cp.kind,
    cp.is_applied,
    c.paid_amount,
    c.status AS credit_status
FROM credit_payments cp
JOIN credits c ON c.id = cp.credit_id
WHERE cp.id = ? AND cp.credit_id = ? AND c.user_id = ?;

-- name: UpdateScheduledCreditPaymentAmount :execrows
UPDATE credit_payments
SET amount = ?
WHERE id = ? AND credit_id = ? AND is_applied = 0 AND kind = 'scheduled';

-- name: DeleteCreditPaymentByID :execrows
DELETE FROM credit_payments
WHERE id = ? AND credit_id = ?;

-- name: RevertCreditPaymentLink :execrows
UPDATE credit_payments
SET transaction_id = NULL, is_applied = 0, kind = 'scheduled'
WHERE id = ? AND credit_id = ? AND transaction_id = ? AND is_applied = 1;

-- name: ReopenCredit :execrows
UPDATE credits
SET status = 'active', closed_at = NULL, updated_at = ?
WHERE id = ? AND user_id = ? AND status = 'closed';

-- name: UpdateFutureTransactionAmount :execrows
UPDATE transactions
SET amount = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND kind = 'future';

-- name: UpdateFutureTransactionAccount :execrows
UPDATE transactions
SET account_id = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND kind = 'future';

-- name: ListUsersWithTimezone :many
SELECT id, timezone FROM users;
