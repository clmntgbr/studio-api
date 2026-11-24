package clip

import (
	"time"

	"github.com/google/uuid"
)

type Clip struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	MediaFileID   uuid.UUID
	SourceStartMs int64
	SourceEndMs   int64
	Origin        string
	Score         *float64
	Hook          *string
	Reason        *string
	Title         *string
	Description   *string
	Hashtags      []string
	Selected      bool
	CreatedAt     time.Time
}

func New(projectID, mediaFileID uuid.UUID, startMs, endMs int64, origin string) *Clip {
	if origin == "" {
		origin = "llm"
	}
	return &Clip{
		ID:            uuid.New(),
		ProjectID:     projectID,
		MediaFileID:   mediaFileID,
		SourceStartMs: startMs,
		SourceEndMs:   endMs,
		Origin:        origin,
		CreatedAt:     time.Now().UTC(),
	}
}
