package overlay

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Overlay struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	ClipID     *uuid.UUID
	Kind       string
	Text       *string
	StorageKey *string
	StartMs    int64
	EndMs      int64
	Position   *string
	Settings   json.RawMessage
}

func New(projectID uuid.UUID, kind string, startMs, endMs int64) *Overlay {
	return &Overlay{
		ID:        uuid.New(),
		ProjectID: projectID,
		Kind:      kind,
		StartMs:   startMs,
		EndMs:     endMs,
		Settings:  json.RawMessage(`{}`),
	}
}
