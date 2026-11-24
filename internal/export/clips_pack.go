package export

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// NumberedClip is one CapCut-style pack entry (001.mp4, 002.mp4, …).
type NumberedClip struct {
	Index    int       `json:"index"`
	Filename string    `json:"filename"`
	ClipID   uuid.UUID `json:"clipId"`
	Title    string    `json:"title,omitempty"`
	Hook     string    `json:"hook,omitempty"`
}

// NumberedClipsPack lists selected clips in order with CapCut-friendly names.
func NumberedClipsPack(clips []NumberedClip) []NumberedClip {
	out := make([]NumberedClip, 0, len(clips))
	for i, c := range clips {
		n := i + 1
		c.Index = n
		c.Filename = fmt.Sprintf("%03d.mp4", n)
		out = append(out, c)
	}
	return out
}

// NumberedClipsManifest is a plain-text CapCut import helper.
func NumberedClipsManifest(clips []NumberedClip) string {
	pack := NumberedClipsPack(clips)
	var b strings.Builder
	b.WriteString("# Studio numbered clips pack (CapCut)\n")
	for _, c := range pack {
		title := c.Title
		if title == "" {
			title = c.Hook
		}
		if title == "" {
			title = c.ClipID.String()
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\n", c.Filename, c.ClipID.String(), title)
	}
	return b.String()
}
