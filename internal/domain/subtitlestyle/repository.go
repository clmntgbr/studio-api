package subtitlestyle

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	Save(ctx context.Context, s *Style) error
	Update(ctx context.Context, s *Style) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Style, error)
}

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Style, error)
	List(ctx context.Context) ([]Style, error)
}