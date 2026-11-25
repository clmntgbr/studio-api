-- +goose Up
-- +goose StatementBegin
ALTER TABLE media_configuration
  ADD COLUMN IF NOT EXISTS master_split TEXT NOT NULL DEFAULT '';

ALTER TABLE render
  ADD COLUMN IF NOT EXISTS output_parts JSONB NOT NULL DEFAULT '[]'::jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE render DROP COLUMN IF EXISTS output_parts;
ALTER TABLE media_configuration DROP COLUMN IF EXISTS master_split;
-- +goose StatementEnd
