-- +goose Up
-- Credit cards cannot be the default (primary) account.
UPDATE accounts SET is_primary = 0 WHERE type = 'credit_card' AND is_primary = 1;

-- Users left without a primary cash/bank account get the oldest eligible one.
UPDATE accounts
SET is_primary = 1
WHERE id IN (
    SELECT a.id
    FROM accounts a
    WHERE a.status = 'active'
      AND a.type IN ('cash', 'bank')
      AND NOT EXISTS (
          SELECT 1
          FROM accounts p
          WHERE p.user_id = a.user_id
            AND p.status = 'active'
            AND p.is_primary = 1
      )
      AND a.id = (
          SELECT b.id
          FROM accounts b
          WHERE b.user_id = a.user_id
            AND b.status = 'active'
            AND b.type IN ('cash', 'bank')
          ORDER BY b.created_at, b.name
          LIMIT 1
      )
);

-- +goose Down
-- Irreversible data fix; no-op.
SELECT 1;
