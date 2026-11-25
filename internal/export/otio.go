package export

import (
	"encoding/json"
	"fmt"
)

type otioRationalTime struct {
	OTIOSchema string  `json:"OTIO_SCHEMA"`
	Value      float64 `json:"value"`
	Rate       float64 `json:"rate"`
}

type otioTimeRange struct {
	OTIOSchema string           `json:"OTIO_SCHEMA"`
	StartTime  otioRationalTime `json:"start_time"`
	Duration   otioRationalTime `json:"duration"`
}

type otioMediaRef struct {
	OTIOSchema     string         `json:"OTIO_SCHEMA"`
	Name           string         `json:"name"`
	TargetURL      string         `json:"target_url"`
	AvailableRange *otioTimeRange `json:"available_range,omitempty"`
}

type otioClip struct {
	OTIOSchema  string         `json:"OTIO_SCHEMA"`
	Name        string         `json:"name"`
	SourceRange otioTimeRange  `json:"source_range"`
	MediaRef    otioMediaRef   `json:"media_reference"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type otioTrack struct {
	OTIOSchema string     `json:"OTIO_SCHEMA"`
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Children   []otioClip `json:"children"`
}

type otioStack struct {
	OTIOSchema string      `json:"OTIO_SCHEMA"`
	Name       string      `json:"name"`
	Children   []otioTrack `json:"children"`
}

type otioTimeline struct {
	OTIOSchema string         `json:"OTIO_SCHEMA"`
	Name       string         `json:"name"`
	Tracks     otioStack      `json:"tracks"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ToOTIO builds an OpenTimelineIO JSON timeline (schema 1.x compatible).
func ToOTIO(in TimelineInput) (string, error) {
	in = in.normalized()
	clips := make([]otioClip, 0, len(in.Segments))
	for i, seg := range in.Segments {
		if seg.SourceEndMs <= seg.SourceStartMs {
			continue
		}
		name := seg.MediaName
		if name == "" {
			name = fmt.Sprintf("clip-%d", i+1)
		}
		src := seg.MediaPath
		if src == "" {
			src = "file://localhost/media/" + seg.MediaFileID.String() + "/source/original.mp4"
		}
		rate := in.FPS
		srcStartFrames := msToFrames(seg.SourceStartMs, rate)
		srcDurFrames := msToFrames(seg.SourceEndMs-seg.SourceStartMs, rate)
		clips = append(clips, otioClip{
			OTIOSchema: "Clip.1",
			Name:       name,
			SourceRange: otioTimeRange{
				OTIOSchema: "TimeRange.1",
				StartTime:  otioRationalTime{OTIOSchema: "RationalTime.1", Value: srcStartFrames, Rate: rate},
				Duration:   otioRationalTime{OTIOSchema: "RationalTime.1", Value: srcDurFrames, Rate: rate},
			},
			MediaRef: otioMediaRef{
				OTIOSchema: "ExternalReference.1",
				Name:       name,
				TargetURL:  src,
			},
			Metadata: map[string]any{
				"studio": map[string]any{
					"outputStartMs": seg.OutputStartMs,
					"outputEndMs":   seg.OutputEndMs,
					"mediaFileId":   seg.MediaFileID.String(),
				},
			},
		})
	}
	doc := otioTimeline{
		OTIOSchema: "Timeline.1",
		Name:       in.ProjectName,
		Tracks: otioStack{
			OTIOSchema: "Stack.1",
			Name:       "tracks",
			Children: []otioTrack{{
				OTIOSchema: "Track.1",
				Name:       "V1",
				Kind:       "Video",
				Children:   clips,
			}},
		},
		Metadata: map[string]any{
			"studio": map[string]any{
				"version":    in.Version,
				"durationMs": in.DurationMs,
			},
		},
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw) + "\n", nil
}

func msToFrames(ms int64, fps float64) float64 {
	if ms < 0 {
		ms = 0
	}
	return float64(ms) * fps / 1000.0
}
