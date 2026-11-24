-- +goose Up
-- +goose StatementBegin
INSERT INTO output_preset (id, name, width, height, aspect_ratio, crop_mode, video_codec, audio_codec, fps, loudness_lufs, settings)
VALUES
  ('01960000-0000-7000-8000-0000000000b1', 'YouTube', 1920, 1080, '16:9', 'none', 'h264', 'aac', 30, -14, '{"crf":18,"preset":"medium","fadeMs":15,"profile":"youtube"}'::jsonb),
  ('01960000-0000-7000-8000-0000000000b2', 'Podcast', 1920, 1080, '16:9', 'none', 'h264', 'aac', 30, -16, '{"crf":20,"preset":"veryfast","fadeMs":20,"profile":"podcast"}'::jsonb),
  ('01960000-0000-7000-8000-0000000000b3', 'Vlog', 1920, 1080, '16:9', 'none', 'h264', 'aac', 30, -14, '{"crf":20,"preset":"veryfast","fadeMs":12,"profile":"vlog"}'::jsonb),
  ('01960000-0000-7000-8000-0000000000b4', 'Short vertical', 1080, 1920, '9:16', 'center', 'h264', 'aac', 30, -14, '{"crf":20,"preset":"veryfast","fadeMs":10,"profile":"short"}'::jsonb)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM output_preset WHERE id IN (
  '01960000-0000-7000-8000-0000000000b1',
  '01960000-0000-7000-8000-0000000000b2',
  '01960000-0000-7000-8000-0000000000b3',
  '01960000-0000-7000-8000-0000000000b4'
);
-- +goose StatementEnd
