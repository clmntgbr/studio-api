package renderplan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type Segment struct {
	MediaKey  string `json:"mediaKey"`
	SrcStartMs int64  `json:"srcStartMs"`
	SrcEndMs   int64  `json:"srcEndMs"`
}

type Crop struct {
	Mode    string  `json:"mode"`
	PathKey *string `json:"pathKey,omitempty"`
}

type Output struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	FPS    float64 `json:"fps"`
	Codec  string  `json:"codec"`
	CRF    int     `json:"crf"`
	Preset string  `json:"preset"`
}

type Audio struct {
	Codec         string  `json:"codec"`
	LoudnessLUFS  float64 `json:"loudnessLufs"`
	FadeMs        int     `json:"fadeMs"`
}

type Subtitles struct {
	ASSKey   string `json:"assKey,omitempty"`
	FontsDir string `json:"fontsDir"`
}

type Overlay struct {
	Kind    string `json:"kind"`
	Text    string `json:"text,omitempty"`
	StartMs int64  `json:"startMs"`
	EndMs   int64  `json:"endMs"`
}

// Plan is the complete render job payload for worker-render.
type Plan struct {
	RenderID  string     `json:"renderId"`
	Kind      string     `json:"kind"`
	Segments  []Segment  `json:"segments"`
	Crop      Crop       `json:"crop"`
	Output    Output     `json:"output"`
	Audio     Audio      `json:"audio"`
	Subtitles *Subtitles `json:"subtitles,omitempty"`
	Overlays  []Overlay  `json:"overlays,omitempty"`
	OutputKey string     `json:"outputKey"`
}

func (p Plan) Marshal() (json.RawMessage, error) {
	return json.Marshal(p)
}

// Fingerprint hashes the durable render inputs (timeline, preset, style, crop, kind, clip).
func Fingerprint(timelineID, presetID uuid.UUID, styleID, clipID *uuid.UUID, kind, cropMode string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s", timelineID, presetID, kind, cropMode)
	if styleID != nil {
		fmt.Fprintf(h, "|style:%s", styleID.String())
	}
	if clipID != nil {
		fmt.Fprintf(h, "|clip:%s", clipID.String())
	}
	return hex.EncodeToString(h.Sum(nil))
}
