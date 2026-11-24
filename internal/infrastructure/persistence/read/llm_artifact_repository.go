package read

import (
	"context"
	"encoding/json"

	domainllm "go-api/internal/domain/llmartifact"
	"go-api/internal/infrastructure/persistence/write"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type llmArtifactReadRepository struct {
	db *gorm.DB
}

func NewLLMArtifactReadRepository(db *gorm.DB) domainllm.ReadRepository {
	return &llmArtifactReadRepository{db: db}
}

func (r *llmArtifactReadRepository) ListByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]domainllm.Artifact, error) {
	var rows []write.LLMArtifactModel
	if err := r.db.WithContext(ctx).
		Where("media_file_id = ?", mediaFileID).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return llmArtifactsFromModels(rows), nil
}

func (r *llmArtifactReadRepository) ListByMediaAndKind(ctx context.Context, mediaFileID uuid.UUID, kind string) ([]domainllm.Artifact, error) {
	var rows []write.LLMArtifactModel
	if err := r.db.WithContext(ctx).
		Where("media_file_id = ? AND kind = ?", mediaFileID, kind).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return llmArtifactsFromModels(rows), nil
}

func llmArtifactsFromModels(rows []write.LLMArtifactModel) []domainllm.Artifact {
	out := make([]domainllm.Artifact, 0, len(rows))
	for _, m := range rows {
		payload := json.RawMessage(m.Payload)
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}
		out = append(out, domainllm.Artifact{
			ID:          m.ID,
			MediaFileID: m.MediaFileID,
			Kind:        m.Kind,
			Payload:     payload,
			Provider:    m.Provider,
			CreatedAt:   m.CreatedAt,
		})
	}
	return out
}
