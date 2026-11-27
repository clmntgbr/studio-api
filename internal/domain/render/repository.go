package render

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, r *Render) error
	Update(ctx context.Context, r *Render) error
	GetByID(ctx context.Context, id uuid.UUID) (*Render, error)
	FindByFingerprint(ctx context.Context, mediaFileID uuid.UUID, fingerprint string) (*Render, error)
	// HasInFlightByMedia is true when any render is pending, queued, or running.
	HasInFlightByMedia(ctx context.Context, mediaFileID uuid.UUID) (bool, error)
}

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Render, error)
	FindByFingerprint(ctx context.Context, mediaFileID uuid.UUID, fingerprint string) (*Render, error)
	ListByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]Render, error)
}
