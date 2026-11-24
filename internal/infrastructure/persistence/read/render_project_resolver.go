package read

import (
	"context"
	"errors"

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
	var projectID uuid.UUID
	err := r.db.WithContext(ctx).
		Table("render").
		Select("project_id").
		Where("id = ?", renderID).
		Scan(&projectID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, nil
		}
		return uuid.Nil, err
	}
	return projectID, nil
}
