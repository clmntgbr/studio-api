package preset

import (
	"encoding/json"

	"github.com/google/uuid"
)

type AspectRatio string

const (
	Aspect16x9   AspectRatio = "16:9"
	Aspect9x16   AspectRatio = "9:16"
	Aspect1x1    AspectRatio = "1:1"
	Aspect4x5    AspectRatio = "4:5"
	AspectCustom AspectRatio = "custom"
)

type CropMode string

const (
	CropNone         CropMode = "none"
	CropCenter       CropMode = "center"
	CropLeft         CropMode = "left"
	CropRight        CropMode = "right"
	CropCustom       CropMode = "custom"
	CropFitBlur      CropMode = "fit_blur"
	CropFaceTrack    CropMode = "face_track"
	CropSpeakerTrack CropMode = "speaker_track"
)

type OutputPreset struct {
	ID           uuid.UUID
	Name         string
	Width        int
	Height       int
	AspectRatio  AspectRatio
	CropMode     CropMode
	VideoCodec   string
	AudioCodec   string
	FPS          *float64
	VideoBitrate *int64
	AudioBitrate *int64
	LoudnessLUFS *float64
	Settings     json.RawMessage
}

func New(name string, width, height int, aspect AspectRatio) *OutputPreset {
	return &OutputPreset{
		ID:          uuid.New(),
		Name:        name,
		Width:       width,
		Height:      height,
		AspectRatio: aspect,
		CropMode:    CropNone,
		VideoCodec:  "h264",
		AudioCodec:  "aac",
		Settings:    json.RawMessage(`{}`),
	}
}
