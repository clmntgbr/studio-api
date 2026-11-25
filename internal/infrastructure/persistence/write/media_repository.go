package write

import (
	"context"
	"encoding/json"
	"errors"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mediaWriteRepository struct {
	db *gorm.DB
}

func NewMediaWriteRepository(db *gorm.DB) domainmedia.WriteRepository {
	return &mediaWriteRepository{db: db}
}

func (r *mediaWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *mediaWriteRepository) Save(ctx context.Context, m *domainmedia.MediaFile) error {
	return DBWithContext(ctx, r.db).Create(mediaFileModelFromDomain(m)).Error
}

func (r *mediaWriteRepository) Update(ctx context.Context, m *domainmedia.MediaFile) error {
	return DBWithContext(ctx, r.db).Save(mediaFileModelFromDomain(m)).Error
}

func (r *mediaWriteRepository) UpdateFromStatuses(ctx context.Context, m *domainmedia.MediaFile, from []domainmedia.Status) (bool, error) {
	if len(from) == 0 {
		return false, errors.New("from statuses required")
	}
	statuses := make([]string, len(from))
	for i, s := range from {
		statuses[i] = string(s)
	}
	res := DBWithContext(ctx, r.db).
		Model(&MediaFileModel{}).
		Where("id = ? AND status IN ?", m.ID, statuses).
		Select("*").
		Updates(mediaFileModelFromDomain(m))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *mediaWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainmedia.MediaFile, error) {
	var model MediaFileModel
	err := DBWithContext(ctx, r.db).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mediaFileDomainFromModel(&model), nil
}

func (r *mediaWriteRepository) SaveAsset(ctx context.Context, asset *domainmedia.Asset) error {
	return DBWithContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "media_file_id"}, {Name: "kind"}},
			DoUpdates: clause.AssignmentColumns([]string{"storage_key", "size_bytes", "meta"}),
		}).
		Create(mediaAssetModelFromDomain(asset)).Error
}

func (r *mediaWriteRepository) SaveConfiguration(ctx context.Context, cfg *domainmedia.Configuration) error {
	return DBWithContext(ctx, r.db).Create(mediaConfigurationModelFromDomain(cfg)).Error
}

func (r *mediaWriteRepository) GetConfiguration(ctx context.Context, mediaFileID uuid.UUID) (*domainmedia.Configuration, error) {
	var model MediaConfigurationModel
	err := DBWithContext(ctx, r.db).First(&model, "media_file_id = ?", mediaFileID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mediaConfigurationDomainFromModel(&model), nil
}

func (r *mediaWriteRepository) UpdateConfiguration(ctx context.Context, cfg *domainmedia.Configuration) error {
	return DBWithContext(ctx, r.db).Save(mediaConfigurationModelFromDomain(cfg)).Error
}

func (r *mediaWriteRepository) GetAssetByKind(ctx context.Context, mediaFileID uuid.UUID, kind string) (*domainmedia.Asset, error) {
	var model MediaAssetModel
	err := DBWithContext(ctx, r.db).First(&model, "media_file_id = ? AND kind = ?", mediaFileID, kind).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mediaAssetDomainFromModel(&model), nil
}

func (r *mediaWriteRepository) SaveStep(ctx context.Context, step *domainmedia.ProcessingStep) error {
	return DBWithContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "media_file_id"}, {Name: "kind"}},
			DoNothing: true,
		}).
		Create(processingStepModelFromDomain(step)).Error
}

func (r *mediaWriteRepository) UpdateStep(ctx context.Context, step *domainmedia.ProcessingStep) error {
	return DBWithContext(ctx, r.db).Save(processingStepModelFromDomain(step)).Error
}

func (r *mediaWriteRepository) GetStep(ctx context.Context, mediaFileID uuid.UUID, kind domainmedia.StepKind) (*domainmedia.ProcessingStep, error) {
	var model ProcessingStepModel
	err := DBWithContext(ctx, r.db).First(&model, "media_file_id = ? AND kind = ?", mediaFileID, string(kind)).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return processingStepDomainFromModel(&model), nil
}

