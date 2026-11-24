package export

import "github.com/google/uuid"

// Segment is a KEEP timeline slice used by text exporters.
type Segment struct {
	Index         int
	MediaFileID   uuid.UUID
	MediaName     string
	MediaPath     string // file:// or relative path for NLE import
	SourceStartMs int64
	SourceEndMs   int64
	OutputStartMs int64
	OutputEndMs   int64
	FPS           float64
}

// TimelineInput feeds every exporter.
type TimelineInput struct {
	ProjectName string
	Version     int
	DurationMs  int64
	Width       int
	Height      int
	FPS         float64
	Segments    []Segment
}

func (in TimelineInput) normalized() TimelineInput {
	out := in
	if out.FPS <= 0 {
		out.FPS = 30
	}
	if out.Width <= 0 {
		out.Width = 1920
	}
	if out.Height <= 0 {
		out.Height = 1080
	}
	if out.ProjectName == "" {
		out.ProjectName = "Studio"
	}
	return out
}
