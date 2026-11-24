-- +goose Up
-- +goose StatementBegin
INSERT INTO output_preset (id, name, width, height, aspect_ratio, crop_mode, video_codec, audio_codec, fps, loudness_lufs, settings)
VALUES
  ('01960000-0000-7000-8000-0000000000a1', 'Master 16:9', 1920, 1080, '16:9', 'none', 'h264', 'aac', 30, -14, '{"crf":20,"preset":"veryfast","fadeMs":15}'::jsonb),
  ('01960000-0000-7000-8000-0000000000a2', 'Vertical 9:16', 1080, 1920, '9:16', 'center', 'h264', 'aac', 30, -14, '{"crf":20,"preset":"veryfast","fadeMs":15}'::jsonb),
  ('01960000-0000-7000-8000-0000000000a3', 'Audio only', 0, 0, '16:9', 'none', 'none', 'aac', 0, -14, '{"kind":"audio"}'::jsonb)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM output_preset WHERE id IN (
  '01960000-0000-7000-8000-0000000000a1',
  '01960000-0000-7000-8000-0000000000a2',
  '01960000-0000-7000-8000-0000000000a3'
);
-- +goose StatementEnd
