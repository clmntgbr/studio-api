package presenter

import (
	"encoding/json"
	"time"

	domainclip "go-api/internal/domain/clip"
	domainllm "go-api/internal/domain/llmartifact"
)

type ClipResponse struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"projectId"`
	MediaFileID   string   `json:"mediaFileId"`
	SourceStartMs int64    `json:"sourceStartMs"`
	SourceEndMs   int64    `json:"sourceEndMs"`
	Origin        string   `json:"origin"`
	Score         *float64 `json:"score,omitempty"`
	Hook          *string  `json:"hook,omitempty"`
	Reason        *string  `json:"reason,omitempty"`
	Title         *string  `json:"title,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Hashtags      []string `json:"hashtags"`
	Selected      bool     `json:"selected"`
	CreatedAt     time.Time `json:"createdAt"`
}

func NewClipResponse(c *domainclip.Clip) ClipResponse {
	tags := c.Hashtags
	if tags == nil {
		tags = []string{}
	}
	return ClipResponse{
		ID:            c.ID.String(),
		ProjectID:     c.ProjectID.String(),
		MediaFileID:   c.MediaFileID.String(),
		SourceStartMs: c.SourceStartMs,
		SourceEndMs:   c.SourceEndMs,
		Origin:        c.Origin,
		Score:         c.Score,
		Hook:          c.Hook,
		Reason:        c.Reason,
		Title:         c.Title,
		Description:   c.Description,
		Hashtags:      tags,
		Selected:      c.Selected,
		CreatedAt:     c.CreatedAt,
	}
}

func NewClipListResponse(items []domainclip.Clip) []ClipResponse {
	out := make([]ClipResponse, 0, len(items))
	for i := range items {
		out = append(out, NewClipResponse(&items[i]))
	}
	return out
}

type LLMArtifactResponse struct {
	ID          string          `json:"id"`
	MediaFileID string          `json:"mediaFileId"`
	Kind        string          `json:"kind"`
	Payload     json.RawMessage `json:"payload"`
	Provider    *string         `json:"provider,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
}

func NewLLMArtifactListResponse(items []domainllm.Artifact) []LLMArtifactResponse {
	out := make([]LLMArtifactResponse, 0, len(items))
	for _, a := range items {
		payload := a.Payload
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}
		out = append(out, LLMArtifactResponse{
			ID:          a.ID.String(),
			MediaFileID: a.MediaFileID.String(),
			Kind:        a.Kind,
			Payload:     payload,
			Provider:    a.Provider,
			CreatedAt:   a.CreatedAt,
		})
	}
	return out
}
