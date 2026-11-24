package write

import (
	"context"
	"encoding/json"

	domainllm "go-api/internal/domain/llmartifact"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type llmArtifactWriteRepository struct {
	db *gorm.DB
}

func NewLLMArtifactWriteRepository(db *gorm.DB) domainllm.WriteRepository {
	return &llmArtifactWriteRepository{db: db}
}

func (r *llmArtifactWriteRepository) Save(ctx context.Context, a *domainllm.Artifact) error {
	return DBWithContext(ctx, r.db).Create(llmArtifactModelFromDomain(a)).Error
}

func (r *llmArtifactWriteRepository) DeleteByMediaAndKind(ctx context.Context, mediaFileID uuid.UUID, kind string) error {
	return DBWithContext(ctx, r.db).
		Where("media_file_id = ? AND kind = ?", mediaFileID, kind).
		Delete(&LLMArtifactModel{}).Error
}

func llmArtifactModelFromDomain(a *domainllm.Artifact) *LLMArtifactModel {
	payload := a.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	return &LLMArtifactModel{
		ID:          a.ID,
		MediaFileID: a.MediaFileID,
		Kind:        a.Kind,
		Payload:     dbtype.JSONB(payload),
		Provider:    a.Provider,
		CreatedAt:   a.CreatedAt,
	}
}
