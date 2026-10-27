-- name: ListAccountsNetWorthByUser :many
SELECT id, type, initial_balance, credit_limit, created_at, currency
FROM accounts
WHERE status != 'deleted' AND currency = sqlc.arg('currency') AND user_id IN (sqlc.slice('owner_ids'));

-- name: NetWorthDebtsByUser :many
SELECT
    d.direction,
    d.amount + CAST(COALESCE((
        SELECT SUM(t.amount)
        FROM debt_transactions dtx
        INNER JOIN transactions t ON t.id = dtx.transaction_id AND t.user_id = d.user_id
        WHERE dtx.debt_id = d.id
          AND dtx.role = 'settle'
          AND t.transaction_date > sqlc.arg(as_of)
    ), 0) AS INTEGER) AS amount_at_as_of
FROM debts d
WHERE d.currency = sqlc.arg(currency)
  AND d.debt_date <= sqlc.arg(as_of)
  AND d.user_id IN (sqlc.slice('owner_ids'))
  AND NOT (
    d.is_settled = 1
    AND d.settled_at IS NOT NULL
    AND d.settled_at <= sqlc.arg(as_of)
  );

-- name: NetWorthCreditsByUser :many
SELECT
    c.id,
    c.principal_amount,
    CAST(COALESCE((
        SELECT SUM(cp.amount)
        FROM credit_payments cp
        WHERE cp.credit_id = c.id
          AND cp.is_applied = 1
          AND cp.payment_date <= sqlc.arg(as_of)
    ), 0) AS INTEGER) AS paid_at_as_of
FROM credits c
JOIN accounts a ON a.id = c.debit_account_id AND a.currency = sqlc.arg(currency)
WHERE c.issue_date <= sqlc.arg(as_of)
  AND c.user_id IN (sqlc.slice('owner_ids'))
  AND (
    c.status = 'active'
    OR (c.status = 'closed' AND c.closed_at IS NOT NULL AND c.closed_at > sqlc.arg(as_of))
  );
