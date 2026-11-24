package llmartifact

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Artifact struct {
	ID          uuid.UUID
	MediaFileID uuid.UUID
	Kind        string
	Payload     json.RawMessage
	Provider    *string
	CreatedAt   time.Time
}

func New(mediaFileID uuid.UUID, kind string, payload json.RawMessage, provider *string) *Artifact {
	return &Artifact{
		ID:          uuid.New(),
		MediaFileID: mediaFileID,
		Kind:        kind,
		Payload:     payload,
		Provider:    provider,
		CreatedAt:   time.Now().UTC(),
	}
}
