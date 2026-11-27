-- +goose Up
ALTER TYPE step_kind ADD VALUE IF NOT EXISTS 'ship';

-- +goose Down
-- Postgres cannot remove enum values safely; leave 'ship' in place on rollback.
SELECT 1;