func (r *mediaWriteRepository) SaveStepRun(ctx context.Context, run *domainmedia.ProcessingStepRun) error {
	return DBWithContext(ctx, r.db).Create(processingStepRunModelFromDomain(run)).Error
}

func processingStepRunModelFromDomain(r *domainmedia.ProcessingStepRun) *ProcessingStepRunModel {
	return &ProcessingStepRunModel{
		ID:           r.ID,
		MediaFileID:  r.MediaFileID,
		Kind:         string(r.Kind),
		Attempt:      r.Attempt,
		Status:       string(r.Status),
		StartedAt:    r.StartedAt,
		CompletedAt:  r.CompletedAt,
		DurationMs:   r.DurationMs,
		ErrorMessage: r.ErrorMessage,
		CreatedAt:    r.CreatedAt,
	}
}

func processingStepDomainFromModel(s *ProcessingStepModel) *domainmedia.ProcessingStep {
	return &domainmedia.ProcessingStep{
		ID:           s.ID,
		MediaFileID:  s.MediaFileID,
		Kind:         domainmedia.StepKind(s.Kind),
		Status:       domainmedia.StepStatus(s.Status),
		Attempt:      s.Attempt,
		InputHash:    s.InputHash,
		Progress:     s.Progress,
		ErrorMessage: s.ErrorMessage,
		QueuedAt:     s.QueuedAt,
		StartedAt:    s.StartedAt,
		CompletedAt:  s.CompletedAt,
		CreatedAt:    s.CreatedAt,
	}
}

func (r *mediaWriteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return DBWithContext(ctx, r.db).Delete(&MediaFileModel{}, id).Error
}

