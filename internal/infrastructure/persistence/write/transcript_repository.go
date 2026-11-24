package write

import (
	"context"
	"errors"

	domaintranscript "go-api/internal/domain/transcript"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type transcriptWriteRepository struct {
	db *gorm.DB
}

func NewTranscriptWriteRepository(db *gorm.DB) domaintranscript.WriteRepository {
	return &transcriptWriteRepository{db: db}
}

func (r *transcriptWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *transcriptWriteRepository) Save(ctx context.Context, t *domaintranscript.Transcript) error {
	db := DBWithContext(ctx, r.db)
	var existing TranscriptModel
	err := db.Where("media_file_id = ?", t.MediaFileID).First(&existing).Error
	if err == nil {
		t.ID = existing.ID
		t.CreatedAt = existing.CreatedAt
		return db.Save(transcriptModelFromDomain(t)).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(transcriptModelFromDomain(t)).Error
}

func (r *transcriptWriteRepository) Update(ctx context.Context, t *domaintranscript.Transcript) error {
	return DBWithContext(ctx, r.db).Save(transcriptModelFromDomain(t)).Error
}

func (r *transcriptWriteRepository) ReplaceWords(ctx context.Context, transcriptID uuid.UUID, words []domaintranscript.Word) error {
	db := DBWithContext(ctx, r.db)
	if err := db.Where("transcript_id = ?", transcriptID).Delete(&TranscriptWordModel{}).Error; err != nil {
		return err
	}
	if len(words) == 0 {
		return nil
	}
	models := make([]TranscriptWordModel, 0, len(words))
	for _, w := range words {
		models = append(models, TranscriptWordModel{
			TranscriptID:  transcriptID,
			WordIndex:     w.WordIndex,
			Text:          w.Text,
			SourceStartMs: w.SourceStartMs,
			SourceEndMs:   w.SourceEndMs,
			Confidence:    w.Confidence,
			Speaker:       w.Speaker,
		})
	}
	return db.Create(&models).Error
}

func transcriptModelFromDomain(t *domaintranscript.Transcript) *TranscriptModel {
	return &TranscriptModel{
		ID:            t.ID,
		MediaFileID:   t.MediaFileID,
		Language:      t.Language,
		Text:          t.Text,
		Provider:      t.Provider,
		ProviderJobID: t.ProviderJobID,
		Status:        string(t.Status),
		CreatedAt:     t.CreatedAt,
	}
}
