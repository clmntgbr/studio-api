package read

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mediaReadRepository struct {
	db *gorm.DB
}

func NewMediaReadRepository(db *gorm.DB) domainmedia.ReadRepository {
	return &mediaReadRepository{db: db}
}

type mediaScan struct {
	ID               uuid.UUID `gorm:"column:id"`
	ProjectID        uuid.UUID `gorm:"column:project_id"`
	Position         int       `gorm:"column:position"`
	SourceKind       string    `gorm:"column:source_kind"`
	SourceURL        *string   `gorm:"column:source_url"`
	Title            *string   `gorm:"column:title"`
	OriginalFilename *string   `gorm:"column:original_filename"`
	StorageKey       *string   `gorm:"column:storage_key"`
	MimeType         *string   `gorm:"column:mime_type"`
	DurationMs       *int64    `gorm:"column:duration_ms"`
	Width            *int      `gorm:"column:width"`
	Height           *int      `gorm:"column:height"`
	FPS              *float64  `gorm:"column:fps"`
	VideoCodec       *string   `gorm:"column:video_codec"`
	AudioCodec       *string   `gorm:"column:audio_codec"`
	HasAudio         *bool     `gorm:"column:has_audio"`
	SizeBytes        *int64    `gorm:"column:size_bytes"`
	Status           string    `gorm:"column:status"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (r *mediaReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainmedia.FileView, error) {
	var m mediaScan
	err := r.db.WithContext(ctx).Table("media_file").Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	view := scanMediaView(m)
	steps, err := r.ListSteps(ctx, id)
	if err != nil {
		return nil, err
	}
	view.Steps = steps
	return view, nil
}

type stepScan struct {
	Kind         string     `gorm:"column:kind"`
	Status       string     `gorm:"column:status"`
	Progress     float64    `gorm:"column:progress"`
	ErrorMessage *string    `gorm:"column:error_message"`
	QueuedAt     *time.Time `gorm:"column:queued_at"`
	StartedAt    *time.Time `gorm:"column:started_at"`
	CompletedAt  *time.Time `gorm:"column:completed_at"`
}

func (r *mediaReadRepository) ListSteps(ctx context.Context, mediaFileID uuid.UUID) ([]domainmedia.StepView, error) {
	var rows []stepScan
	err := r.db.WithContext(ctx).
		Table("processing_step").
		Where("media_file_id = ?", mediaFileID).
		Order("created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domainmedia.StepView, 0, len(rows))
	for _, row := range rows {
		out = append(out, domainmedia.StepView{
			Kind:         domainmedia.StepKind(row.Kind),
			Status:       domainmedia.StepStatus(row.Status),
			Progress:     row.Progress,
			ErrorMessage: row.ErrorMessage,
			QueuedAt:     row.QueuedAt,
			StartedAt:    row.StartedAt,
			CompletedAt:  row.CompletedAt,
			DurationMs:   stepDurationMs(row.StartedAt, row.QueuedAt, row.CompletedAt),
		})
	}
	return out, nil
}

func stepDurationMs(startedAt, queuedAt, completedAt *time.Time) *int64 {
	if completedAt == nil {
		return nil
	}
	start := startedAt
	if start == nil {
		start = queuedAt
	}
	if start == nil {
		return nil
	}
	ms := completedAt.Sub(*start).Milliseconds()
	if ms < 0 {
		ms = 0
	}
	return &ms
}

func (r *mediaReadRepository) ListStepRuns(ctx context.Context, mediaFileID uuid.UUID) ([]domainmedia.StepRunView, error) {
	var rows []struct {
		ID           uuid.UUID  `gorm:"column:id"`
		Kind         string     `gorm:"column:kind"`
		Attempt      int        `gorm:"column:attempt"`
		Status       string     `gorm:"column:status"`
		StartedAt    time.Time  `gorm:"column:started_at"`
		CompletedAt  *time.Time `gorm:"column:completed_at"`
		DurationMs   *int64     `gorm:"column:duration_ms"`
		ErrorMessage *string    `gorm:"column:error_message"`
		CreatedAt    time.Time  `gorm:"column:created_at"`
	}
	err := r.db.WithContext(ctx).
		Table("processing_step_run").
		Where("media_file_id = ?", mediaFileID).
		Order("created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domainmedia.StepRunView, 0, len(rows))
	for _, row := range rows {
		out = append(out, domainmedia.StepRunView{
			ID:           row.ID,
			Kind:         domainmedia.StepKind(row.Kind),
			Attempt:      row.Attempt,
			Status:       domainmedia.StepStatus(row.Status),
			StartedAt:    row.StartedAt,
			CompletedAt:  row.CompletedAt,
			DurationMs:   row.DurationMs,
			ErrorMessage: row.ErrorMessage,
			CreatedAt:    row.CreatedAt,
		})
	}
	return out, nil
}

func (r *mediaReadRepository) ListByProject(ctx context.Context, projectID uuid.UUID, query paginate.PaginateQuery) ([]domainmedia.FileView, int64, error) {
	db := r.db.WithContext(ctx).Table("media_file").Where("project_id = ?", projectID)
	if query.Search != "" {
		db = db.Where("(title ILIKE ? OR original_filename ILIKE ?)", "%"+query.Search+"%", "%"+query.Search+"%")
	}
	if query.SortBy == "" {
		query.SortBy = "position"
	}
	scoped, total, err := Paginate(db, query)
	if err != nil {
		return nil, 0, err
	}
	var rows []mediaScan
	if err := scoped.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	views := make([]domainmedia.FileView, 0, len(rows))
	for _, row := range rows {
		views = append(views, *scanMediaView(row))
	}
	return views, total, nil
}

func (r *mediaReadRepository) GetConfiguration(ctx context.Context, mediaFileID uuid.UUID) (*domainmedia.Configuration, error) {
	type cfgScan struct {
		MediaFileID          uuid.UUID       `gorm:"column:media_file_id"`
		SilenceEnabled       bool            `gorm:"column:silence_enabled"`
		ThresholdMode        string          `gorm:"column:threshold_mode"`
		SilenceThresholdDB   *float64        `gorm:"column:silence_threshold_db"`
		DetectionLevel       string          `gorm:"column:detection_level"`
		SilenceMinDurationMs int             `gorm:"column:silence_min_duration_ms"`
		PaddingBeforeMs      int             `gorm:"column:padding_before_ms"`
		PaddingAfterMs       int             `gorm:"column:padding_after_ms"`
		SpeechMinDurationMs  int             `gorm:"column:speech_min_duration_ms"`
		FillersEnabled       bool            `gorm:"column:fillers_enabled"`
		RepetitionsEnabled   bool            `gorm:"column:repetitions_enabled"`
		FalseStartsEnabled   bool            `gorm:"column:false_starts_enabled"`
		SponsorsAction       string          `gorm:"column:sponsors_action"`
		AdsAction            string          `gorm:"column:ads_action"`
		Language             *string         `gorm:"column:language"`
		MasterSplit          string          `gorm:"column:master_split"`
		BurnSubtitles        bool            `gorm:"column:burn_subtitles"`
		OutputPresetID       *uuid.UUID      `gorm:"column:output_preset_id"`
		Settings             json.RawMessage `gorm:"column:settings"`
		UpdatedAt            time.Time       `gorm:"column:updated_at"`
	}
	var row cfgScan
	err := r.db.WithContext(ctx).Table("media_configuration").Where("media_file_id = ?", mediaFileID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	settings := row.Settings
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	return &domainmedia.Configuration{
		MediaFileID:          row.MediaFileID,
		SilenceEnabled:       row.SilenceEnabled,
		ThresholdMode:        row.ThresholdMode,
		SilenceThresholdDB:   row.SilenceThresholdDB,
		DetectionLevel:       row.DetectionLevel,
		SilenceMinDurationMs: row.SilenceMinDurationMs,
		PaddingBeforeMs:      row.PaddingBeforeMs,
		PaddingAfterMs:       row.PaddingAfterMs,
		SpeechMinDurationMs:  row.SpeechMinDurationMs,
		FillersEnabled:       row.FillersEnabled,
		RepetitionsEnabled:   row.RepetitionsEnabled,
		FalseStartsEnabled:   row.FalseStartsEnabled,
		SponsorsAction:       domainmedia.DecisionAction(row.SponsorsAction),
		AdsAction:            domainmedia.DecisionAction(row.AdsAction),
		Language:             row.Language,
		MasterSplit:          row.MasterSplit,
		BurnSubtitles:        row.BurnSubtitles,
		OutputPresetID:       row.OutputPresetID,
		Settings:             settings,
		UpdatedAt:            row.UpdatedAt,
	}, nil
}

func scanMediaView(m mediaScan) *domainmedia.FileView {
	return &domainmedia.FileView{
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
