package detection

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindSilence    Kind = "silence"
	KindFiller     Kind = "filler"
	KindRepetition Kind = "repetition"
	KindFalseStart Kind = "false_start"
	KindSponsor    Kind = "sponsor"
	KindAd         Kind = "ad"
)

type DecisionAction string

const (
	ActionRemove  DecisionAction = "remove"
	ActionSuggest DecisionAction = "suggest"
	ActionKeep    DecisionAction = "keep"
)

type Detection struct {
	ID             uuid.UUID
	MediaFileID    uuid.UUID
	Kind           Kind
	SourceStartMs  int64
	SourceEndMs    int64
	Confidence     *float64
	Detector       string
	FirstWordIndex *int
	LastWordIndex  *int
	Payload        json.RawMessage
	CreatedAt      time.Time
}

func New(mediaFileID uuid.UUID, kind Kind, startMs, endMs int64, detector string, payload json.RawMessage) *Detection {
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	return &Detection{
		ID:            uuid.New(),
		MediaFileID:   mediaFileID,
		Kind:          kind,
		SourceStartMs: startMs,
		SourceEndMs:   endMs,
		Detector:      detector,
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	}
}

// Override is a user keep/remove decision anchored on a source time range.
type Override struct {
	ID            uuid.UUID
	MediaFileID   uuid.UUID
	Kind          *Kind
	Action        DecisionAction
	SourceStartMs int64
	SourceEndMs   int64
	CreatedAt     time.Time
}

func NewOverride(mediaFileID uuid.UUID, action DecisionAction, startMs, endMs int64, kind *Kind) *Override {
	return &Override{
		ID:            uuid.New(),
		MediaFileID:   mediaFileID,
		Kind:          kind,
		Action:        action,
		SourceStartMs: startMs,
		SourceEndMs:   endMs,
		CreatedAt:     time.Now().UTC(),
	}
}
