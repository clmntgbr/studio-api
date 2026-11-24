package export

import (
	"fmt"
	"strings"
)

// ToEDL builds a CMX 3600 EDL (non-drop frame).
func ToEDL(in TimelineInput) string {
	in = in.normalized()
	var b strings.Builder
	fmt.Fprintf(&b, "TITLE: %s\n", sanitizeEDLTitle(in.ProjectName))
	b.WriteString("FCM: NON-DROP FRAME\n\n")
	for i, seg := range in.Segments {
		if seg.SourceEndMs <= seg.SourceStartMs || seg.OutputEndMs <= seg.OutputStartMs {
			continue
		}
		srcIn := FormatSMPTE(seg.SourceStartMs, in.FPS)
		srcOut := FormatSMPTE(seg.SourceEndMs, in.FPS)
		recIn := FormatSMPTE(seg.OutputStartMs, in.FPS)
		recOut := FormatSMPTE(seg.OutputEndMs, in.FPS)
		reel := edlReel(seg, i+1)
		fmt.Fprintf(&b, "%03d  %-8s V     C        %s %s %s %s\n",
			i+1, reel, srcIn, srcOut, recIn, recOut)
		name := seg.MediaName
		if name == "" {
			name = seg.MediaFileID.String()
		}
		fmt.Fprintf(&b, "* FROM CLIP NAME: %s\n\n", name)
	}
	return b.String()
}

func edlReel(seg Segment, n int) string {
	if seg.MediaName != "" {
		r := strings.ToUpper(seg.MediaName)
		r = strings.Map(func(ch rune) rune {
			if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
				return ch
			}
			return -1
		}, r)
		if len(r) > 8 {
			r = r[:8]
		}
		if r != "" {
			return r
		}
	}
	return fmt.Sprintf("AX%06d", n)
}

func sanitizeEDLTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if s == "" {
		return "UNTITLED"
	}
	return s
}