func mediaFileModelFromDomain(m *domainmedia.MediaFile) *MediaFileModel {
	return &MediaFileModel{
		ID:               m.ID,
		ProjectID:        m.ProjectID,
		Position:         m.Position,
		SourceKind:       string(m.SourceKind),
		SourceURL:        m.SourceURL,
		Title:            m.Title,
		OriginalFilename: m.OriginalFilename,
		StorageKey:       m.StorageKey,
		MimeType:         m.MimeType,
		DurationMs:       m.DurationMs,
		Width:            m.Width,
		Height:           m.Height,
		FPS:              m.FPS,
		VideoCodec:       m.VideoCodec,
		AudioCodec:       m.AudioCodec,
		HasAudio:         m.HasAudio,
		SizeBytes:        m.SizeBytes,
		Status:           string(m.Status),
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func mediaFileDomainFromModel(m *MediaFileModel) *domainmedia.MediaFile {
	return &domainmedia.MediaFile{
		ID:               m.ID,
		ProjectID:        m.ProjectID,
		Position:         m.Position,
		SourceKind:       domainmedia.SourceKind(m.SourceKind),
		SourceURL:        m.SourceURL,
		Title:            m.Title,
		OriginalFilename: m.OriginalFilename,
		StorageKey:       m.StorageKey,
		MimeType:         m.MimeType,
		DurationMs:       m.DurationMs,
		Width:            m.Width,
		Height:           m.Height,
		FPS:              m.FPS,
		VideoCodec:       m.VideoCodec,
		AudioCodec:       m.AudioCodec,
		HasAudio:         m.HasAudio,
		SizeBytes:        m.SizeBytes,
		Status:           domainmedia.Status(m.Status),
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func mediaAssetModelFromDomain(a *domainmedia.Asset) *MediaAssetModel {
	meta := a.Meta
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	return &MediaAssetModel{
		ID:          a.ID,
		MediaFileID: a.MediaFileID,
		Kind:        a.Kind,
		StorageKey:  a.StorageKey,
		SizeBytes:   a.SizeBytes,
		Meta:        dbtype.JSONB(meta),
		CreatedAt:   a.CreatedAt,
	}
}

func mediaConfigurationModelFromDomain(c *domainmedia.Configuration) *MediaConfigurationModel {
	settings := c.Settings
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	return &MediaConfigurationModel{
		MediaFileID:          c.MediaFileID,
		SilenceEnabled:       c.SilenceEnabled,
		ThresholdMode:        c.ThresholdMode,
		SilenceThresholdDB:   c.SilenceThresholdDB,
		DetectionLevel:       c.DetectionLevel,
		SilenceMinDurationMs: c.SilenceMinDurationMs,
		PaddingBeforeMs:      c.PaddingBeforeMs,
		PaddingAfterMs:       c.PaddingAfterMs,
		SpeechMinDurationMs:  c.SpeechMinDurationMs,
		FillersEnabled:       c.FillersEnabled,
		RepetitionsEnabled:   c.RepetitionsEnabled,
		FalseStartsEnabled:   c.FalseStartsEnabled,
		SponsorsAction:       string(c.SponsorsAction),
		AdsAction:            string(c.AdsAction),
		Language:             c.Language,
		MasterSplit:          c.MasterSplit,
		BurnSubtitles:        c.BurnSubtitles,
		OutputPresetID:       c.OutputPresetID,
		Settings:             dbtype.JSONB(settings),
		UpdatedAt:            c.UpdatedAt,
	}
}

func mediaConfigurationDomainFromModel(c *MediaConfigurationModel) *domainmedia.Configuration {
	settings := json.RawMessage(c.Settings)
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	return &domainmedia.Configuration{
		MediaFileID:          c.MediaFileID,
		SilenceEnabled:       c.SilenceEnabled,
		ThresholdMode:        c.ThresholdMode,
		SilenceThresholdDB:   c.SilenceThresholdDB,
		DetectionLevel:       c.DetectionLevel,
		SilenceMinDurationMs: c.SilenceMinDurationMs,
		PaddingBeforeMs:      c.PaddingBeforeMs,
		PaddingAfterMs:       c.PaddingAfterMs,
		SpeechMinDurationMs:  c.SpeechMinDurationMs,
		FillersEnabled:       c.FillersEnabled,
		RepetitionsEnabled:   c.RepetitionsEnabled,
		FalseStartsEnabled:   c.FalseStartsEnabled,
		SponsorsAction:       domainmedia.DecisionAction(c.SponsorsAction),
		AdsAction:            domainmedia.DecisionAction(c.AdsAction),
		Language:             c.Language,
		MasterSplit:          c.MasterSplit,
		BurnSubtitles:        c.BurnSubtitles,
		OutputPresetID:       c.OutputPresetID,
		Settings:             settings,
		UpdatedAt:            c.UpdatedAt,
	}
}

func mediaAssetDomainFromModel(a *MediaAssetModel) *domainmedia.Asset {
	meta := json.RawMessage(a.Meta)
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	return &domainmedia.Asset{
		ID:          a.ID,
		MediaFileID: a.MediaFileID,
		Kind:        a.Kind,
		StorageKey:  a.StorageKey,
		SizeBytes:   a.SizeBytes,
		Meta:        meta,
		CreatedAt:   a.CreatedAt,
	}
}

func processingStepModelFromDomain(s *domainmedia.ProcessingStep) *ProcessingStepModel {
	return &ProcessingStepModel{
		ID:           s.ID,
		MediaFileID:  s.MediaFileID,
		Kind:         string(s.Kind),
		Status:       string(s.Status),
		Attempt:      s.Attempt,
		InputHash:    s.InputHash,
		Progress:     s.Progress,
		ErrorMessage: s.ErrorMessage,
		QueuedAt:     s.QueuedAt,
		StartedAt:    s.StartedAt,
		CompletedAt:  s.CompletedAt,
		CreatedAt:    s.CreatedAt,
	}
}
