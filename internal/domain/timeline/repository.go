package timeline

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, t *Timeline, segments []Segment, decisions []Decision) error
	Update(ctx context.Context, t *Timeline) error
	GetByID(ctx context.Context, id uuid.UUID) (*Timeline, error)
}

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Timeline, error)
	FindHeadByProject(ctx context.Context, projectID uuid.UUID) (*Timeline, error)
	ListSegments(ctx context.Context, timelineID uuid.UUID) ([]Segment, error)
	ListDecisions(ctx context.Context, timelineID uuid.UUID) ([]Decision, error)
}
