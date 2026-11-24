package detection

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, d *Detection) error
	SaveMany(ctx context.Context, items []*Detection) error
	DeleteByMediaAndKind(ctx context.Context, mediaFileID uuid.UUID, kind Kind) error
	SaveOverride(ctx context.Context, o *Override) error
	DeleteOverridesByMedia(ctx context.Context, mediaFileID uuid.UUID) error
}

type ReadRepository interface {
	ListByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]Detection, error)
	ListOverridesByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]Override, error)
}
