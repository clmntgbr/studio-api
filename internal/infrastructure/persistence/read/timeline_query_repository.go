package read

import (
	"context"
	"errors"
	"time"

	domaintimeline "go-api/internal/domain/timeline"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type timelineReadRepository struct {
	db *gorm.DB
}

func NewTimelineReadRepository(db *gorm.DB) domaintimeline.ReadRepository {
	return &timelineReadRepository{db: db}
}

func (r *timelineReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domaintimeline.Timeline, error) {
	var row timelineScan
	err := r.db.WithContext(ctx).Table("timeline").Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return timelineFromScan(row), nil
}

func (r *timelineReadRepository) FindHeadByProject(ctx context.Context, projectID uuid.UUID) (*domaintimeline.Timeline, error) {
	var row timelineScan
	err := r.db.WithContext(ctx).
		Table("timeline").
		Where("project_id = ? AND status IN ?", projectID, []string{
			string(domaintimeline.StatusDraft),
			string(domaintimeline.StatusValidated),
		}).
		Order("version DESC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return timelineFromScan(row), nil
}

func (r *timelineReadRepository) ListSegments(ctx context.Context, timelineID uuid.UUID) ([]domaintimeline.Segment, error) {
	type segScan struct {
		ID                  uuid.UUID `gorm:"column:id"`
		TimelineID          uuid.UUID `gorm:"column:timeline_id"`
		SegmentIndex        int       `gorm:"column:segment_index"`
		MediaFileID         uuid.UUID `gorm:"column:media_file_id"`
		SourceStartMs       int64     `gorm:"column:source_start_ms"`
		SourceEndMs         int64     `gorm:"column:source_end_ms"`
		OutputStartMs       int64     `gorm:"column:output_start_ms"`
		OutputEndMs         int64     `gorm:"column:output_end_ms"`
		RemovedBeforeReason *string   `gorm:"column:removed_before_reason"`
	}
	var rows []segScan
	err := r.db.WithContext(ctx).
		Table("timeline_segment").
		Where("timeline_id = ?", timelineID).
		Order("segment_index ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domaintimeline.Segment, 0, len(rows))
	for _, row := range rows {
		var reason *domaintimeline.CutReason
		if row.RemovedBeforeReason != nil {
			r := domaintimeline.CutReason(*row.RemovedBeforeReason)
			reason = &r
		}
		out = append(out, domaintimeline.Segment{
			ID:                  row.ID,
			TimelineID:          row.TimelineID,
			SegmentIndex:        row.SegmentIndex,
			MediaFileID:         row.MediaFileID,
			SourceStartMs:       row.SourceStartMs,
			SourceEndMs:         row.SourceEndMs,
			OutputStartMs:       row.OutputStartMs,
			OutputEndMs:         row.OutputEndMs,
			RemovedBeforeReason: reason,
		})
	}
	return out, nil
}

func (r *timelineReadRepository) ListDecisions(ctx context.Context, timelineID uuid.UUID) ([]domaintimeline.Decision, error) {
	type decScan struct {
		TimelineID  uuid.UUID `gorm:"column:timeline_id"`
		DetectionID uuid.UUID `gorm:"column:detection_id"`
		Action      string    `gorm:"column:action"`
		Origin      string    `gorm:"column:origin"`
		Reason      *string   `gorm:"column:reason"`
	}
	var rows []decScan
	err := r.db.WithContext(ctx).Table("timeline_decision").Where("timeline_id = ?", timelineID).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domaintimeline.Decision, 0, len(rows))
	for _, row := range rows {
		out = append(out, domaintimeline.Decision{
			TimelineID:  row.TimelineID,
			DetectionID: row.DetectionID,
			Action:      domaintimeline.DecisionAction(row.Action),
			Origin:      domaintimeline.DecisionOrigin(row.Origin),
			Reason:      row.Reason,
		})
	}
	return out, nil
}

type timelineScan struct {
	ID          uuid.UUID  `gorm:"column:id"`
	ProjectID   uuid.UUID  `gorm:"column:project_id"`
	Version     int        `gorm:"column:version"`
	Status      string     `gorm:"column:status"`
	Fingerprint string     `gorm:"column:fingerprint"`
	DurationMs  int64      `gorm:"column:duration_ms"`
	ValidatedAt *time.Time `gorm:"column:validated_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func timelineFromScan(row timelineScan) *domaintimeline.Timeline {
	return &domaintimeline.Timeline{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		Version:     row.Version,
		Status:      domaintimeline.Status(row.Status),
		Fingerprint: row.Fingerprint,
		DurationMs:  row.DurationMs,
		ValidatedAt: row.ValidatedAt,
		CreatedAt:   row.CreatedAt,
	}
}
