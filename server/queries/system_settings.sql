-- name: GetRegistrationEnabled :one
SELECT registration_enabled FROM system_settings WHERE id = 1;

-- name: GetAdminSettings :one
SELECT registration_enabled, external_url, notification_secret_key
FROM system_settings WHERE id = 1;

-- name: GetSetupStatus :one
SELECT registration_enabled, external_url FROM system_settings WHERE id = 1;

-- name: GetDiagnosticsSettings :one
SELECT external_url, previous_app_version FROM system_settings WHERE id = 1;

-- name: GetExternalURL :one
SELECT external_url FROM system_settings WHERE id = 1;

-- name: GetNotificationSecretKey :one
SELECT notification_secret_key FROM system_settings WHERE id = 1;

-- name: GetIsConfigured :one
SELECT is_configured FROM system_settings WHERE id = 1;

-- name: GetAppVersion :one
SELECT app_version FROM system_settings WHERE id = 1;

-- name: GetBackupRetention :one
SELECT backup_retention FROM system_settings WHERE id = 1;

-- name: GetBackupSettings :one
SELECT backup_enabled, backup_time, backup_retention FROM system_settings WHERE id = 1;

-- name: UpdateAdminSettings :exec
UPDATE system_settings
SET registration_enabled = ?, external_url = ?, updated_at = datetime('now')
WHERE id = 1;

-- name: UpdateNotificationSecretKey :exec
UPDATE system_settings
SET notification_secret_key = ?, updated_at = datetime('now')
WHERE id = 1;

-- name: CompleteSetup :exec
UPDATE system_settings
SET is_configured = 1, external_url = ?, registration_enabled = ?, updated_at = datetime('now')
WHERE id = 1;

-- name: UpdateDBPath :exec
UPDATE system_settings SET db_path = ? WHERE id = 1;

-- name: SetAppVersionFirst :exec
UPDATE system_settings
SET app_version = ?, previous_app_version = NULL, updated_at = datetime('now')
WHERE id = 1;

-- name: SetAppVersionUpgrade :exec
UPDATE system_settings
SET previous_app_version = ?, app_version = ?, updated_at = datetime('now')
WHERE id = 1;

-- name: UpdateBackupSettings :exec
UPDATE system_settings
SET backup_enabled = ?, backup_time = ?, backup_retention = ?, updated_at = datetime('now')
WHERE id = 1;
