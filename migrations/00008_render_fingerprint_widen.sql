-- +goose Up
-- Allow superseded fingerprints and full sha256 hex without truncation surprises.
ALTER TABLE render
  ALTER COLUMN fingerprint TYPE VARCHAR(128);

-- +goose Down
ALTER TABLE render
  ALTER COLUMN fingerprint TYPE VARCHAR(64);
