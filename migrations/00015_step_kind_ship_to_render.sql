-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE step_kind ADD VALUE IF NOT EXISTS 'render';

UPDATE processing_step SET kind = 'render' WHERE kind::text = 'ship';
UPDATE processing_step_run SET kind = 'render' WHERE kind::text = 'ship';

-- +goose Down
UPDATE processing_step SET kind = 'ship' WHERE kind::text = 'render';
UPDATE processing_step_run SET kind = 'ship' WHERE kind::text = 'render';
