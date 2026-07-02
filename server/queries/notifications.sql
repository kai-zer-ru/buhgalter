-- name: EnsureNotificationSettings :exec
INSERT INTO notification_settings (user_id)
VALUES (?)
ON CONFLICT(user_id) DO NOTHING;

-- name: GetNotificationSettings :one
SELECT
    user_id,
    telegram_enabled,
    telegram_bot_token,
    telegram_chat_id,
    max_enabled,
    max_provider,
    max_token,
    max_user_id,
    max_recipient_id,
    trigger_debt,
    trigger_credit,
    trigger_planned,
    trigger_negative_balance,
    trigger_budget,
    trigger_auto_topup_disabled,
    trigger_user_registration,
    trigger_password_reset,
    debt_days_before,
    my_debt_overdue_days_limit,
    owed_debt_overdue_start_after_days,
    owed_debt_overdue_days_limit,
    credit_days_before,
    notification_time_local,
    updated_at
FROM notification_settings
WHERE user_id = ?;

-- name: UpsertNotificationSettings :exec
INSERT INTO notification_settings (
    user_id,
    telegram_enabled,
    telegram_bot_token,
    telegram_chat_id,
    max_enabled,
    max_provider,
    max_token,
    max_user_id,
    max_recipient_id,
    trigger_debt,
    trigger_credit,
    trigger_planned,
    trigger_negative_balance,
    trigger_budget,
    trigger_auto_topup_disabled,
    trigger_user_registration,
    trigger_password_reset,
    debt_days_before,
    my_debt_overdue_days_limit,
    owed_debt_overdue_start_after_days,
    owed_debt_overdue_days_limit,
    credit_days_before,
    notification_time_local,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    telegram_enabled = excluded.telegram_enabled,
    telegram_bot_token = excluded.telegram_bot_token,
    telegram_chat_id = excluded.telegram_chat_id,
    max_enabled = excluded.max_enabled,
    max_provider = excluded.max_provider,
    max_token = excluded.max_token,
    max_user_id = excluded.max_user_id,
    max_recipient_id = excluded.max_recipient_id,
    trigger_debt = excluded.trigger_debt,
    trigger_credit = excluded.trigger_credit,
    trigger_planned = excluded.trigger_planned,
    trigger_negative_balance = excluded.trigger_negative_balance,
    trigger_budget = excluded.trigger_budget,
    trigger_auto_topup_disabled = excluded.trigger_auto_topup_disabled,
    trigger_user_registration = excluded.trigger_user_registration,
    trigger_password_reset = excluded.trigger_password_reset,
    debt_days_before = excluded.debt_days_before,
    my_debt_overdue_days_limit = excluded.my_debt_overdue_days_limit,
    owed_debt_overdue_start_after_days = excluded.owed_debt_overdue_start_after_days,
    owed_debt_overdue_days_limit = excluded.owed_debt_overdue_days_limit,
    credit_days_before = excluded.credit_days_before,
    notification_time_local = excluded.notification_time_local,
    updated_at = excluded.updated_at;

-- name: UpsertNotificationTemplate :exec
INSERT INTO notification_templates (user_id, trigger_type, template, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(user_id, trigger_type) DO UPDATE SET
    template = excluded.template,
    updated_at = excluded.updated_at;

-- name: ListNotificationTemplates :many
SELECT user_id, trigger_type, template, updated_at
FROM notification_templates
WHERE user_id = ?;

-- name: DeleteNotificationTemplate :execrows
DELETE FROM notification_templates
WHERE user_id = ? AND trigger_type = ?;

-- name: DeleteNotificationTemplatesByUser :execrows
DELETE FROM notification_templates
WHERE user_id = ?;

-- name: InsertNotificationLog :exec
INSERT INTO notification_log (
    id,
    user_id,
    trigger_type,
    channel,
    entity_id,
    dedup_date,
    status,
    message,
    created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ExistsNotificationDedup :one
SELECT COUNT(*)
FROM notification_log
WHERE user_id = ?
  AND trigger_type = ?
  AND channel = ?
  AND entity_id = ?
  AND dedup_date = ?;
