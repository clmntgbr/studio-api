package transcript

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusSkipped   Status = "skipped"
	StatusCancelled Status = "cancelled"
)

type Transcript struct {
	ID            uuid.UUID
	MediaFileID   uuid.UUID
	Language      *string
	Text          *string
	Provider      string
	ProviderJobID *string
	Status        Status
	CreatedAt     time.Time
}

func New(mediaFileID uuid.UUID) *Transcript {
	return &Transcript{
		ID:          uuid.New(),
		MediaFileID: mediaFileID,
		Provider:    "assemblyai",
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC(),
	}
}

type Word struct {
	ID            int64
	TranscriptID  uuid.UUID
	WordIndex     int
	Text          string
	SourceStartMs int64
	SourceEndMs   int64
	Confidence    *float64
	Speaker       *string
}
