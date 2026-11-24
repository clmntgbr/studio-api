package outputpreset

import (
	"context"

	"github.com/google/uuid"
)

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Preset, error)
	List(ctx context.Context) ([]Preset, error)
}
