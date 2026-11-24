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
}

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Render, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]Render, error)
}
