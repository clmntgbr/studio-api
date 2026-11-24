package transcript

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, t *Transcript) error
	Update(ctx context.Context, t *Transcript) error
	ReplaceWords(ctx context.Context, transcriptID uuid.UUID, words []Word) error
}

type ReadRepository interface {
	FindByMediaID(ctx context.Context, mediaFileID uuid.UUID) (*Transcript, error)
	ListWords(ctx context.Context, transcriptID uuid.UUID) ([]Word, error)
}
