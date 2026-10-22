-- +goose Up
-- Backfill of upcoming_run_ats JSON is applied at startup via subscription.BackfillUpcomingRunAts.
ALTER TABLE subscriptions ADD COLUMN upcoming_run_ats TEXT NOT NULL DEFAULT '[]';

-- +goose Down
-- Forward-only migration.
