-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE media_status ADD VALUE IF NOT EXISTS 'uploading';
ALTER TYPE media_status ADD VALUE IF NOT EXISTS 'rendering';

UPDATE media_file SET status = 'uploading' WHERE status::text = 'pending';

-- +goose Down
UPDATE media_file SET status = 'pending' WHERE status::text = 'uploading';
UPDATE media_file SET status = 'ready' WHERE status::text = 'rendering';
