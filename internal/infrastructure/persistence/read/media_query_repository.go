package read

import (
	"context"
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
