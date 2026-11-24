package read

import (
	"context"
	"errors"
	"time"

	domaintranscript "go-api/internal/domain/transcript"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type transcriptReadRepository struct {
	db *gorm.DB
}

func NewTranscriptReadRepository(db *gorm.DB) domaintranscript.ReadRepository {
	return &transcriptReadRepository{db: db}
}

func (r *transcriptReadRepository) FindByMediaID(ctx context.Context, mediaFileID uuid.UUID) (*domaintranscript.Transcript, error) {
	var row struct {
		ID            uuid.UUID `gorm:"column:id"`
		MediaFileID   uuid.UUID `gorm:"column:media_file_id"`
		Language      *string   `gorm:"column:language"`
		Text          *string   `gorm:"column:text"`
		Provider      string    `gorm:"column:provider"`
		ProviderJobID *string   `gorm:"column:provider_job_id"`
		Status        string    `gorm:"column:status"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}
	err := r.db.WithContext(ctx).Table("transcript").Where("media_file_id = ?", mediaFileID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domaintranscript.Transcript{
		ID:            row.ID,
		MediaFileID:   row.MediaFileID,
		Language:      row.Language,
		Text:          row.Text,
		Provider:      row.Provider,
		ProviderJobID: row.ProviderJobID,
		Status:        domaintranscript.Status(row.Status),
		CreatedAt:     row.CreatedAt,
	}, nil
}

func (r *transcriptReadRepository) ListWords(ctx context.Context, transcriptID uuid.UUID) ([]domaintranscript.Word, error) {
	var rows []struct {
		ID            int64     `gorm:"column:id"`
		TranscriptID  uuid.UUID `gorm:"column:transcript_id"`
		WordIndex     int       `gorm:"column:word_index"`
		Text          string    `gorm:"column:text"`
		SourceStartMs int64     `gorm:"column:source_start_ms"`
		SourceEndMs   int64     `gorm:"column:source_end_ms"`
		Confidence    *float64  `gorm:"column:confidence"`
		Speaker       *string   `gorm:"column:speaker"`
	}
	err := r.db.WithContext(ctx).
		Table("transcript_word").
		Where("transcript_id = ?", transcriptID).
		Order("word_index ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domaintranscript.Word, 0, len(rows))
	for _, row := range rows {
		out = append(out, domaintranscript.Word{
			ID:            row.ID,
			TranscriptID:  row.TranscriptID,
			WordIndex:     row.WordIndex,
			Text:          row.Text,
			SourceStartMs: row.SourceStartMs,
			SourceEndMs:   row.SourceEndMs,
			Confidence:    row.Confidence,
			Speaker:       row.Speaker,
		})
	}
	return out, nil
}
