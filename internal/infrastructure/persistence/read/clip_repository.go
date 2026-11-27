package read

import (
	"context"
	"errors"

	domainclip "go-api/internal/domain/clip"
	"go-api/internal/infrastructure/persistence/write"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type clipReadRepository struct {
	db *gorm.DB
}

func NewClipReadRepository(db *gorm.DB) domainclip.ReadRepository {
	return &clipReadRepository{db: db}
}

func (r *clipReadRepository) ListByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]domainclip.Clip, error) {
	var rows []write.ClipModel
	if err := r.db.WithContext(ctx).
		Where("media_file_id = ?", mediaFileID).
		Order("source_start_ms ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domainclip.Clip, 0, len(rows))
	for i := range rows {
		out = append(out, *clipDomainFromModel(&rows[i]))
	}
	return out, nil
}

func (r *clipReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainclip.Clip, error) {
	var row write.ClipModel
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return clipDomainFromModel(&row), nil
}

func clipDomainFromModel(m *write.ClipModel) *domainclip.Clip {
	tags := []string(m.Hashtags)
	if tags == nil {
		tags = []string{}
	}
	return &domainclip.Clip{
		ID:            m.ID,
		MediaFileID:   m.MediaFileID,
		SourceStartMs: m.SourceStartMs,
		SourceEndMs:   m.SourceEndMs,
		Origin:        m.Origin,
		Score:         m.Score,
		Hook:          m.Hook,
		Reason:        m.Reason,
		Title:         m.Title,
		Description:   m.Description,
		Hashtags:      tags,
		Selected:      m.Selected,
		CreatedAt:     m.CreatedAt,
	}
}
