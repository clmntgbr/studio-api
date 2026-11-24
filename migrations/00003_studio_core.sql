-- +goose Up
-- +goose StatementBegin
CREATE TYPE project_status  AS ENUM ('draft','processing','ready','rendering','completed','failed');
CREATE TYPE media_status    AS ENUM ('pending','fetching','processing','ready','failed');
CREATE TYPE step_kind       AS ENUM ('fetch','probe','audio','proxy','waveform','silence','transcribe','text','llm','scenes','reframe');
CREATE TYPE step_status     AS ENUM ('pending','queued','running','completed','failed','skipped','cancelled');
CREATE TYPE detection_kind  AS ENUM ('silence','filler','repetition','false_start','sponsor','ad');
CREATE TYPE decision_action AS ENUM ('remove','suggest','keep');
CREATE TYPE decision_origin AS ENUM ('auto','user');
CREATE TYPE cut_reason      AS ENUM ('silence','filler','repetition','false_start','sponsor','ad','manual','micro');
CREATE TYPE timeline_status AS ENUM ('draft','validated','superseded');
CREATE TYPE render_kind     AS ENUM ('master','clip','audio','custom');
CREATE TYPE render_strategy AS ENUM ('full_render','stream_copy');
CREATE TYPE aspect_ratio    AS ENUM ('16:9','9:16','1:1','4:5','custom');
CREATE TYPE crop_mode       AS ENUM ('none','center','left','right','custom','fit_blur','face_track','speaker_track');

