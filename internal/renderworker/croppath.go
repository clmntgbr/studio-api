package renderworker

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type cropSample struct {
	TMs  int64   `json:"tMs"`
	CX   float64 `json:"cx"`
	CY   float64 `json:"cy"`
	Zoom float64 `json:"zoom"`
}

type cropPathDoc struct {
	FrameWidth  int          `json:"frameWidth"`
	FrameHeight int          `json:"frameHeight"`
	Samples     []cropSample `json:"samples"`
}

func loadCropPath(path string) (*cropPathDoc, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc cropPathDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func probeVideoSize(ctxPath string) (w, h int, err error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=p=0:s=x", ctxPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe: %w: %s", err, string(out))
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe size: %s", string(out))
	}
	w, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	h, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
}

// scaledFillSize is the intermediate size after scale=W:H:force_original_aspect_ratio=increase.
func scaledFillSize(srcW, srcH, outW, outH int) (int, int) {
	if srcW <= 0 || srcH <= 0 {
		return outW, outH
	}
	fx := float64(outW) / float64(srcW)
	fy := float64(outH) / float64(srcH)
	f := fx
	if fy > fx {
		f = fy
	}
	return int(math.Round(float64(srcW) * f)), int(math.Round(float64(srcH) * f))
}

// writeFaceTrackSendcmd builds numeric sendcmd updates for crop x/y (segment-local time).
func writeFaceTrackSendcmd(dir string, index int, doc *cropPathDoc, srcStartMs, srcEndMs int64, srcW, srcH, outW, outH int) (string, error) {
	if doc == nil || len(doc.Samples) == 0 {
		return "", fmt.Errorf("empty crop path")
	}
	if outW <= 0 || outH <= 0 {
		outW, outH = 1080, 1920
	}
	sw, sh := scaledFillSize(srcW, srcH, outW, outH)

	var b strings.Builder
	wrote := 0
	emit := func(t float64, cx, cy float64) {
		cx = clamp01(cx)
		cy = clamp01(cy)
		x := int(math.Round(cx*float64(sw) - float64(outW)/2))
		y := int(math.Round(cy*float64(sh) - float64(outH)/2))
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		if x > sw-outW {
			x = sw - outW
		}
		if y > sh-outH {
			y = sh - outH
		}
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		fmt.Fprintf(&b, "%.3f crop x %d;\n", t, x)
		fmt.Fprintf(&b, "%.3f crop y %d;\n", t, y)
		wrote++
	}

	for _, s := range doc.Samples {
		if s.TMs < srcStartMs {
			continue
		}
		if srcEndMs > srcStartMs && s.TMs > srcEndMs {
			break
		}
		t := float64(s.TMs-srcStartMs) / 1000.0
		if t < 0 {
			t = 0
		}
		emit(t, s.CX, s.CY)
	}
	if wrote == 0 {
		emit(0, 0.5, 0.42)
	}

	path := filepath.Join(dir, fmt.Sprintf("face_track_%d.txt", index))
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0.5
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func faceTrackGeometry(cmdPath string, w, h int) string {
	if w <= 0 || h <= 0 {
		w, h = 1080, 1920
	}
	escaped := escapeFilterPath(cmdPath)
	return fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=increase,sendcmd=f=%s,crop=%d:%d:x:y,setsar=1",
		w, h, escaped, w, h,
	)
}
