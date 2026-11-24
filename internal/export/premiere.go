package export

import (
	"fmt"
	"html"
	"path"
	"strings"
)

// ToPremiereXML builds a Premiere Pro xmeml 4 sequence from KEEP segments.
func ToPremiereXML(in TimelineInput) string {
	in = in.normalized()
	timebase := int(in.FPS + 0.5)
	if timebase <= 0 {
		timebase = 30
	}
	ntsc := "FALSE"
	if in.FPS > 0 && in.FPS != float64(int(in.FPS)) {
		ntsc = "TRUE"
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString("<!DOCTYPE xmeml>\n")
	b.WriteString(`<xmeml version="4">` + "\n")
	fmt.Fprintf(&b, "  <sequence id=\"sequence-1\">\n")
	fmt.Fprintf(&b, "    <name>%s</name>\n", html.EscapeString(in.ProjectName))
	fmt.Fprintf(&b, "    <duration>%d</duration>\n", premiereFrames(in.DurationMs, in.FPS))
	b.WriteString("    <rate>\n")
	fmt.Fprintf(&b, "      <timebase>%d</timebase>\n", timebase)
	fmt.Fprintf(&b, "      <ntsc>%s</ntsc>\n", ntsc)
	b.WriteString("    </rate>\n")
	b.WriteString("    <media>\n")
	b.WriteString("      <video>\n")
	b.WriteString("        <format>\n")
	b.WriteString("          <samplecharacteristics>\n")
	b.WriteString("            <rate>\n")
	fmt.Fprintf(&b, "              <timebase>%d</timebase>\n", timebase)
	fmt.Fprintf(&b, "              <ntsc>%s</ntsc>\n", ntsc)
	b.WriteString("            </rate>\n")
	fmt.Fprintf(&b, "            <width>%d</width>\n", in.Width)
	fmt.Fprintf(&b, "            <height>%d</height>\n", in.Height)
	b.WriteString("            <pixelaspectratio>square</pixelaspectratio>\n")
	b.WriteString("            <fielddominance>none</fielddominance>\n")
	b.WriteString("          </samplecharacteristics>\n")
	b.WriteString("        </format>\n")
	b.WriteString("        <track>\n")

	definedFiles := map[string]bool{}
	type clipRef struct {
		clipID  string
		fileID  string
		name    string
		src     string
		in      int64
		out     int64
		start   int64
		end     int64
		dur     int64
		mediaID string
	}
	refs := make([]clipRef, 0, len(in.Segments))

	for i, seg := range in.Segments {
		if seg.SourceEndMs <= seg.SourceStartMs || seg.OutputEndMs <= seg.OutputStartMs {
			continue
		}
		name := seg.MediaName
		if name == "" {
			name = seg.MediaFileID.String()
		}
		src := seg.MediaPath
		if src == "" {
			src = "file://localhost/media/" + seg.MediaFileID.String() + "/source/original.mp4"
		}
		inPoint := premiereFrames(seg.SourceStartMs, in.FPS)
		outPoint := premiereFrames(seg.SourceEndMs, in.FPS)
		start := premiereFrames(seg.OutputStartMs, in.FPS)
		end := premiereFrames(seg.OutputEndMs, in.FPS)
		dur := outPoint - inPoint
		if dur <= 0 {
			continue
		}
		fileID := fmt.Sprintf("file-%s", seg.MediaFileID.String())
		refs = append(refs, clipRef{
			clipID:  fmt.Sprintf("clipitem-%d", i+1),
			fileID:  fileID,
			name:    name,
			src:     src,
			in:      inPoint,
			out:     outPoint,
			start:   start,
			end:     end,
			dur:     dur,
			mediaID: seg.MediaFileID.String(),
		})
	}

	for _, ref := range refs {
		fmt.Fprintf(&b, "          <clipitem id=\"%s\">\n", ref.clipID)
		fmt.Fprintf(&b, "            <name>%s</name>\n", html.EscapeString(ref.name))
		fmt.Fprintf(&b, "            <duration>%d</duration>\n", ref.dur)
		b.WriteString("            <rate>\n")
		fmt.Fprintf(&b, "              <timebase>%d</timebase>\n", timebase)
		fmt.Fprintf(&b, "              <ntsc>%s</ntsc>\n", ntsc)
		b.WriteString("            </rate>\n")
		fmt.Fprintf(&b, "            <start>%d</start>\n", ref.start)
		fmt.Fprintf(&b, "            <end>%d</end>\n", ref.end)
		fmt.Fprintf(&b, "            <in>%d</in>\n", ref.in)
		fmt.Fprintf(&b, "            <out>%d</out>\n", ref.out)
		if !definedFiles[ref.fileID] {
			definedFiles[ref.fileID] = true
			fmt.Fprintf(&b, "            <file id=\"%s\">\n", ref.fileID)
			fmt.Fprintf(&b, "              <name>%s</name>\n", html.EscapeString(path.Base(ref.name)))
			fmt.Fprintf(&b, "              <pathurl>%s</pathurl>\n", html.EscapeString(ref.src))
			b.WriteString("              <rate>\n")
			fmt.Fprintf(&b, "                <timebase>%d</timebase>\n", timebase)
			fmt.Fprintf(&b, "                <ntsc>%s</ntsc>\n", ntsc)
			b.WriteString("              </rate>\n")
			fmt.Fprintf(&b, "              <duration>%d</duration>\n", premiereAssetFrames(in.Segments, ref.mediaID, in.FPS))
			b.WriteString("              <media>\n")
			b.WriteString("                <video>\n")
			b.WriteString("                  <samplecharacteristics>\n")
			fmt.Fprintf(&b, "                    <width>%d</width>\n", in.Width)
			fmt.Fprintf(&b, "                    <height>%d</height>\n", in.Height)
			b.WriteString("                  </samplecharacteristics>\n")
			b.WriteString("                </video>\n")
			b.WriteString("                <audio>\n")
			b.WriteString("                  <channelcount>2</channelcount>\n")
			b.WriteString("                </audio>\n")
			b.WriteString("              </media>\n")
			b.WriteString("            </file>\n")
		} else {
			fmt.Fprintf(&b, "            <file id=\"%s\"/>\n", ref.fileID)
		}
		fmt.Fprintf(&b, "            <link><linkclipref>%s</linkclipref><mediatype>video</mediatype><trackindex>1</trackindex><clipindex>1</clipindex></link>\n", ref.clipID)
		fmt.Fprintf(&b, "            <link><linkclipref>%s-audio</linkclipref><mediatype>audio</mediatype><trackindex>1</trackindex><clipindex>1</clipindex></link>\n", ref.clipID)
		b.WriteString("          </clipitem>\n")
	}

	b.WriteString("        </track>\n")
	b.WriteString("      </video>\n")
	b.WriteString("      <audio>\n")
	b.WriteString("        <track>\n")
	for _, ref := range refs {
		fmt.Fprintf(&b, "          <clipitem id=\"%s-audio\">\n", ref.clipID)
		fmt.Fprintf(&b, "            <name>%s</name>\n", html.EscapeString(ref.name))
		fmt.Fprintf(&b, "            <duration>%d</duration>\n", ref.dur)
		b.WriteString("            <rate>\n")
		fmt.Fprintf(&b, "              <timebase>%d</timebase>\n", timebase)
		fmt.Fprintf(&b, "              <ntsc>%s</ntsc>\n", ntsc)
		b.WriteString("            </rate>\n")
		fmt.Fprintf(&b, "            <start>%d</start>\n", ref.start)
		fmt.Fprintf(&b, "            <end>%d</end>\n", ref.end)
		fmt.Fprintf(&b, "            <in>%d</in>\n", ref.in)
		fmt.Fprintf(&b, "            <out>%d</out>\n", ref.out)
		fmt.Fprintf(&b, "            <file id=\"%s\"/>\n", ref.fileID)
		b.WriteString("          </clipitem>\n")
	}
	b.WriteString("        </track>\n")
	b.WriteString("      </audio>\n")
	b.WriteString("    </media>\n")
	b.WriteString("  </sequence>\n")
	b.WriteString("</xmeml>\n")
	return b.String()
}

func premiereFrames(ms int64, fps float64) int64 {
	if fps <= 0 {
		fps = 30
	}
	return int64(float64(ms) * fps / 1000.0)
}

func premiereAssetFrames(segs []Segment, mediaID string, fps float64) int64 {
	var maxEnd int64
	for _, s := range segs {
		if s.MediaFileID.String() != mediaID {
			continue
		}
		if s.SourceEndMs > maxEnd {
			maxEnd = s.SourceEndMs
		}
	}
	return premiereFrames(maxEnd, fps)
}
