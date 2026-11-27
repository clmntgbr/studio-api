package render

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindMaster Kind = "master"
	KindClip   Kind = "clip"
	KindAudio  Kind = "audio"
	KindCustom Kind = "custom"
)

type Strategy string

const (
	StrategyFullRender Strategy = "full_render"
	StrategyStreamCopy Strategy = "stream_copy"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusSkipped   Status = "skipped"
	StatusCancelled Status = "cancelled"
)

type Render struct {
	ID              uuid.UUID
	MediaFileID     uuid.UUID
	TimelineID      uuid.UUID
	Kind            Kind
	Strategy        Strategy
	ClipID          *uuid.UUID
	OutputPresetID  uuid.UUID
	SubtitleStyleID *uuid.UUID
	Plan            json.RawMessage
	Fingerprint     string
	Status          Status
	Progress        float64
	StorageKey      *string
	OutputParts     []string // extra keys after masterSplit (part-01…)
	DurationMs      *int64
	SizeBytes       *int64
	ErrorMessage    *string
	CreatedAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
}

func New(mediaFileID, timelineID, presetID uuid.UUID, kind Kind, fingerprint string, plan json.RawMessage) *Render {
	return &Render{
		ID:             uuid.New(),
		MediaFileID:    mediaFileID,
		TimelineID:     timelineID,
		Kind:           kind,
		Strategy:       StrategyFullRender,
		OutputPresetID: presetID,
		Plan:           plan,
		Fingerprint:    fingerprint,
		Status:         StatusPending,
		CreatedAt:      time.Now().UTC(),
	}
}
