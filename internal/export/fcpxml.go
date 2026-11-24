package export

import (
	"fmt"
	"html"
	"strings"

	"github.com/google/uuid"
)

// ToFCPXML builds a minimal FCPXML 1.9 document importable by Final Cut / Resolve.
func ToFCPXML(in TimelineInput) string {
	in = in.normalized()
	frameDur := FormatFCPFrameDuration(in.FPS)
	seqDur := FormatFCPDuration(in.DurationMs, in.FPS)

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString("<!DOCTYPE fcpxml>\n")
	b.WriteString(`<fcpxml version="1.9">` + "\n")
	b.WriteString("  <resources>\n")
	fmt.Fprintf(&b,
		`    <format id="r1" name="FFVideoFormat%dx%dp%d" frameDuration="%s" width="%d" height="%d"/>`+"\n",
		in.Width, in.Height, int(in.FPS+0.5), frameDur, in.Width, in.Height,
	)

	assetIDs := make(map[string]string)
	nextAsset := 2
	for _, seg := range in.Segments {
		key := seg.MediaFileID.String()
		if _, ok := assetIDs[key]; ok {
			continue
		}
		id := fmt.Sprintf("r%d", nextAsset)
		nextAsset++
		assetIDs[key] = id
		name := seg.MediaName
		if name == "" {
			name = key
		}
		src := seg.MediaPath
		if src == "" {
			src = "file://localhost/media/" + key + "/source/original.mp4"
		}
		// Asset duration: use the farthest source end among segments for this media.
		assetDur := assetDurationMs(in.Segments, seg.MediaFileID)
		fmt.Fprintf(&b,
			`    <asset id="%s" name="%s" src="%s" start="0s" duration="%s" hasVideo="1" hasAudio="1" format="r1"/>`+"\n",
			id, html.EscapeString(name), html.EscapeString(src), FormatFCPDuration(assetDur, in.FPS),
		)
	}
	b.WriteString("  </resources>\n")
	b.WriteString("  <library>\n")
	b.WriteString(`    <event name="Studio">` + "\n")
	fmt.Fprintf(&b, `      <project name="%s">`+"\n", html.EscapeString(in.ProjectName))
	fmt.Fprintf(&b, `        <sequence format="r1" duration="%s" tcStart="0s" tcFormat="NDF">`+"\n", seqDur)
	b.WriteString("          <spine>\n")

	for _, seg := range in.Segments {
		if seg.SourceEndMs <= seg.SourceStartMs || seg.OutputEndMs <= seg.OutputStartMs {
			continue
		}
		ref := assetIDs[seg.MediaFileID.String()]
		offset := FormatFCPDuration(seg.OutputStartMs, in.FPS)
		start := FormatFCPDuration(seg.SourceStartMs, in.FPS)
		dur := FormatFCPDuration(seg.SourceEndMs-seg.SourceStartMs, in.FPS)
		fmt.Fprintf(&b,
			`            <asset-clip ref="%s" offset="%s" start="%s" duration="%s" tcFormat="NDF"/>`+"\n",
			ref, offset, start, dur,
		)
	}

	b.WriteString("          </spine>\n")
	b.WriteString("        </sequence>\n")
	b.WriteString("      </project>\n")
	b.WriteString("    </event>\n")
	b.WriteString("  </library>\n")
	b.WriteString("</fcpxml>\n")
	return b.String()
}

func assetDurationMs(segs []Segment, mediaID uuid.UUID) int64 {
	var max int64
	for _, s := range segs {
		if s.MediaFileID != mediaID {
			continue
		}
		if s.SourceEndMs > max {
			max = s.SourceEndMs
		}
	}
	if max <= 0 {
		max = 1000
	}
	return max
}
