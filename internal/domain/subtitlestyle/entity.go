package subtitlestyle

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Style struct {
	ID             uuid.UUID
	ProjectID      *uuid.UUID
	Name           string
	FontFamily     *string
	FontSize       *int
	BaseColor      *string
	HighlightColor *string
	OutlineColor   *string
	OutlineSize    *int
	WordsPerLine   *int16
	Position       *string
	Uppercase      bool
	Settings       json.RawMessage
	CreatedAt      time.Time
}

func New(name string, projectID *uuid.UUID) *Style {
	return &Style{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
		Uppercase: true,
		Settings:  json.RawMessage(`{}`),
		CreatedAt: time.Now().UTC(),
	}
}
