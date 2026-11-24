package outputpreset

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Preset struct {
	ID           uuid.UUID
	Name         string
	Width        int
	Height       int
	AspectRatio  string
	CropMode     string
	VideoCodec   string
	AudioCodec   string
	FPS          *float64
	VideoBitrate *int64
	AudioBitrate *int64
	LoudnessLUFS *float64
	Settings     json.RawMessage
}
