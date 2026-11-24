package read

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RenderProjectResolver looks up project_id for render progress → realtime fan-out.
type RenderProjectResolver struct {
	db *gorm.DB
}

func NewRenderProjectResolver(db *gorm.DB) *RenderProjectResolver {
	return &RenderProjectResolver{db: db}
}

func (r *RenderProjectResolver) ProjectIDForRender(ctx context.Context, renderID uuid.UUID) (uuid.UUID, error) {
	var raw string
	err := r.db.WithContext(ctx).
		Table("render").
		Select("project_id").
		Where("id = ?", renderID).
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
