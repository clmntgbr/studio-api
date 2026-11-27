package subtitlestyle

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	domainstyle "go-api/internal/domain/subtitlestyle"
)

type CreateCommand struct {
	Name           string
	FontFamily     *string
	FontSize       *int
	BaseColor      *string
	HighlightColor *string
	OutlineColor   *string
	OutlineSize    *int
	WordsPerLine   *int
	Position       *string
	Uppercase      *bool
	Settings       json.RawMessage
}

type CreateHandler struct {
	repo domainstyle.WriteRepository
}

func NewCreateHandler(repo domainstyle.WriteRepository) *CreateHandler {
	return &CreateHandler{repo: repo}
}

func (h *CreateHandler) Handle(ctx context.Context, cmd CreateCommand) (*domainstyle.Style, error) {
	if cmd.Name == "" {
		return nil, errors.New("name is required")
	}
	if cmd.FontFamily != nil && strings.ContainsAny(*cmd.FontFamily, ",\r\n{}") {
		return nil, errors.New("fontFamily is invalid")
	}
	if cmd.WordsPerLine != nil && (*cmd.WordsPerLine < 2 || *cmd.WordsPerLine > 4) {
		return nil, errors.New("wordsPerLine must be between 2 and 4")
	}
	if cmd.Position != nil {
		switch *cmd.Position {
		case "top", "middle", "bottom":
		default:
			return nil, errors.New("position must be top, middle or bottom")
		}
	}

	s := domainstyle.New(cmd.Name)
	s.FontFamily = cmd.FontFamily
	s.FontSize = cmd.FontSize
	s.BaseColor = cmd.BaseColor
	s.HighlightColor = cmd.HighlightColor
	s.OutlineColor = cmd.OutlineColor
	s.OutlineSize = cmd.OutlineSize
	s.WordsPerLine = cmd.WordsPerLine
	s.Position = cmd.Position
	if cmd.Uppercase != nil {
		s.Uppercase = *cmd.Uppercase
	}
	if len(cmd.Settings) > 0 {
		s.Settings = cmd.Settings
	}
	if err := h.repo.Save(ctx, s); err != nil {
		return nil, errors.New("failed to create subtitle style")
	}
	return s, nil
}