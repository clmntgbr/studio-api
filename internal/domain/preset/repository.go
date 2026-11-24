package preset

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	Save(ctx context.Context, p *OutputPreset) error
	Update(ctx context.Context, p *OutputPreset) error
}

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*OutputPreset, error)
	List(ctx context.Context) ([]OutputPreset, error)
}
