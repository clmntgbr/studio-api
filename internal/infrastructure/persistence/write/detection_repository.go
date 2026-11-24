package write

import (
	"context"
	"encoding/json"

	domaindetection "go-api/internal/domain/detection"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type detectionWriteRepository struct {
	db *gorm.DB
}

func NewDetectionWriteRepository(db *gorm.DB) domaindetection.WriteRepository {
	return &detectionWriteRepository{db: db}
}

func (r *detectionWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *detectionWriteRepository) Save(ctx context.Context, d *domaindetection.Detection) error {
	return DBWithContext(ctx, r.db).Create(detectionModelFromDomain(d)).Error
}

func (r *detectionWriteRepository) SaveMany(ctx context.Context, items []*domaindetection.Detection) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([]*DetectionModel, 0, len(items))
	for _, d := range items {
		rows = append(rows, detectionModelFromDomain(d))
	}
	return DBWithContext(ctx, r.db).CreateInBatches(rows, 500).Error
}

func (r *detectionWriteRepository) DeleteByMediaAndKind(ctx context.Context, mediaFileID uuid.UUID, kind domaindetection.Kind) error {
	return DBWithContext(ctx, r.db).
		Where("media_file_id = ? AND kind = ?", mediaFileID, string(kind)).
		Delete(&DetectionModel{}).Error
}

func (r *detectionWriteRepository) SaveOverride(ctx context.Context, o *domaindetection.Override) error {
	return DBWithContext(ctx, r.db).Create(overrideModelFromDomain(o)).Error
}

func (r *detectionWriteRepository) DeleteOverride(ctx context.Context, id uuid.UUID) error {
	return DBWithContext(ctx, r.db).Where("id = ?", id).Delete(&UserOverrideModel{}).Error
}

func (r *detectionWriteRepository) DeleteOverridesByMedia(ctx context.Context, mediaFileID uuid.UUID) error {
	return DBWithContext(ctx, r.db).Where("media_file_id = ?", mediaFileID).Delete(&UserOverrideModel{}).Error
}

func detectionModelFromDomain(d *domaindetection.Detection) *DetectionModel {
	meta := d.Payload
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	return &DetectionModel{
		ID:             d.ID,
		MediaFileID:    d.MediaFileID,
		Kind:           string(d.Kind),
		SourceStartMs:  d.SourceStartMs,
		SourceEndMs:    d.SourceEndMs,
		Confidence:     d.Confidence,
		Detector:       d.Detector,
		FirstWordIndex: d.FirstWordIndex,
		LastWordIndex:  d.LastWordIndex,
		Payload:        dbtype.JSONB(meta),
		CreatedAt:      d.CreatedAt,
	}
}

func overrideModelFromDomain(o *domaindetection.Override) *UserOverrideModel {
	var kind *string
	if o.Kind != nil {
		k := string(*o.Kind)
		kind = &k
	}
	return &UserOverrideModel{
		ID:            o.ID,
		MediaFileID:   o.MediaFileID,
		Kind:          kind,
		Action:        string(o.Action),
		SourceStartMs: o.SourceStartMs,
		SourceEndMs:   o.SourceEndMs,
		CreatedAt:     o.CreatedAt,
	}
}
