-- +goose Up
-- The application ensures the highlighted column exists during startup as well,
-- so this migration is intentionally a no-op for idempotent upgrades.

-- +goose Down
-- SQLite does not support dropping a column directly, so the rollback is left as a no-op.
