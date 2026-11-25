-- +goose Up
-- Soften factory defaults: silence on, fillers/repetitions off, standard detection.
ALTER TABLE media_configuration
  ALTER COLUMN detection_level SET DEFAULT 'standard',
  ALTER COLUMN fillers_enabled SET DEFAULT FALSE,
  ALTER COLUMN repetitions_enabled SET DEFAULT FALSE,
  ALTER COLUMN padding_before_ms SET DEFAULT 120,
  ALTER COLUMN padding_after_ms SET DEFAULT 200;

UPDATE media_configuration
SET
  detection_level = 'standard',
  fillers_enabled = FALSE,
  repetitions_enabled = FALSE,
  padding_before_ms = 120,
  padding_after_ms = 200,
  updated_at = NOW()
WHERE detection_level = 'aggressive'
   OR fillers_enabled = TRUE
   OR repetitions_enabled = TRUE
   OR padding_before_ms = 50
   OR padding_after_ms = 150;

-- +goose Down
ALTER TABLE media_configuration
  ALTER COLUMN detection_level SET DEFAULT 'aggressive',
  ALTER COLUMN fillers_enabled SET DEFAULT TRUE,
  ALTER COLUMN repetitions_enabled SET DEFAULT TRUE,
  ALTER COLUMN padding_before_ms SET DEFAULT 50,
  ALTER COLUMN padding_after_ms SET DEFAULT 150;
