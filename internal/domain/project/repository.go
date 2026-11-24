package project

import (
	"context"
	"time"

	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type WriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, project *Project) error
	Update(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*View, error)
	List(ctx context.Context, query paginate.PaginateQuery) ([]View, int64, error)
}

type View struct {
	ID        uuid.UUID
	Name      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}