CREATE TABLE project (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  status project_status NOT NULL DEFAULT 'draft',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE media_file (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  source_kind VARCHAR(10) NOT NULL DEFAULT 'upload',
  source_url TEXT,
  title TEXT,
  original_filename TEXT,
  storage_key TEXT,
  mime_type VARCHAR(100),
  duration_ms BIGINT,
  width INTEGER,
  height INTEGER,
  fps NUMERIC(8,3),
  video_codec VARCHAR(50),
  audio_codec VARCHAR(50),
  has_audio BOOLEAN,
  size_bytes BIGINT,
  status media_status NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (duration_ms IS NULL OR duration_ms >= 0)
);
CREATE INDEX idx_media_file_project ON media_file(project_id, position);

-- Derived files. Audio kinds: analysis | transcription | master (three tracks).
CREATE TABLE media_asset (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind VARCHAR(30) NOT NULL,
  storage_key TEXT NOT NULL,
  size_bytes BIGINT,
  meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (media_file_id, kind)
);

CREATE TABLE media_configuration (
  media_file_id UUID PRIMARY KEY REFERENCES media_file(id) ON DELETE CASCADE,
  silence_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  threshold_mode VARCHAR(10) NOT NULL DEFAULT 'auto',
  silence_threshold_db NUMERIC(5,1),
  detection_level VARCHAR(12) NOT NULL DEFAULT 'aggressive',
  silence_min_duration_ms INTEGER NOT NULL DEFAULT 500,
  padding_before_ms INTEGER NOT NULL DEFAULT 50,
  padding_after_ms INTEGER NOT NULL DEFAULT 150,
  speech_min_duration_ms INTEGER NOT NULL DEFAULT 300,
  fillers_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  repetitions_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  false_starts_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  sponsors_action decision_action NOT NULL DEFAULT 'suggest',
  ads_action decision_action NOT NULL DEFAULT 'suggest',
  language VARCHAR(10),
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE processing_step (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind step_kind NOT NULL,
  status step_status NOT NULL DEFAULT 'pending',
  attempt INTEGER NOT NULL DEFAULT 0,
  input_hash VARCHAR(64),
  progress NUMERIC(5,2) NOT NULL DEFAULT 0,
  error_message TEXT,
  queued_at TIMESTAMPTZ,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (media_file_id, kind)
);

CREATE TABLE transcript (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL UNIQUE REFERENCES media_file(id) ON DELETE CASCADE,
  language VARCHAR(10),
  text TEXT,
  provider VARCHAR(30) NOT NULL DEFAULT 'assemblyai',
  provider_job_id VARCHAR(100),
  status step_status NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transcript_word (
  id BIGSERIAL PRIMARY KEY,
  transcript_id UUID NOT NULL REFERENCES transcript(id) ON DELETE CASCADE,
  word_index INTEGER NOT NULL,
  text TEXT NOT NULL,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  confidence NUMERIC(5,4),
  speaker VARCHAR(10),
  CHECK (source_start_ms >= 0 AND source_end_ms >= source_start_ms),
  UNIQUE (transcript_id, word_index)
);
CREATE INDEX idx_transcript_word_time ON transcript_word(transcript_id, source_start_ms);

CREATE TABLE detection (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind detection_kind NOT NULL,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  confidence NUMERIC(5,4),
  detector VARCHAR(60) NOT NULL,
  first_word_index INTEGER,
  last_word_index INTEGER,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (source_start_ms >= 0 AND source_end_ms > source_start_ms)
);
CREATE INDEX idx_detection_media_time ON detection(media_file_id, source_start_ms);
CREATE INDEX idx_detection_media_kind ON detection(media_file_id, kind);

CREATE TABLE user_override (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind detection_kind,
  action decision_action NOT NULL,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (action IN ('keep','remove') AND source_end_ms > source_start_ms)
);
CREATE INDEX idx_override_media ON user_override(media_file_id, source_start_ms);

CREATE TABLE timeline (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,
  status timeline_status NOT NULL DEFAULT 'draft',
  fingerprint VARCHAR(64) NOT NULL,
  duration_ms BIGINT NOT NULL,
  validated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (project_id, version),
  UNIQUE (project_id, fingerprint)
);

CREATE TABLE timeline_segment (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  timeline_id UUID NOT NULL REFERENCES timeline(id) ON DELETE CASCADE,
  segment_index INTEGER NOT NULL,
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  output_start_ms BIGINT NOT NULL,
  output_end_ms BIGINT NOT NULL,
  removed_before_reason cut_reason,
  CHECK (source_end_ms > source_start_ms AND output_end_ms > output_start_ms),
  UNIQUE (timeline_id, segment_index)
);
CREATE INDEX idx_segment_source ON timeline_segment(media_file_id, source_start_ms);

CREATE TABLE timeline_decision (
  timeline_id UUID NOT NULL REFERENCES timeline(id) ON DELETE CASCADE,
  detection_id UUID NOT NULL REFERENCES detection(id) ON DELETE CASCADE,
  action decision_action NOT NULL,
  origin decision_origin NOT NULL,
  reason TEXT,
  PRIMARY KEY (timeline_id, detection_id)
);

CREATE TABLE clip (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  origin VARCHAR(10) NOT NULL DEFAULT 'llm',
  score NUMERIC(5,4),
  hook TEXT,
  reason TEXT,
  title TEXT,
  description TEXT,
  hashtags TEXT[],
  selected BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (source_end_ms > source_start_ms),
  CHECK (score IS NULL OR score BETWEEN 0 AND 1)
);

CREATE TABLE llm_artifact (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind VARCHAR(30) NOT NULL,
  payload JSONB NOT NULL,
  provider VARCHAR(30),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subtitle_style (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID REFERENCES project(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  font_family VARCHAR(255),
  font_size INTEGER,
  base_color VARCHAR(9),
  highlight_color VARCHAR(9),
  outline_color VARCHAR(9),
  outline_size INTEGER,
  words_per_line SMALLINT,
  position VARCHAR(20),
  uppercase BOOLEAN NOT NULL DEFAULT TRUE,
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE overlay (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  clip_id UUID REFERENCES clip(id) ON DELETE CASCADE,
  kind VARCHAR(30) NOT NULL,
  text TEXT,
  storage_key TEXT,
  start_ms BIGINT NOT NULL,
  end_ms BIGINT NOT NULL,
  position VARCHAR(30),
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  CHECK (start_ms >= 0 AND end_ms > start_ms)
);

CREATE TABLE output_preset (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(100) NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  aspect_ratio aspect_ratio NOT NULL,
  crop_mode crop_mode NOT NULL DEFAULT 'none',
  video_codec VARCHAR(50) NOT NULL DEFAULT 'h264',
  audio_codec VARCHAR(50) NOT NULL DEFAULT 'aac',
  fps NUMERIC(8,3),
  video_bitrate BIGINT,
  audio_bitrate BIGINT,
  loudness_lufs NUMERIC(4,1),
  settings JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE render (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  timeline_id UUID NOT NULL REFERENCES timeline(id) ON DELETE RESTRICT,
  kind render_kind NOT NULL,
  strategy render_strategy NOT NULL DEFAULT 'full_render',
  clip_id UUID REFERENCES clip(id) ON DELETE SET NULL,
  output_preset_id UUID NOT NULL REFERENCES output_preset(id) ON DELETE RESTRICT,
  subtitle_style_id UUID REFERENCES subtitle_style(id) ON DELETE SET NULL,
  plan JSONB NOT NULL,
  fingerprint VARCHAR(64) NOT NULL,
  status step_status NOT NULL DEFAULT 'pending',
  progress NUMERIC(5,2) NOT NULL DEFAULT 0,
  storage_key TEXT,
  duration_ms BIGINT,
  size_bytes BIGINT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE (project_id, fingerprint)
);
CREATE INDEX idx_render_status ON render(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS render;
DROP TABLE IF EXISTS output_preset;
DROP TABLE IF EXISTS overlay;
DROP TABLE IF EXISTS subtitle_style;
DROP TABLE IF EXISTS llm_artifact;
DROP TABLE IF EXISTS clip;
DROP TABLE IF EXISTS timeline_decision;
DROP TABLE IF EXISTS timeline_segment;
DROP TABLE IF EXISTS timeline;
DROP TABLE IF EXISTS user_override;
DROP TABLE IF EXISTS detection;
DROP TABLE IF EXISTS transcript_word;
DROP TABLE IF EXISTS transcript;
DROP TABLE IF EXISTS processing_step;
DROP TABLE IF EXISTS media_configuration;
DROP TABLE IF EXISTS media_asset;
DROP TABLE IF EXISTS media_file;
DROP TABLE IF EXISTS project;

DROP TYPE IF EXISTS crop_mode;
DROP TYPE IF EXISTS aspect_ratio;
DROP TYPE IF EXISTS render_strategy;
DROP TYPE IF EXISTS render_kind;
DROP TYPE IF EXISTS timeline_status;
DROP TYPE IF EXISTS cut_reason;
DROP TYPE IF EXISTS decision_origin;
DROP TYPE IF EXISTS decision_action;
DROP TYPE IF EXISTS detection_kind;
DROP TYPE IF EXISTS step_status;
DROP TYPE IF EXISTS step_kind;
DROP TYPE IF EXISTS media_status;
DROP TYPE IF EXISTS project_status;
-- +goose StatementEnd
