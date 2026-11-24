package read

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domainstyle "go-api/internal/domain/subtitlestyle"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type subtitleStyleReadRepository struct {
	db *gorm.DB
}

func NewSubtitleStyleReadRepository(db *gorm.DB) domainstyle.ReadRepository {
	return &subtitleStyleReadRepository{db: db}
}

type subtitleStyleScan struct {
	ID             uuid.UUID    `gorm:"column:id"`
	ProjectID      *uuid.UUID   `gorm:"column:project_id"`
	Name           string       `gorm:"column:name"`
	FontFamily     *string      `gorm:"column:font_family"`
	FontSize       *int         `gorm:"column:font_size"`
	BaseColor      *string      `gorm:"column:base_color"`
	HighlightColor *string      `gorm:"column:highlight_color"`
	OutlineColor   *string      `gorm:"column:outline_color"`
	OutlineSize    *int         `gorm:"column:outline_size"`
	WordsPerLine   *int16       `gorm:"column:words_per_line"`
	Position       *string      `gorm:"column:position"`
	Uppercase      bool         `gorm:"column:uppercase"`
	Settings       dbtype.JSONB `gorm:"column:settings"`
	CreatedAt      time.Time    `gorm:"column:created_at"`
}

func (r *subtitleStyleReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainstyle.Style, error) {
	var row subtitleStyleScan
	err := r.db.WithContext(ctx).Table("subtitle_style").Where("id = ?", id).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	s := styleFromScan(row)
	return &s, nil
}

func (r *subtitleStyleReadRepository) List(ctx context.Context, projectID *uuid.UUID) ([]domainstyle.Style, error) {
	q := r.db.WithContext(ctx).Table("subtitle_style").Order("created_at ASC")
	if projectID != nil {
		q = q.Where("project_id = ? OR project_id IS NULL", *projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	var rows []subtitleStyleScan
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domainstyle.Style, 0, len(rows))
	for _, row := range rows {
		out = append(out, styleFromScan(row))
	}
	return out, nil
}

func styleFromScan(row subtitleStyleScan) domainstyle.Style {
	settings := json.RawMessage(row.Settings)
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	var wordsPerLine *int
	if row.WordsPerLine != nil {
		v := int(*row.WordsPerLine)
		wordsPerLine = &v
	}
	return domainstyle.Style{
		ID:             row.ID,
		ProjectID:      row.ProjectID,
		Name:           row.Name,
		FontFamily:     row.FontFamily,
		FontSize:       row.FontSize,
		BaseColor:      row.BaseColor,
		HighlightColor: row.HighlightColor,
		OutlineColor:   row.OutlineColor,
		OutlineSize:    row.OutlineSize,
		WordsPerLine:   wordsPerLine,
		Position:       row.Position,
		Uppercase:      row.Uppercase,
		Settings:       settings,
		CreatedAt:      row.CreatedAt,
	}
}
