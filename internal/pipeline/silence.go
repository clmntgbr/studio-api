package pipeline

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type SilenceInterval struct {
	StartMs int64 `json:"startMs"`
	EndMs   int64 `json:"endMs"`
}

type SilenceResult struct {
	NoiseFloorDb          float64           `json:"noiseFloorDb"`
	CalculatedThresholdDb float64           `json:"calculatedThresholdDb"`
	Intervals             []SilenceInterval `json:"intervals"`
}

var (
	reMeanVolume   = regexp.MustCompile(`mean_volume:\s*(-?[0-9.]+)\s*dB`)
	reSilenceStart = regexp.MustCompile(`silence_start:\s*(-?[0-9.]+)`)
	reSilenceEnd   = regexp.MustCompile(`silence_end:\s*(-?[0-9.]+)`)
)

// DetectSilence runs volumedetect then silencedetect on analysis audio.
// thresholdMode: "auto" | "manual". For auto, offsetDb is added to measured noise floor.
func DetectSilence(ctx context.Context, audioPath string, thresholdMode string, manualThresholdDb *float64, offsetDb float64) (*SilenceResult, error) {
	noiseFloor, err := measureNoiseFloor(ctx, audioPath)
	if err != nil {
		return nil, err
	}

	threshold := noiseFloor + offsetDb
	if thresholdMode == "manual" && manualThresholdDb != nil {
		threshold = *manualThresholdDb
	}

	intervals, err := detectSilenceIntervals(ctx, audioPath, threshold)
	if err != nil {
		return nil, err
	}

	return &SilenceResult{
		NoiseFloorDb:          noiseFloor,
		CalculatedThresholdDb: threshold,
		Intervals:             intervals,
	}, nil
}

func measureNoiseFloor(ctx context.Context, audioPath string) (float64, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-i", audioPath, "-af", "volumedetect", "-f", "null", "-")
	out, _ := cmd.CombinedOutput()
	m := reMeanVolume.FindSubmatch(out)
	if m == nil {
		return -40, nil // sensible default when probe fails to parse
	}
	v, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return -40, nil
	}
	return v, nil
}

func detectSilenceIntervals(ctx context.Context, audioPath string, thresholdDb float64) ([]SilenceInterval, error) {
	// Analysis floor 50ms per docs; product min duration applied later in timeline.
	filter := fmt.Sprintf("silencedetect=noise=%.1fdB:d=0.05", thresholdDb)
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-i", audioPath, "-af", filter, "-f", "null", "-")
	out, _ := cmd.CombinedOutput()
	lines := strings.Split(string(out), "\n")

	var intervals []SilenceInterval
	var startSec *float64
	for _, line := range lines {
		if m := reSilenceStart.FindStringSubmatch(line); m != nil {
			v, err := strconv.ParseFloat(m[1], 64)
			if err == nil {
				startSec = &v
			}
			continue
		}
		if m := reSilenceEnd.FindStringSubmatch(line); m != nil && startSec != nil {
			end, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			startMs := int64(math.Round(*startSec * 1000))
			endMs := int64(math.Round(end * 1000))
			if endMs > startMs {
				intervals = append(intervals, SilenceInterval{StartMs: startMs, EndMs: endMs})
			}
			startSec = nil
		}
	}
	return intervals, nil
}
