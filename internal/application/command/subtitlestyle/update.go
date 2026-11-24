package subtitlestyle

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	domainstyle "go-api/internal/domain/subtitlestyle"

	"github.com/google/uuid"
)

type UpdateCommand struct {
	ID             uuid.UUID
	Name           *string
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

type UpdateHandler struct {
	repo domainstyle.WriteRepository
}

func NewUpdateHandler(repo domainstyle.WriteRepository) *UpdateHandler {
	return &UpdateHandler{repo: repo}
}

func (h *UpdateHandler) Handle(ctx context.Context, cmd UpdateCommand) (*domainstyle.Style, error) {
	s, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errors.New("subtitle style not found")
	}
	if cmd.Name != nil {
		if *cmd.Name == "" {
			return nil, errors.New("name is required")
		}
		s.Name = *cmd.Name
	}
	if cmd.FontFamily != nil {
		if strings.ContainsAny(*cmd.FontFamily, ",\r\n{}") {
			return nil, errors.New("fontFamily is invalid")
		}
		s.FontFamily = cmd.FontFamily
	}
	if cmd.FontSize != nil {
		s.FontSize = cmd.FontSize
	}
	if cmd.BaseColor != nil {
		s.BaseColor = cmd.BaseColor
	}
	if cmd.HighlightColor != nil {
		s.HighlightColor = cmd.HighlightColor
	}
	if cmd.OutlineColor != nil {
		s.OutlineColor = cmd.OutlineColor
	}
	if cmd.OutlineSize != nil {
		s.OutlineSize = cmd.OutlineSize
	}
	if cmd.WordsPerLine != nil {
		if *cmd.WordsPerLine < 2 || *cmd.WordsPerLine > 4 {
			return nil, errors.New("wordsPerLine must be between 2 and 4")
		}
		s.WordsPerLine = cmd.WordsPerLine
	}
	if cmd.Position != nil {
		switch *cmd.Position {
		case "top", "middle", "bottom":
			s.Position = cmd.Position
		default:
			return nil, errors.New("position must be top, middle or bottom")
		}
	}
	if cmd.Uppercase != nil {
		s.Uppercase = *cmd.Uppercase
	}
	if len(cmd.Settings) > 0 {
		s.Settings = cmd.Settings
	}
	if err := h.repo.Update(ctx, s); err != nil {
		return nil, errors.New("failed to update subtitle style")
	}
	return s, nil
}
