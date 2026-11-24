package read

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domainrender "go-api/internal/domain/render"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type renderReadRepository struct {
	db *gorm.DB
}

func NewRenderReadRepository(db *gorm.DB) domainrender.ReadRepository {
	return &renderReadRepository{db: db}
}

type renderScan struct {
	ID              uuid.UUID    `gorm:"column:id"`
	ProjectID       uuid.UUID    `gorm:"column:project_id"`
	TimelineID      uuid.UUID    `gorm:"column:timeline_id"`
	Kind            string       `gorm:"column:kind"`
	Strategy        string       `gorm:"column:strategy"`
	ClipID          *uuid.UUID   `gorm:"column:clip_id"`
	OutputPresetID  uuid.UUID    `gorm:"column:output_preset_id"`
	SubtitleStyleID *uuid.UUID   `gorm:"column:subtitle_style_id"`
	Plan            dbtype.JSONB `gorm:"column:plan"`
	Fingerprint     string       `gorm:"column:fingerprint"`
	Status          string       `gorm:"column:status"`
	Progress        float64      `gorm:"column:progress"`
	StorageKey      *string      `gorm:"column:storage_key"`
	DurationMs      *int64       `gorm:"column:duration_ms"`
	SizeBytes       *int64       `gorm:"column:size_bytes"`
	ErrorMessage    *string      `gorm:"column:error_message"`
	CreatedAt       time.Time    `gorm:"column:created_at"`
	StartedAt       *time.Time   `gorm:"column:started_at"`
	CompletedAt     *time.Time   `gorm:"column:completed_at"`
}

func (r *renderReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainrender.Render, error) {
	var row renderScan
	err := r.db.WithContext(ctx).Table("render").Where("id = ?", id).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := renderFromScan(row)
	return &out, nil
}

func (r *renderReadRepository) FindByFingerprint(ctx context.Context, projectID uuid.UUID, fingerprint string) (*domainrender.Render, error) {
	var row renderScan
	err := r.db.WithContext(ctx).Table("render").
		Where("project_id = ? AND fingerprint = ?", projectID, fingerprint).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := renderFromScan(row)
	return &out, nil
}

func (r *renderReadRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domainrender.Render, error) {
	var rows []renderScan
	err := r.db.WithContext(ctx).Table("render").
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domainrender.Render, 0, len(rows))
	for _, row := range rows {
		out = append(out, renderFromScan(row))
	}
	return out, nil
}

func renderFromScan(row renderScan) domainrender.Render {
	plan := json.RawMessage(row.Plan)
	if len(plan) == 0 {
		plan = json.RawMessage(`{}`)
	}
	return domainrender.Render{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		TimelineID:      row.TimelineID,
		Kind:            domainrender.Kind(row.Kind),
		Strategy:        domainrender.Strategy(row.Strategy),
		ClipID:          row.ClipID,
		OutputPresetID:  row.OutputPresetID,
		SubtitleStyleID: row.SubtitleStyleID,
		Plan:            plan,
		Fingerprint:     row.Fingerprint,
		Status:          domainrender.Status(row.Status),
		Progress:        row.Progress,
		StorageKey:      row.StorageKey,
		DurationMs:      row.DurationMs,
		SizeBytes:       row.SizeBytes,
		ErrorMessage:    row.ErrorMessage,
		CreatedAt:       row.CreatedAt,
		StartedAt:       row.StartedAt,
		CompletedAt:     row.CompletedAt,
	}
}
