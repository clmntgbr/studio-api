package subtitles

import (
	"encoding/json"

	domainstyle "go-api/internal/domain/subtitlestyle"
)

type styleSettings struct {
	MaxChars      *int    `json:"maxChars"`
	PauseBreakMs  *int64  `json:"pauseBreakMs"`
	LastWordPadMs *int64  `json:"lastWordPadMs"`
	Animation     *string `json:"animation"`
	MarginV       *int    `json:"marginV"`
	Shadow        *int    `json:"shadow"`
}

// FromDomain maps a persisted subtitle style onto the generator Style.
func FromDomain(s *domainstyle.Style) Style {
	out := DefaultStyle()
	if s == nil {
		return out
	}
	if s.FontFamily != nil && *s.FontFamily != "" {
		out.FontFamily = *s.FontFamily
	}
	if s.FontSize != nil && *s.FontSize > 0 {
		out.FontSize = *s.FontSize
	}
	if s.BaseColor != nil && *s.BaseColor != "" {
		out.BaseColor = *s.BaseColor
	}
	if s.HighlightColor != nil && *s.HighlightColor != "" {
		out.HighlightColor = *s.HighlightColor
	}
	if s.OutlineColor != nil && *s.OutlineColor != "" {
		out.OutlineColor = *s.OutlineColor
	}
	if s.OutlineSize != nil {
		out.OutlineSize = *s.OutlineSize
	}
	if s.WordsPerLine != nil {
		out.WordsPerLine = *s.WordsPerLine
	}
	if s.Position != nil && *s.Position != "" {
		out.Position = *s.Position
	}
	out.Uppercase = s.Uppercase

	var settings styleSettings
	if len(s.Settings) > 0 {
		_ = json.Unmarshal(s.Settings, &settings)
	}
	if settings.MaxChars != nil && *settings.MaxChars > 0 {
		out.MaxChars = *settings.MaxChars
	}
	if settings.PauseBreakMs != nil && *settings.PauseBreakMs > 0 {
		out.PauseBreakMs = *settings.PauseBreakMs
	}
	if settings.LastWordPadMs != nil && *settings.LastWordPadMs > 0 {
		out.LastWordPadMs = *settings.LastWordPadMs
	}
	if settings.Animation != nil && *settings.Animation != "" {
		out.Animation = *settings.Animation
	}
	if settings.MarginV != nil && *settings.MarginV > 0 {
		out.MarginV = *settings.MarginV
	}
	if settings.Shadow != nil && *settings.Shadow >= 0 {
		out.Shadow = *settings.Shadow
	}
	return out
}
