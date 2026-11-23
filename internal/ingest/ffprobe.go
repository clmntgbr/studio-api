package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

type probeJSON struct {
	Streams []probeStream `json:"streams"`
	Format  probeFormat   `json:"format"`
}

type probeStream struct {
	CodecType          string `json:"codec_type"`
	CodecName          string `json:"codec_name"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	AvgFrameRate       string `json:"avg_frame_rate"`
	RFrameRate         string `json:"r_frame_rate"`
	Channels           int    `json:"channels"`
	Tags               map[string]string `json:"tags"`
	SideDataList       []map[string]any  `json:"side_data_list"`
}

type probeFormat struct {
	Duration string `json:"duration"`
}

func ProbeFile(ctx context.Context, path string) (Probe, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_streams",
		"-show_format",
		"-print_format", "json",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return Probe{}, fmt.Errorf("ffprobe: %w", err)
	}

	var raw probeJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		return Probe{}, fmt.Errorf("ffprobe json: %w", err)
	}

	var p Probe
	if dur, err := strconv.ParseFloat(raw.Format.Duration, 64); err == nil {
		p.DurationMs = int64(math.Round(dur * 1000))
	}

	var avgRate, rRate string
	for _, s := range raw.Streams {
		switch s.CodecType {
		case "video":
			if p.VideoCodec != "" {
				continue
			}
			p.VideoCodec = s.CodecName
			p.Width = s.Width
			p.Height = s.Height
			avgRate = s.AvgFrameRate
			rRate = s.RFrameRate
			p.Rotation = rotationFromStream(s)
		case "audio":
			if p.AudioCodec != "" {
				continue
			}
			p.AudioCodec = s.CodecName
			p.AudioChannels = s.Channels
		}
	}

	num, den := parseRate(avgRate)
	if den == 0 {
		num, den = parseRate(rRate)
	}
	p.FPSNum, p.FPSDen = num, den
	p.VFR = isVFR(avgRate, rRate)

	return p, nil
}

func parseRate(rate string) (int, int) {
	parts := strings.Split(rate, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	n, err1 := strconv.Atoi(parts[0])
	d, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || d == 0 {
		return 0, 0
	}
	return n, d
}

func isVFR(avg, r string) bool {
	if avg == "" || r == "" || avg == "0/0" || r == "0/0" {
		return false
	}
	an, ad := parseRate(avg)
	rn, rd := parseRate(r)
	if ad == 0 || rd == 0 {
		return false
	}
	// Compare as floats with small tolerance.
	a := float64(an) / float64(ad)
	b := float64(rn) / float64(rd)
	return math.Abs(a-b) > 0.05
}

func rotationFromStream(s probeStream) int {
	if s.Tags != nil {
		if v, ok := s.Tags["rotate"]; ok {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
	}
	for _, side := range s.SideDataList {
		if v, ok := side["rotation"]; ok {
			switch t := v.(type) {
			case float64:
				return int(t)
			case json.Number:
				n, _ := t.Int64()
				return int(n)
			}
		}
	}
	return 0
}

func needsMezzanine(p Probe) bool {
	if p.VFR {
		return true
	}
	switch strings.ToLower(p.VideoCodec) {
	case "h264", "hevc", "av1", "vp9", "vp8":
		return false
	default:
		return p.VideoCodec != ""
	}
}
