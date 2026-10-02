-- name: ListNotificationInterceptBankBindings :many
SELECT account_id, bank_id, package_name
FROM notification_intercept_bank_bindings
WHERE user_id = ?
ORDER BY account_id;

-- name: ListNotificationInterceptCardBindings :many
SELECT bank_id, last4, account_id
FROM notification_intercept_card_bindings
WHERE user_id = ?
ORDER BY bank_id, last4;

-- name: DeleteNotificationInterceptBankBindings :exec
DELETE FROM notification_intercept_bank_bindings
WHERE user_id = ?;

-- name: DeleteNotificationInterceptCardBindings :exec
DELETE FROM notification_intercept_card_bindings
WHERE user_id = ?;

-- name: InsertNotificationInterceptBankBinding :exec
INSERT INTO notification_intercept_bank_bindings (user_id, account_id, bank_id, package_name)
VALUES (?, ?, ?, ?);

-- name: InsertNotificationInterceptCardBinding :exec
INSERT INTO notification_intercept_card_bindings (user_id, bank_id, last4, account_id)
VALUES (?, ?, ?, ?);
