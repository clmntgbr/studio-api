package presenter

import (
	"encoding/json"
	"time"

	domainpreset "go-api/internal/domain/outputpreset"
	domainrender "go-api/internal/domain/render"
)

type OutputPresetResponse struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Width        int             `json:"width"`
	Height       int             `json:"height"`
	AspectRatio  string          `json:"aspectRatio"`
	CropMode     string          `json:"cropMode"`
	VideoCodec   string          `json:"videoCodec"`
	AudioCodec   string          `json:"audioCodec"`
	FPS          *float64        `json:"fps,omitempty"`
	LoudnessLUFS *float64        `json:"loudnessLufs,omitempty"`
	Settings     json.RawMessage `json:"settings"`
}

func NewOutputPresetListResponse(items []domainpreset.Preset) []OutputPresetResponse {
	out := make([]OutputPresetResponse, 0, len(items))
	for _, p := range items {
		settings := p.Settings
		if len(settings) == 0 {
			settings = json.RawMessage(`{}`)
		}
		out = append(out, OutputPresetResponse{
			ID:           p.ID.String(),
			Name:         p.Name,
			Width:        p.Width,
			Height:       p.Height,
			AspectRatio:  p.AspectRatio,
			CropMode:     p.CropMode,
			VideoCodec:   p.VideoCodec,
			AudioCodec:   p.AudioCodec,
			FPS:          p.FPS,
			LoudnessLUFS: p.LoudnessLUFS,
			Settings:     settings,
		})
	}
	return out
}

type RenderResponse struct {
	ID              string          `json:"id"`
	ProjectID       string          `json:"projectId"`
	TimelineID      string          `json:"timelineId"`
	Kind            string          `json:"kind"`
	Strategy        string          `json:"strategy"`
	OutputPresetID  string          `json:"outputPresetId"`
	SubtitleStyleID *string         `json:"subtitleStyleId,omitempty"`
	ClipID          *string         `json:"clipId,omitempty"`
	Fingerprint     string          `json:"fingerprint"`
	Status          string          `json:"status"`
	Progress        float64         `json:"progress"`
	StorageKey      *string         `json:"storageKey,omitempty"`
	DurationMs      *int64          `json:"durationMs,omitempty"`
	SizeBytes       *int64          `json:"sizeBytes,omitempty"`
	ErrorMessage    *string         `json:"errorMessage,omitempty"`
	Plan            json.RawMessage `json:"plan"`
	CreatedAt       time.Time       `json:"createdAt"`
	StartedAt       *time.Time      `json:"startedAt,omitempty"`
	CompletedAt     *time.Time      `json:"completedAt,omitempty"`
}

func NewRenderResponse(r *domainrender.Render) RenderResponse {
	plan := r.Plan
	if len(plan) == 0 {
		plan = json.RawMessage(`{}`)
	}
	var styleID *string
	if r.SubtitleStyleID != nil {
		v := r.SubtitleStyleID.String()
		styleID = &v
	}
	var clipID *string
	if r.ClipID != nil {
		v := r.ClipID.String()
		clipID = &v
	}
	return RenderResponse{
		ID:              r.ID.String(),
		ProjectID:       r.ProjectID.String(),
		TimelineID:      r.TimelineID.String(),
		Kind:            string(r.Kind),
		Strategy:        string(r.Strategy),
		OutputPresetID:  r.OutputPresetID.String(),
		SubtitleStyleID: styleID,
		ClipID:          clipID,
		Fingerprint:     r.Fingerprint,
		Status:          string(r.Status),
		Progress:        r.Progress,
		StorageKey:      r.StorageKey,
		DurationMs:      r.DurationMs,
		SizeBytes:       r.SizeBytes,
		ErrorMessage:    r.ErrorMessage,
		Plan:            plan,
		CreatedAt:       r.CreatedAt,
		StartedAt:       r.StartedAt,
		CompletedAt:     r.CompletedAt,
	}
}

func NewRenderListResponse(items []domainrender.Render) []RenderResponse {
	out := make([]RenderResponse, 0, len(items))
	for i := range items {
		out = append(out, NewRenderResponse(&items[i]))
	}
	return out
}
