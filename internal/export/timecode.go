package export

import "fmt"

// FormatSMPTE formats milliseconds as HH:MM:SS:FF at the given fps (non-drop).
func FormatSMPTE(ms int64, fps float64) string {
	if ms < 0 {
		ms = 0
	}
	if fps <= 0 {
		fps = 30
	}
	totalFrames := int64(float64(ms) * fps / 1000.0)
	fpsInt := int64(fps + 0.5)
	if fpsInt <= 0 {
		fpsInt = 30
	}
	ff := totalFrames % fpsInt
	totalSeconds := totalFrames / fpsInt
	s := totalSeconds % 60
	totalMinutes := totalSeconds / 60
	m := totalMinutes % 60
	h := totalMinutes / 60
	return fmt.Sprintf("%02d:%02d:%02d:%02d", h, m, s, ff)
}

// FormatFCPDuration formats milliseconds as a rational seconds string for FCPXML (e.g. "3000/30000s").
func FormatFCPDuration(ms int64, fps float64) string {
	if ms < 0 {
		ms = 0
	}
	if fps <= 0 {
		fps = 30
	}
	// Use 1/1000s ticks for stable ms mapping.
	return fmt.Sprintf("%d/1000s", ms)
}

// FormatFCPFrameDuration returns the frame duration resource string.
func FormatFCPFrameDuration(fps float64) string {
	if fps <= 0 {
		fps = 30
	}
	// Prefer integer fps when close.
	if almostInt(fps) {
		n := int(fps + 0.5)
		return fmt.Sprintf("100/%d00s", n)
	}
	return fmt.Sprintf("1001/%ds", int(fps*1001+0.5))
}

func almostInt(v float64) bool {
	n := float64(int(v + 0.5))
	if n-v < 0 {
		return v-n < 0.001
	}
	return n-v < 0.001
}
