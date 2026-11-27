package presenter

import (
	"encoding/json"
	"time"

	domainstyle "go-api/internal/domain/subtitlestyle"
)

type SubtitleStyleResponse struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	FontFamily     *string         `json:"fontFamily,omitempty"`
	FontSize       *int            `json:"fontSize,omitempty"`
	BaseColor      *string         `json:"baseColor,omitempty"`
	HighlightColor *string         `json:"highlightColor,omitempty"`
	OutlineColor   *string         `json:"outlineColor,omitempty"`
	OutlineSize    *int            `json:"outlineSize,omitempty"`
	WordsPerLine   *int            `json:"wordsPerLine,omitempty"`
	Position       *string         `json:"position,omitempty"`
	Uppercase      bool            `json:"uppercase"`
	Settings       json.RawMessage `json:"settings"`
	CreatedAt      time.Time       `json:"createdAt"`
}

func NewSubtitleStyleResponse(s *domainstyle.Style) SubtitleStyleResponse {
	settings := s.Settings
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	return SubtitleStyleResponse{
		ID:             s.ID.String(),
		Name:           s.Name,
		FontFamily:     s.FontFamily,
		FontSize:       s.FontSize,
		BaseColor:      s.BaseColor,
		HighlightColor: s.HighlightColor,
		OutlineColor:   s.OutlineColor,
		OutlineSize:    s.OutlineSize,
		WordsPerLine:   s.WordsPerLine,
		Position:       s.Position,
		Uppercase:      s.Uppercase,
		Settings:       settings,
		CreatedAt:      s.CreatedAt,
	}
}

func NewSubtitleStyleListResponse(items []domainstyle.Style) []SubtitleStyleResponse {
	out := make([]SubtitleStyleResponse, 0, len(items))
	for i := range items {
		out = append(out, NewSubtitleStyleResponse(&items[i]))
	}
	return out
}