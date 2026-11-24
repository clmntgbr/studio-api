package read

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MediaProjectResolver looks up project_id for progress → realtime fan-out.
type MediaProjectResolver struct {
	db *gorm.DB
}

func NewMediaProjectResolver(db *gorm.DB) *MediaProjectResolver {
	return &MediaProjectResolver{db: db}
}

func (r *MediaProjectResolver) ProjectIDForMedia(ctx context.Context, mediaID uuid.UUID) (uuid.UUID, error) {
	var projectID uuid.UUID
	err := r.db.WithContext(ctx).
		Table("media_file").
		Select("project_id").
		Where("id = ?", mediaID).
		Scan(&projectID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, nil
		}
		return uuid.Nil, err
	}
	return projectID, nil
}
