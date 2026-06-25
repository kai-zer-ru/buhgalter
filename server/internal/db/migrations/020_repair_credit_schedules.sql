-- +goose Up
-- Repair of short credit schedules is applied at startup via credit.RepairShortSchedules.

-- +goose Down
-- No-op: appended credit_payments are not rolled back automatically.
