package write

import (
	"context"
	"encoding/json"
	"errors"

	domainrender "go-api/internal/domain/render"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type renderWriteRepository struct {
	db *gorm.DB
}

func NewRenderWriteRepository(db *gorm.DB) domainrender.WriteRepository {
	return &renderWriteRepository{db: db}
}

func (r *renderWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *renderWriteRepository) Save(ctx context.Context, render *domainrender.Render) error {
	return DBWithContext(ctx, r.db).Create(renderModelFromDomain(render)).Error
}

func (r *renderWriteRepository) Update(ctx context.Context, render *domainrender.Render) error {
	return DBWithContext(ctx, r.db).Save(renderModelFromDomain(render)).Error
}

func (r *renderWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainrender.Render, error) {
	var model RenderModel
	err := DBWithContext(ctx, r.db).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return renderDomainFromModel(&model), nil
}

func (r *renderWriteRepository) FindByFingerprint(ctx context.Context, mediaFileID uuid.UUID, fingerprint string) (*domainrender.Render, error) {
	var model RenderModel
	err := DBWithContext(ctx, r.db).
		Where("media_file_id = ? AND fingerprint = ?", mediaFileID, fingerprint).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return renderDomainFromModel(&model), nil
}

func (r *renderWriteRepository) HasInFlightByMedia(ctx context.Context, mediaFileID uuid.UUID) (bool, error) {
	var n int64
	err := DBWithContext(ctx, r.db).Model(&RenderModel{}).
		Where("media_file_id = ? AND status IN ?", mediaFileID, []string{
			string(domainrender.StatusPending),
			string(domainrender.StatusQueued),
			string(domainrender.StatusRunning),
		}).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func renderModelFromDomain(r *domainrender.Render) *RenderModel {
	plan := r.Plan
	if len(plan) == 0 {
		plan = json.RawMessage(`{}`)
	}
	parts, _ := json.Marshal(r.OutputParts)
	if len(parts) == 0 || string(parts) == "null" {
		parts = []byte("[]")
	}
	return &RenderModel{
		ID:              r.ID,
		MediaFileID:     r.MediaFileID,
		TimelineID:      r.TimelineID,
		Kind:            string(r.Kind),
		Strategy:        string(r.Strategy),
		ClipID:          r.ClipID,
		OutputPresetID:  r.OutputPresetID,
		SubtitleStyleID: r.SubtitleStyleID,
		Plan:            dbtype.JSONB(plan),
		Fingerprint:     r.Fingerprint,
		Status:          string(r.Status),
		Progress:        r.Progress,
		StorageKey:      r.StorageKey,
		OutputParts:     dbtype.JSONB(parts),
		DurationMs:      r.DurationMs,
		SizeBytes:       r.SizeBytes,
		ErrorMessage:    r.ErrorMessage,
		CreatedAt:       r.CreatedAt,
		StartedAt:       r.StartedAt,
		CompletedAt:     r.CompletedAt,
	}
}

func renderDomainFromModel(m *RenderModel) *domainrender.Render {
	plan := json.RawMessage(m.Plan)
	if len(plan) == 0 {
		plan = json.RawMessage(`{}`)
	}
	var parts []string
	if len(m.OutputParts) > 0 && string(m.OutputParts) != "null" {
		_ = json.Unmarshal(m.OutputParts, &parts)
	}
	return &domainrender.Render{
		ID:              m.ID,
		MediaFileID:     m.MediaFileID,
		TimelineID:      m.TimelineID,
		Kind:            domainrender.Kind(m.Kind),
		Strategy:        domainrender.Strategy(m.Strategy),
		ClipID:          m.ClipID,
		OutputPresetID:  m.OutputPresetID,
		SubtitleStyleID: m.SubtitleStyleID,
		Plan:            plan,
		Fingerprint:     m.Fingerprint,
		Status:          domainrender.Status(m.Status),
		Progress:        m.Progress,
		StorageKey:      m.StorageKey,
		OutputParts:     parts,
		DurationMs:      m.DurationMs,
		SizeBytes:       m.SizeBytes,
		ErrorMessage:    m.ErrorMessage,
		CreatedAt:       m.CreatedAt,
		StartedAt:       m.StartedAt,
		CompletedAt:     m.CompletedAt,
	}
}
