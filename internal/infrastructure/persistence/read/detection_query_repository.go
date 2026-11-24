package read

import (
	"context"
	"encoding/json"
	"time"

	domaindetection "go-api/internal/domain/detection"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type detectionReadRepository struct {
	db *gorm.DB
}

func NewDetectionReadRepository(db *gorm.DB) domaindetection.ReadRepository {
	return &detectionReadRepository{db: db}
}

type detectionScan struct {
	ID             uuid.UUID    `gorm:"column:id"`
	MediaFileID    uuid.UUID    `gorm:"column:media_file_id"`
	Kind           string       `gorm:"column:kind"`
	SourceStartMs  int64        `gorm:"column:source_start_ms"`
	SourceEndMs    int64        `gorm:"column:source_end_ms"`
	Confidence     *float64     `gorm:"column:confidence"`
	Detector       string       `gorm:"column:detector"`
	FirstWordIndex *int         `gorm:"column:first_word_index"`
	LastWordIndex  *int         `gorm:"column:last_word_index"`
	Payload        dbtype.JSONB `gorm:"column:payload"`
	CreatedAt      time.Time    `gorm:"column:created_at"`
}

func (r *detectionReadRepository) ListByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]domaindetection.Detection, error) {
	var rows []detectionScan
	err := r.db.WithContext(ctx).
		Table("detection").
		Where("media_file_id = ?", mediaFileID).
		Order("source_start_ms ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domaindetection.Detection, 0, len(rows))
	for _, row := range rows {
		payload := json.RawMessage(row.Payload)
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}
		out = append(out, domaindetection.Detection{
			ID:             row.ID,
			MediaFileID:    row.MediaFileID,
			Kind:           domaindetection.Kind(row.Kind),
			SourceStartMs:  row.SourceStartMs,
			SourceEndMs:    row.SourceEndMs,
			Confidence:     row.Confidence,
			Detector:       row.Detector,
			FirstWordIndex: row.FirstWordIndex,
			LastWordIndex:  row.LastWordIndex,
			Payload:        payload,
			CreatedAt:      row.CreatedAt,
		})
	}
	return out, nil
}

func (r *detectionReadRepository) ListOverridesByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]domaindetection.Override, error) {
	type overrideScan struct {
		ID            uuid.UUID `gorm:"column:id"`
		MediaFileID   uuid.UUID `gorm:"column:media_file_id"`
		Kind          *string   `gorm:"column:kind"`
		Action        string    `gorm:"column:action"`
		SourceStartMs int64     `gorm:"column:source_start_ms"`
		SourceEndMs   int64     `gorm:"column:source_end_ms"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}
	var rows []overrideScan
	err := r.db.WithContext(ctx).
		Table("user_override").
		Where("media_file_id = ?", mediaFileID).
		Order("source_start_ms ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domaindetection.Override, 0, len(rows))
	for _, row := range rows {
		var kind *domaindetection.Kind
		if row.Kind != nil {
			k := domaindetection.Kind(*row.Kind)
			kind = &k
		}
		out = append(out, domaindetection.Override{
			ID:            row.ID,
			MediaFileID:   row.MediaFileID,
			Kind:          kind,
			Action:        domaindetection.DecisionAction(row.Action),
			SourceStartMs: row.SourceStartMs,
			SourceEndMs:   row.SourceEndMs,
			CreatedAt:     row.CreatedAt,
		})
	}
	return out, nil
}
