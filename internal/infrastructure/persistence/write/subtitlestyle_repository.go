package write

import (
	"context"
	"encoding/json"
	"errors"

	domainstyle "go-api/internal/domain/subtitlestyle"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type subtitleStyleWriteRepository struct {
	db *gorm.DB
}

func NewSubtitleStyleWriteRepository(db *gorm.DB) domainstyle.WriteRepository {
	return &subtitleStyleWriteRepository{db: db}
}

func (r *subtitleStyleWriteRepository) Save(ctx context.Context, s *domainstyle.Style) error {
	return DBWithContext(ctx, r.db).Create(subtitleStyleModelFromDomain(s)).Error
}

func (r *subtitleStyleWriteRepository) Update(ctx context.Context, s *domainstyle.Style) error {
	return DBWithContext(ctx, r.db).Save(subtitleStyleModelFromDomain(s)).Error
}

func (r *subtitleStyleWriteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return DBWithContext(ctx, r.db).Delete(&SubtitleStyleModel{}, "id = ?", id).Error
}

func (r *subtitleStyleWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainstyle.Style, error) {
	var model SubtitleStyleModel
	err := DBWithContext(ctx, r.db).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return subtitleStyleDomainFromModel(&model), nil
}

func subtitleStyleModelFromDomain(s *domainstyle.Style) *SubtitleStyleModel {
	settings := s.Settings
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	var wordsPerLine *int16
	if s.WordsPerLine != nil {
		v := int16(*s.WordsPerLine)
		wordsPerLine = &v
	}
	return &SubtitleStyleModel{
		ID:             s.ID,
		Name:           s.Name,
		FontFamily:     s.FontFamily,
		FontSize:       s.FontSize,
		BaseColor:      s.BaseColor,
		HighlightColor: s.HighlightColor,
		OutlineColor:   s.OutlineColor,
		OutlineSize:    s.OutlineSize,
		WordsPerLine:   wordsPerLine,
		Position:       s.Position,
		Uppercase:      s.Uppercase,
		Settings:       dbtype.JSONB(settings),
		CreatedAt:      s.CreatedAt,
	}
}

func subtitleStyleDomainFromModel(m *SubtitleStyleModel) *domainstyle.Style {
	settings := json.RawMessage(m.Settings)
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	var wordsPerLine *int
	if m.WordsPerLine != nil {
		v := int(*m.WordsPerLine)
		wordsPerLine = &v
	}
	return &domainstyle.Style{
		ID:             m.ID,
		Name:           m.Name,
		FontFamily:     m.FontFamily,
		FontSize:       m.FontSize,
		BaseColor:      m.BaseColor,
		HighlightColor: m.HighlightColor,
		OutlineColor:   m.OutlineColor,
		OutlineSize:    m.OutlineSize,
		WordsPerLine:   wordsPerLine,
		Position:       m.Position,
		Uppercase:      m.Uppercase,
		Settings:       settings,
		CreatedAt:      m.CreatedAt,
	}
}
