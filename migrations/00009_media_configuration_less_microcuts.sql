-- +goose Up
-- Fewer micro-cuts: longer min silence / min speech keeps.
ALTER TABLE media_configuration
  ALTER COLUMN silence_min_duration_ms SET DEFAULT 700,
  ALTER COLUMN speech_min_duration_ms SET DEFAULT 800;

UPDATE media_configuration
SET
  silence_min_duration_ms = GREATEST(silence_min_duration_ms, 700),
  speech_min_duration_ms = GREATEST(speech_min_duration_ms, 800),
  updated_at = NOW()
WHERE silence_min_duration_ms < 700
   OR speech_min_duration_ms < 800;

-- +goose Down
ALTER TABLE media_configuration
  ALTER COLUMN silence_min_duration_ms SET DEFAULT 500,
  ALTER COLUMN speech_min_duration_ms SET DEFAULT 300;
