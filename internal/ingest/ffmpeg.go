package ingest

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
)

func runFFmpeg(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", append([]string{"-hide_banner", "-y"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg %v: %w: %s", args[:min(4, len(args))], err, truncate(string(out), 1500))
	}
	return nil
}

func MakeMezzanine(ctx context.Context, src, dest string) error {
	return runFFmpeg(ctx,
		"-i", src,
		"-map", "0:v:0", "-map", "0:a:0?",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "18",
		"-fps_mode", "cfr",
		"-c:a", "aac", "-b:a", "192k",
		"-movflags", "+faststart",
		dest,
	)
}

func MakeProxy(ctx context.Context, src, dest string, spec ProxySpec) error {
	height := spec.Height
	if height <= 0 {
		height = 720
	}
	gop := spec.GOPFrames
	if gop <= 0 {
		gop = 15
	}
	crf := spec.CRF
	if crf <= 0 {
		crf = 26
	}
	scale := fmt.Sprintf("scale=-2:%d:flags=lanczos", height)
	return runFFmpeg(ctx,
		"-i", src,
		"-map", "0:v:0", "-map", "0:a:0?",
		"-vf", scale,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", strconv.Itoa(crf),
		"-pix_fmt", "yuv420p",
		"-fps_mode", "cfr",
		"-g", strconv.Itoa(gop), "-keyint_min", strconv.Itoa(gop), "-sc_threshold", "0",
		"-c:a", "aac", "-b:a", "128k",
		"-movflags", "+faststart",
		dest,
	)
}

func ExtractAnalysisAudio(ctx context.Context, src, dest string, spec AudioAnalysisSpec) error {
	sr := spec.SampleRate
	if sr <= 0 {
		sr = 16000
	}
	ch := spec.Channels
	if ch <= 0 {
		ch = 1
	}
	return runFFmpeg(ctx,
		"-i", src,
		"-vn",
		"-ac", strconv.Itoa(ch),
		"-ar", strconv.Itoa(sr),
		dest,
	)
}

func ExtractTranscriptionAudio(ctx context.Context, src, dest string, spec AudioTranscriptionSpec) error {
	br := spec.BitrateKbps
	if br <= 0 {
		br = 96
	}
	return runFFmpeg(ctx,
		"-i", src,
		"-vn",
		"-c:a", "aac",
		"-b:a", fmt.Sprintf("%dk", br),
		dest,
	)
}

func ExtractMasterAudio(ctx context.Context, src, dest string, spec AudioMasterSpec) error {
	sr := spec.SampleRate
	if sr <= 0 {
		sr = 48000
	}
	ch := spec.Channels
	if ch <= 0 {
		ch = 2
	}
	return runFFmpeg(ctx,
		"-i", src,
		"-vn",
		"-ac", strconv.Itoa(ch),
		"-ar", strconv.Itoa(sr),
		"-c:a", "flac",
		dest,
	)
}

// MakeSprite builds a tiled jpeg contact sheet from the proxy/source.
func MakeSprite(ctx context.Context, src, dest string, durationMs int64) (SpriteOut, error) {
	intervalMs := 2000
	tileW, tileH, cols := 160, 90, 10
	if durationMs > 0 && durationMs < 20000 {
		intervalMs = 1000
	}
	fps := 1000.0 / float64(intervalMs)
	filter := fmt.Sprintf(
		"fps=%.6f,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,tile=%dx100",
		fps, tileW, tileH, tileW, tileH, cols,
	)
	if err := runFFmpeg(ctx,
		"-i", src,
		"-vf", filter,
		"-frames:v", "1",
		"-q:v", "3",
		dest,
	); err != nil {
		return SpriteOut{}, err
	}
	return SpriteOut{
		Key:        filepath.Base(dest),
		IntervalMs: intervalMs,
		TileW:      tileW,
		TileH:      tileH,
		Cols:       cols,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
