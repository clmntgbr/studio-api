package write

import (
	"context"

	domainclip "go-api/internal/domain/clip"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type clipWriteRepository struct {
	db *gorm.DB
}

func NewClipWriteRepository(db *gorm.DB) domainclip.WriteRepository {
	return &clipWriteRepository{db: db}
}

func (r *clipWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *clipWriteRepository) Save(ctx context.Context, c *domainclip.Clip) error {
	return DBWithContext(ctx, r.db).Create(clipModelFromDomain(c)).Error
}

func (r *clipWriteRepository) Update(ctx context.Context, c *domainclip.Clip) error {
	return DBWithContext(ctx, r.db).Save(clipModelFromDomain(c)).Error
}

func (r *clipWriteRepository) DeleteByMediaAndOrigin(ctx context.Context, mediaFileID uuid.UUID, origin string) error {
	return DBWithContext(ctx, r.db).
		Where("media_file_id = ? AND origin = ?", mediaFileID, origin).
		Delete(&ClipModel{}).Error
}

func clipModelFromDomain(c *domainclip.Clip) *ClipModel {
	tags := c.Hashtags
	if tags == nil {
		tags = []string{}
	}
	return &ClipModel{
		ID:            c.ID,
		ProjectID:     c.ProjectID,
		MediaFileID:   c.MediaFileID,
		SourceStartMs: c.SourceStartMs,
		SourceEndMs:   c.SourceEndMs,
		Origin:        c.Origin,
		Score:         c.Score,
		Hook:          c.Hook,
		Reason:        c.Reason,
		Title:         c.Title,
		Description:   c.Description,
		Hashtags:      dbtype.StringArray(tags),
		Selected:      c.Selected,
		CreatedAt:     c.CreatedAt,
	}
}
