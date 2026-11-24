package presenter

import (
	"encoding/json"
	"time"

	domaindetection "go-api/internal/domain/detection"
)

type DetectionResponse struct {
	ID             string          `json:"id"`
	MediaFileID    string          `json:"mediaFileId"`
	Kind           string          `json:"kind"`
	SourceStartMs  int64           `json:"sourceStartMs"`
	SourceEndMs    int64           `json:"sourceEndMs"`
	Confidence     *float64        `json:"confidence,omitempty"`
	Detector       string          `json:"detector"`
	FirstWordIndex *int            `json:"firstWordIndex,omitempty"`
	LastWordIndex  *int            `json:"lastWordIndex,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	CreatedAt      time.Time       `json:"createdAt"`
}

func NewDetectionListResponse(items []domaindetection.Detection) []DetectionResponse {
	out := make([]DetectionResponse, 0, len(items))
	for _, d := range items {
		payload := d.Payload
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}
		out = append(out, DetectionResponse{
			ID:             d.ID.String(),
			MediaFileID:    d.MediaFileID.String(),
			Kind:           string(d.Kind),
			SourceStartMs:  d.SourceStartMs,
			SourceEndMs:    d.SourceEndMs,
			Confidence:     d.Confidence,
			Detector:       d.Detector,
			FirstWordIndex: d.FirstWordIndex,
			LastWordIndex:  d.LastWordIndex,
			Payload:        payload,
			CreatedAt:      d.CreatedAt,
		})
	}
	return out
}

type OverrideResponse struct {
	ID            string    `json:"id"`
	MediaFileID   string    `json:"mediaFileId"`
	Kind          *string   `json:"kind,omitempty"`
	Action        string    `json:"action"`
	SourceStartMs int64     `json:"sourceStartMs"`
	SourceEndMs   int64     `json:"sourceEndMs"`
	CreatedAt     time.Time `json:"createdAt"`
}

func NewOverrideResponse(o *domaindetection.Override) OverrideResponse {
	var kind *string
	if o.Kind != nil {
		k := string(*o.Kind)
		kind = &k
	}
	return OverrideResponse{
		ID:            o.ID.String(),
		MediaFileID:   o.MediaFileID.String(),
		Kind:          kind,
		Action:        string(o.Action),
		SourceStartMs: o.SourceStartMs,
		SourceEndMs:   o.SourceEndMs,
		CreatedAt:     o.CreatedAt,
	}
}
