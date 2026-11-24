package timeline

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft      Status = "draft"
	StatusValidated  Status = "validated"
	StatusSuperseded Status = "superseded"
)

type DecisionAction string

const (
	ActionRemove  DecisionAction = "remove"
	ActionSuggest DecisionAction = "suggest"
	ActionKeep    DecisionAction = "keep"
)

type DecisionOrigin string

const (
	OriginAuto DecisionOrigin = "auto"
	OriginUser DecisionOrigin = "user"
)

type CutReason string

const (
	CutSilence    CutReason = "silence"
	CutFiller     CutReason = "filler"
	CutRepetition CutReason = "repetition"
	CutFalseStart CutReason = "false_start"
	CutSponsor    CutReason = "sponsor"
	CutAd         CutReason = "ad"
	CutManual     CutReason = "manual"
	CutMicro      CutReason = "micro"
)

type Timeline struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Version     int
	Status      Status
	Fingerprint string
	DurationMs  int64
	ValidatedAt *time.Time
	CreatedAt   time.Time
}

func New(projectID uuid.UUID, version int, fingerprint string, durationMs int64) *Timeline {
	return &Timeline{
		ID:          uuid.New(),
		ProjectID:   projectID,
		Version:     version,
		Status:      StatusDraft,
		Fingerprint: fingerprint,
		DurationMs:  durationMs,
		CreatedAt:   time.Now().UTC(),
	}
}

func (t *Timeline) Validate() {
	now := time.Now().UTC()
	t.Status = StatusValidated
	t.ValidatedAt = &now
}

type Segment struct {
	ID                  uuid.UUID
	TimelineID          uuid.UUID
	SegmentIndex        int
	MediaFileID         uuid.UUID
	SourceStartMs       int64
	SourceEndMs         int64
	OutputStartMs       int64
	OutputEndMs         int64
	RemovedBeforeReason *CutReason
}

type Decision struct {
	TimelineID  uuid.UUID
	DetectionID uuid.UUID
	Action      DecisionAction
	Origin      DecisionOrigin
	Reason      *string
}
