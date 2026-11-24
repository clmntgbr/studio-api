package write

import (
	"context"
	"errors"

	domaintimeline "go-api/internal/domain/timeline"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type timelineWriteRepository struct {
	db *gorm.DB
}

func NewTimelineWriteRepository(db *gorm.DB) domaintimeline.WriteRepository {
	return &timelineWriteRepository{db: db}
}

func (r *timelineWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *timelineWriteRepository) Save(ctx context.Context, t *domaintimeline.Timeline, segments []domaintimeline.Segment, decisions []domaintimeline.Decision) error {
	db := DBWithContext(ctx, r.db)
	if err := db.Create(timelineModelFromDomain(t)).Error; err != nil {
		return err
	}
	for i := range segments {
		seg := segments[i]
		seg.TimelineID = t.ID
		if seg.ID == uuid.Nil {
			seg.ID = uuid.New()
		}
		if err := db.Create(timelineSegmentModelFromDomain(&seg)).Error; err != nil {
			return err
		}
	}
	for i := range decisions {
		dec := decisions[i]
		dec.TimelineID = t.ID
		if err := db.Create(timelineDecisionModelFromDomain(&dec)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *timelineWriteRepository) Update(ctx context.Context, t *domaintimeline.Timeline) error {
	return DBWithContext(ctx, r.db).Save(timelineModelFromDomain(t)).Error
}

func (r *timelineWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domaintimeline.Timeline, error) {
	var model TimelineModel
	err := DBWithContext(ctx, r.db).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return timelineDomainFromModel(&model), nil
}

func (r *timelineWriteRepository) SupersedeDrafts(ctx context.Context, projectID uuid.UUID) error {
	return DBWithContext(ctx, r.db).
		Model(&TimelineModel{}).
		Where("project_id = ? AND status = ?", projectID, string(domaintimeline.StatusDraft)).
		Update("status", string(domaintimeline.StatusSuperseded)).Error
}

func (r *timelineWriteRepository) NextVersion(ctx context.Context, projectID uuid.UUID) (int, error) {
	var max *int
	err := DBWithContext(ctx, r.db).
		Model(&TimelineModel{}).
		Select("MAX(version)").
		Where("project_id = ?", projectID).
		Scan(&max).Error
	if err != nil {
		return 0, err
	}
	if max == nil {
		return 1, nil
	}
	return *max + 1, nil
}

func timelineModelFromDomain(t *domaintimeline.Timeline) *TimelineModel {
	return &TimelineModel{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Version:     t.Version,
		Status:      string(t.Status),
		Fingerprint: t.Fingerprint,
		DurationMs:  t.DurationMs,
		ValidatedAt: t.ValidatedAt,
		CreatedAt:   t.CreatedAt,
	}
}

func timelineDomainFromModel(t *TimelineModel) *domaintimeline.Timeline {
	return &domaintimeline.Timeline{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Version:     t.Version,
		Status:      domaintimeline.Status(t.Status),
		Fingerprint: t.Fingerprint,
		DurationMs:  t.DurationMs,
		ValidatedAt: t.ValidatedAt,
		CreatedAt:   t.CreatedAt,
	}
}

func timelineSegmentModelFromDomain(s *domaintimeline.Segment) *TimelineSegmentModel {
	var reason *string
	if s.RemovedBeforeReason != nil {
		r := string(*s.RemovedBeforeReason)
		reason = &r
	}
	return &TimelineSegmentModel{
		ID:                  s.ID,
		TimelineID:          s.TimelineID,
		SegmentIndex:        s.SegmentIndex,
		MediaFileID:         s.MediaFileID,
		SourceStartMs:       s.SourceStartMs,
		SourceEndMs:         s.SourceEndMs,
		OutputStartMs:       s.OutputStartMs,
		OutputEndMs:         s.OutputEndMs,
		RemovedBeforeReason: reason,
	}
}

func timelineDecisionModelFromDomain(d *domaintimeline.Decision) *TimelineDecisionModel {
	return &TimelineDecisionModel{
		TimelineID:  d.TimelineID,
		DetectionID: d.DetectionID,
		Action:      string(d.Action),
		Origin:      string(d.Origin),
		Reason:      d.Reason,
	}
}
