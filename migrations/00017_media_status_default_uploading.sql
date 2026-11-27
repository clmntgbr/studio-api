-- +goose Up
-- +goose StatementBegin
ALTER TABLE media_file ALTER COLUMN status SET DEFAULT 'uploading';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE media_file ALTER COLUMN status SET DEFAULT 'pending';
-- +goose StatementEnd
