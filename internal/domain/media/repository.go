package media

import (
	"context"
	"time"

	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type WriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, media *MediaFile) error
	Update(ctx context.Context, media *MediaFile) error
	// UpdateFromStatuses persists media only when the current DB status is in from.
	// Returns false when no row matched (already transitioned or missing).
	UpdateFromStatuses(ctx context.Context, media *MediaFile, from []Status) (bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*MediaFile, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SaveAsset(ctx context.Context, asset *Asset) error
	SaveConfiguration(ctx context.Context, cfg *Configuration) error
	SaveStep(ctx context.Context, step *ProcessingStep) error
	UpdateStep(ctx context.Context, step *ProcessingStep) error
	GetStep(ctx context.Context, mediaFileID uuid.UUID, kind StepKind) (*ProcessingStep, error)
}

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*FileView, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, query paginate.PaginateQuery) ([]FileView, int64, error)
	ListSteps(ctx context.Context, mediaFileID uuid.UUID) ([]StepView, error)
}

type FileView struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	Position         int
	SourceKind       SourceKind
	SourceURL        *string
	Title            *string
	OriginalFilename *string
	StorageKey       *string
	MimeType         *string
	DurationMs       *int64
	Width            *int
	Height           *int
	FPS              *float64
	VideoCodec       *string
	AudioCodec       *string
	HasAudio         *bool
	SizeBytes        *int64
	Status           Status
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Steps            []StepView
}

type StepView struct {
	Kind         StepKind
	Status       StepStatus
	Progress     float64
	ErrorMessage *string
	QueuedAt     *time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}
