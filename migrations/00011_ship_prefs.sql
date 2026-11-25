-- +goose Up
-- +goose StatementBegin
ALTER TABLE media_configuration
  ADD COLUMN IF NOT EXISTS burn_subtitles BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS output_preset_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE media_configuration
  DROP COLUMN IF EXISTS output_preset_id,
  DROP COLUMN IF EXISTS burn_subtitles;
-- +goose StatementEnd
