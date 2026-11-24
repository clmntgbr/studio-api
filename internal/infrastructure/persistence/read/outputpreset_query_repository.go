package read

import (
	"context"
	"encoding/json"
	"errors"

	domainpreset "go-api/internal/domain/outputpreset"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type outputPresetReadRepository struct {
	db *gorm.DB
}

func NewOutputPresetReadRepository(db *gorm.DB) domainpreset.ReadRepository {
	return &outputPresetReadRepository{db: db}
}

type outputPresetScan struct {
	ID           uuid.UUID    `gorm:"column:id"`
	Name         string       `gorm:"column:name"`
	Width        int          `gorm:"column:width"`
	Height       int          `gorm:"column:height"`
	AspectRatio  string       `gorm:"column:aspect_ratio"`
	CropMode     string       `gorm:"column:crop_mode"`
	VideoCodec   string       `gorm:"column:video_codec"`
	AudioCodec   string       `gorm:"column:audio_codec"`
	FPS          *float64     `gorm:"column:fps"`
	VideoBitrate *int64       `gorm:"column:video_bitrate"`
	AudioBitrate *int64       `gorm:"column:audio_bitrate"`
	LoudnessLUFS *float64     `gorm:"column:loudness_lufs"`
	Settings     dbtype.JSONB `gorm:"column:settings"`
}

func (r *outputPresetReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainpreset.Preset, error) {
	var row outputPresetScan
	err := r.db.WithContext(ctx).Table("output_preset").Where("id = ?", id).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	p := presetFromScan(row)
	return &p, nil
}

func (r *outputPresetReadRepository) List(ctx context.Context) ([]domainpreset.Preset, error) {
	var rows []outputPresetScan
	if err := r.db.WithContext(ctx).Table("output_preset").Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domainpreset.Preset, 0, len(rows))
	for _, row := range rows {
		out = append(out, presetFromScan(row))
	}
	return out, nil
}

func presetFromScan(row outputPresetScan) domainpreset.Preset {
	settings := json.RawMessage(row.Settings)
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	return domainpreset.Preset{
		ID:           row.ID,
		Name:         row.Name,
		Width:        row.Width,
		Height:       row.Height,
		AspectRatio:  row.AspectRatio,
		CropMode:     row.CropMode,
		VideoCodec:   row.VideoCodec,
		AudioCodec:   row.AudioCodec,
		FPS:          row.FPS,
		VideoBitrate: row.VideoBitrate,
		AudioBitrate: row.AudioBitrate,
		LoudnessLUFS: row.LoudnessLUFS,
		Settings:     settings,
	}
}
