-- +goose Up
-- Timeline and Ship/render are owned by a media file, not shared across a project.

ALTER TABLE timeline
  ADD COLUMN media_file_id UUID REFERENCES media_file(id) ON DELETE CASCADE;

UPDATE timeline t
SET media_file_id = sub.media_file_id
FROM (
  SELECT DISTINCT ON (timeline_id) timeline_id, media_file_id
  FROM timeline_segment
  ORDER BY timeline_id, segment_index ASC
) AS sub
WHERE t.id = sub.timeline_id
  AND t.media_file_id IS NULL;

-- Orphan timelines without segments cannot be media-scoped — drop them.
DELETE FROM timeline_decision
WHERE timeline_id IN (SELECT id FROM timeline WHERE media_file_id IS NULL);
DELETE FROM timeline_segment
WHERE timeline_id IN (SELECT id FROM timeline WHERE media_file_id IS NULL);
DELETE FROM timeline WHERE media_file_id IS NULL;

ALTER TABLE timeline
  ALTER COLUMN media_file_id SET NOT NULL;

ALTER TABLE timeline
  DROP CONSTRAINT IF EXISTS timeline_project_id_version_key;
ALTER TABLE timeline
  DROP CONSTRAINT IF EXISTS timeline_project_id_fingerprint_key;

ALTER TABLE timeline
  ADD CONSTRAINT timeline_media_file_id_version_key UNIQUE (media_file_id, version);
ALTER TABLE timeline
  ADD CONSTRAINT timeline_media_file_id_fingerprint_key UNIQUE (media_file_id, fingerprint);

CREATE INDEX IF NOT EXISTS idx_timeline_media_head
  ON timeline (media_file_id, version DESC);

ALTER TABLE render
  ADD COLUMN media_file_id UUID REFERENCES media_file(id) ON DELETE CASCADE;

UPDATE render r
SET media_file_id = t.media_file_id
FROM timeline t
WHERE r.timeline_id = t.id
  AND r.media_file_id IS NULL;

-- Fallback: plan JSON mediaId (deferred reframe markers).
UPDATE render
SET media_file_id = NULLIF(plan->>'mediaId', '')::uuid
WHERE media_file_id IS NULL
  AND plan ? 'mediaId'
  AND NULLIF(plan->>'mediaId', '') IS NOT NULL;

DELETE FROM render WHERE media_file_id IS NULL;

ALTER TABLE render
  ALTER COLUMN media_file_id SET NOT NULL;

ALTER TABLE render
  DROP CONSTRAINT IF EXISTS render_project_id_fingerprint_key;

ALTER TABLE render
  ADD CONSTRAINT render_media_file_id_fingerprint_key UNIQUE (media_file_id, fingerprint);

CREATE INDEX IF NOT EXISTS idx_render_media_created
  ON render (media_file_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_render_media_created;
ALTER TABLE render DROP CONSTRAINT IF EXISTS render_media_file_id_fingerprint_key;
ALTER TABLE render
  ADD CONSTRAINT render_project_id_fingerprint_key UNIQUE (project_id, fingerprint);
ALTER TABLE render DROP COLUMN IF EXISTS media_file_id;

DROP INDEX IF EXISTS idx_timeline_media_head;
ALTER TABLE timeline DROP CONSTRAINT IF EXISTS timeline_media_file_id_fingerprint_key;
ALTER TABLE timeline DROP CONSTRAINT IF EXISTS timeline_media_file_id_version_key;
ALTER TABLE timeline
  ADD CONSTRAINT timeline_project_id_fingerprint_key UNIQUE (project_id, fingerprint);
ALTER TABLE timeline
  ADD CONSTRAINT timeline_project_id_version_key UNIQUE (project_id, version);
ALTER TABLE timeline DROP COLUMN IF EXISTS media_file_id;
