package read

import (
	"context"
	"errors"
	"fmt"

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
	var raw string
	err := r.db.WithContext(ctx).
		Table("media_file").
		Select("project_id").
		Where("id = ?", mediaID).
		Limit(1).
		Scan(&raw).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, nil
		}
		return uuid.Nil, err
	}
	if raw == "" {
		return uuid.Nil, nil
	}
	projectID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse project_id: %w", err)
	}
	return projectID, nil
}
