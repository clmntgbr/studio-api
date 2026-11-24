package presenter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainclip "go-api/internal/domain/clip"
	domainllm "go-api/internal/domain/llmartifact"
	"go-api/internal/export"
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

type NumberedClipResponse struct {
	Index    int      `json:"index"`
	Filename string   `json:"filename"`
	ClipID   string   `json:"clipId"`
	Title    string   `json:"title,omitempty"`
	Hook     string   `json:"hook,omitempty"`
	Hashtags []string `json:"hashtags,omitempty"`
}

// NewNumberedClipsPack orders selected clips first, then others, as 001.mp4…
func NewNumberedClipsPack(items []domainclip.Clip) []NumberedClipResponse {
	ordered := make([]domainclip.Clip, 0, len(items))
	for i := range items {
		if items[i].Selected {
			ordered = append(ordered, items[i])
		}
	}
	if len(ordered) == 0 {
		ordered = items
	}
	raw := make([]export.NumberedClip, 0, len(ordered))
	for i := range ordered {
		c := &ordered[i]
		entry := export.NumberedClip{ClipID: c.ID}
		if c.Title != nil {
			entry.Title = *c.Title
		}
		if c.Hook != nil {
			entry.Hook = *c.Hook
		}
		raw = append(raw, entry)
	}
	pack := export.NumberedClipsPack(raw)
	out := make([]NumberedClipResponse, 0, len(pack))
	for i, p := range pack {
		tags := ordered[i].Hashtags
		out = append(out, NumberedClipResponse{
			Index:    p.Index,
			Filename: p.Filename,
			ClipID:   p.ClipID.String(),
			Title:    p.Title,
			Hook:     p.Hook,
			Hashtags: tags,
		})
	}
	return out
}

func NumberedClipsManifestText(pack []NumberedClipResponse) string {
	var b strings.Builder
	b.WriteString("# Studio numbered clips pack (CapCut)\n")
	for _, c := range pack {
		title := c.Title
		if title == "" {
			title = c.Hook
		}
		if title == "" {
			title = c.ClipID
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\n", c.Filename, c.ClipID, title)
	}
	return b.String()
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
