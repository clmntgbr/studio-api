-- +goose Up
-- +goose StatementBegin
ALTER TABLE media_configuration
  ALTER COLUMN sponsors_action SET DEFAULT 'remove',
  ALTER COLUMN ads_action SET DEFAULT 'remove';

UPDATE media_configuration
SET
  sponsors_action = 'remove',
  ads_action = 'remove',
  language = NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE media_configuration
  ALTER COLUMN sponsors_action SET DEFAULT 'suggest',
  ALTER COLUMN ads_action SET DEFAULT 'suggest';
-- +goose StatementEnd
