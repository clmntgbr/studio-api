package subtitlestyle

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	Save(ctx context.Context, s *Style) error
	Update(ctx context.Context, s *Style) error
}

type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Style, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]Style, error)
}
