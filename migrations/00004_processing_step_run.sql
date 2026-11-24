-- +goose Up
-- +goose StatementBegin
CREATE TABLE processing_step_run (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind step_kind NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 1,
  status step_status NOT NULL,
  started_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  duration_ms BIGINT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (duration_ms IS NULL OR duration_ms >= 0),
  CHECK (completed_at IS NULL OR completed_at >= started_at)
);
CREATE INDEX idx_processing_step_run_media ON processing_step_run(media_file_id, kind, created_at DESC);
CREATE INDEX idx_processing_step_run_media_created ON processing_step_run(media_file_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS processing_step_run;
-- +goose StatementEnd
