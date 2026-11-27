-- +goose Up
-- +goose StatementBegin

ALTER TABLE media_file DROP CONSTRAINT IF EXISTS media_file_project_id_fkey;
ALTER TABLE timeline DROP CONSTRAINT IF EXISTS timeline_project_id_fkey;
ALTER TABLE render DROP CONSTRAINT IF EXISTS render_project_id_fkey;
ALTER TABLE clip DROP CONSTRAINT IF EXISTS clip_project_id_fkey;
ALTER TABLE overlay DROP CONSTRAINT IF EXISTS overlay_project_id_fkey;
ALTER TABLE subtitle_style DROP CONSTRAINT IF EXISTS subtitle_style_project_id_fkey;

DROP INDEX IF EXISTS idx_media_file_project;

ALTER TABLE media_file DROP COLUMN IF EXISTS project_id;
ALTER TABLE timeline DROP COLUMN IF EXISTS project_id;
ALTER TABLE render DROP COLUMN IF EXISTS project_id;
ALTER TABLE clip DROP COLUMN IF EXISTS project_id;
ALTER TABLE overlay DROP COLUMN IF EXISTS project_id;
ALTER TABLE subtitle_style DROP COLUMN IF EXISTS project_id;

DROP TABLE IF EXISTS project;
DROP TYPE IF EXISTS project_status;

ALTER TABLE users DROP COLUMN IF EXISTS active_project_id;

CREATE INDEX IF NOT EXISTS idx_media_file_created ON media_file(created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_media_file_created;

CREATE TYPE project_status AS ENUM ('draft','processing','ready','rendering','completed','failed');

CREATE TABLE project (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  status project_status NOT NULL DEFAULT 'draft',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE media_file ADD COLUMN project_id UUID;
ALTER TABLE timeline ADD COLUMN project_id UUID;
ALTER TABLE render ADD COLUMN project_id UUID;
ALTER TABLE clip ADD COLUMN project_id UUID;
ALTER TABLE overlay ADD COLUMN project_id UUID;
ALTER TABLE subtitle_style ADD COLUMN project_id UUID;

-- +goose StatementEnd