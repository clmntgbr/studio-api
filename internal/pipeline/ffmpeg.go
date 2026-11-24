package pipeline

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func runFFmpeg(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", append([]string{"-hide_banner", "-y"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		n := 4
		if len(args) < n {
			n = len(args)
		}
		return fmt.Errorf("ffmpeg %v: %w: %s", args[:n], err, truncate(string(out), 1500))
	}
	return nil
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

// MakeWaveformPeaks writes peaks.bin: 100 pairs/sec of min/max uint8 from mono s16le PCM.
func MakeWaveformPeaks(ctx context.Context, audioSrc, dest string) error {
	pcmPath := dest + ".pcm"
	defer os.Remove(pcmPath)

	if err := runFFmpeg(ctx,
		"-i", audioSrc,
		"-ac", "1",
		"-ar", "100",
		"-f", "s16le",
		pcmPath,
	); err != nil {
		return err
	}

	raw, err := os.ReadFile(pcmPath)
	if err != nil {
		return err
	}
	if len(raw) < 2 {
		return fmt.Errorf("empty pcm for waveform")
	}

	out := make([]byte, 0, (len(raw)/2)*2)
	for i := 0; i+1 < len(raw); i += 2 {
		sample := int16(binary.LittleEndian.Uint16(raw[i : i+2]))
		// Map [-32768,32767] → [0,255]
		u := uint8((int(sample) + 32768) >> 8)
		out = append(out, u, u) // min=max per sample at 100 Hz
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, out, 0o644)
}

func ContentTypeForExt(ext string) string {
	switch ext {
	case "mp4", "m4v":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	case "webm":
		return "video/webm"
	case "mkv":
		return "video/x-matroska"
	case "wav":
		return "audio/wav"
	case "m4a":
		return "audio/mp4"
	case "flac":
		return "audio/flac"
	default:
		return "application/octet-stream"
	}
}
