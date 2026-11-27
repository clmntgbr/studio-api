package media

import (
	"encoding/json"
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type Status string

const (
	StatusUploading  Status = "uploading"
	StatusFetching   Status = "fetching"
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusRendering  Status = "rendering"
	StatusFailed     Status = "failed"
)

// StatusPending is the legacy enum value; prefer StatusUploading for new writes.
const StatusPending Status = "pending"

// IsUploading reports multipart ingress (new uploading or legacy pending).
func (m *MediaFile) IsUploading() bool {
	return m.Status == StatusUploading || m.Status == StatusPending
}

type SourceKind string

const (
	SourceKindUpload SourceKind = "upload"
	SourceKindURL    SourceKind = "url"
)

// Asset kinds for derived objects (three audio tracks + preview/analysis artifacts).
const (
	AssetKindAnalysis      = "analysis"
	AssetKindTranscription = "transcription"
	AssetKindMaster        = "master"
	AssetKindProxy         = "proxy"
	AssetKindPeaks         = "peaks"
	AssetKindWords         = "words"
	AssetKindSRT           = "srt"
	AssetKindScenes        = "scenes"
	AssetKindAdCandidates  = "ad_candidates"
	AssetKindCropPath      = "crop_path"
)

// ProxyMaxShortSide is the preview encode target (720p-class). Sources already at or below
// this short side skip the proxy job and reuse the original as the proxy asset.
const ProxyMaxShortSide = 720

// NeedsProxyEncode reports whether a source should be re-encoded for preview.
func NeedsProxyEncode(width, height int) bool {
	if width <= 0 || height <= 0 {
		return true
	}
	short := width
	if height < short {
		short = height
	}
	return short > ProxyMaxShortSide
}

type StepKind string

const (
	StepFetch      StepKind = "fetch"
	StepProbe      StepKind = "probe"
	StepAudio      StepKind = "audio"
	StepProxy      StepKind = "proxy"
	StepWaveform   StepKind = "waveform"
	StepSilence    StepKind = "silence"
	StepTranscribe StepKind = "transcribe"
	StepText       StepKind = "text"
	StepLLM        StepKind = "llm"
	StepScenes     StepKind = "scenes"
	StepReframe    StepKind = "reframe"
	StepRender     StepKind = "render"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusQueued    StepStatus = "queued"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusCancelled StepStatus = "cancelled"
)

type DecisionAction string

const (
	DecisionRemove  DecisionAction = "remove"
	DecisionSuggest DecisionAction = "suggest"
	DecisionKeep    DecisionAction = "keep"
)

type MediaFile struct {
	ID               uuid.UUID
	Position         int
	SourceKind       SourceKind
	SourceURL        *string
	Title            *string
	OriginalFilename *string
	StorageKey       *string
	MimeType         *string
	DurationMs       *int64
	Width            *int
	Height           *int
	FPS              *float64
	VideoCodec       *string
	AudioCodec       *string
	HasAudio         *bool
	SizeBytes        *int64
	Status           Status
	CreatedAt        time.Time
	UpdatedAt        time.Time

	events []event.DomainEvent
}

func NewUpload(filename string, position int) *MediaFile {
	now := time.Now().UTC()
	fn := filename
	m := &MediaFile{
		ID:               uuid.New(),
		Position:         position,
		SourceKind:       SourceKindUpload,
		OriginalFilename: &fn,
		Status:           StatusUploading,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	m.recordEvent(MediaUploaded{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		Timestamp: now,
	})
	return m
}

func NewFromURL(rawURL string, position int) *MediaFile {
	now := time.Now().UTC()
	u := rawURL
	m := &MediaFile{
		ID:         uuid.New(),
		Position:   position,
		SourceKind: SourceKindURL,
		SourceURL:  &u,
		Status:     StatusFetching,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m.recordEvent(MediaFetchRequested{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		URL:       rawURL,
		Timestamp: now,
	})
	return m
}

func (m *MediaFile) PullEvents() []event.DomainEvent {
	ev := m.events
	m.events = nil
	return ev
}

func (m *MediaFile) recordEvent(e event.DomainEvent) {
	m.events = append(m.events, e)
}

func (m *MediaFile) MarkReady() {
	now := time.Now().UTC()
	m.Status = StatusReady
	m.UpdatedAt = now
	m.recordEvent(MediaReady{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		Timestamp: now,
	})
}

// ReturnToReady restores ready after rendering without re-emitting MediaReady.
func (m *MediaFile) ReturnToReady() {
	m.Status = StatusReady
	m.UpdatedAt = time.Now().UTC()
}

func (m *MediaFile) MarkProcessing() {
	m.Status = StatusProcessing
	m.UpdatedAt = time.Now().UTC()
}

func (m *MediaFile) MarkRendering() {
	m.Status = StatusRendering
	m.UpdatedAt = time.Now().UTC()
}

// Editable reports whether timeline/config edits are allowed.
func (m *MediaFile) Editable() bool {
	return m.Status == StatusReady
}

// CanEnqueueRender reports whether a new encode may be queued.
func (m *MediaFile) CanEnqueueRender() bool {
	return m.Status == StatusReady || m.Status == StatusRendering
}

func (m *MediaFile) RecordDetectionsUpdated(kind string) {
	now := time.Now().UTC()
	m.UpdatedAt = now
	m.recordEvent(DetectionsUpdated{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		Kind:      kind,
		Timestamp: now,
	})
}

func (m *MediaFile) MarkFailed() {
	m.Status = StatusFailed
	m.UpdatedAt = time.Now().UTC()
}

// MarkSourceReady records that the original object is in object storage.
// Upload (webhook) and fetch (result) both use this before probe.
func (m *MediaFile) MarkSourceReady(sourceKey string) {
	key := sourceKey
	m.StorageKey = &key
	m.Status = StatusProcessing
	now := time.Now().UTC()
	m.UpdatedAt = now
	m.recordEvent(MediaSourceReady{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		SourceKey: sourceKey,
		Timestamp: now,
	})
}

func (m *MediaFile) ApplyProbe(durationMs int64, width, height int, fps float64, videoCodec, audioCodec string, hasAudio bool) {
	m.DurationMs = &durationMs
	m.Width = &width
	m.Height = &height
	m.FPS = &fps
	m.VideoCodec = &videoCodec
	m.AudioCodec = &audioCodec
	m.HasAudio = &hasAudio
	m.Status = StatusProcessing
	now := time.Now().UTC()
	m.UpdatedAt = now
	sourceKey := ""
	if m.StorageKey != nil {
		sourceKey = *m.StorageKey
	}
	m.recordEvent(MediaProbed{
		ID:         uuid.New().String(),
		MediaID:    m.ID.String(),
		SourceKey:  sourceKey,
		HasAudio:   hasAudio,
		DurationMs: durationMs,
		Width:      width,
		Height:     height,
		SkipProxy:  !NeedsProxyEncode(width, height),
		Timestamp:  now,
	})
}

func (m *MediaFile) RecordAudioReady(analysisKey, transcriptionKey, masterKey string) {
	now := time.Now().UTC()
	m.UpdatedAt = now
	m.recordEvent(MediaAudioReady{
		ID:               uuid.New().String(),
		MediaID:          m.ID.String(),
		AnalysisKey:      analysisKey,
		TranscriptionKey: transcriptionKey,
		MasterKey:        masterKey,
		Timestamp:        now,
	})
}

func (m *MediaFile) RecordProxyReady(proxyKey string) {
	now := time.Now().UTC()
	m.UpdatedAt = now
	m.recordEvent(MediaProxyReady{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		ProxyKey:  proxyKey,
		Timestamp: now,
	})
}

func (m *MediaFile) RecordWaveformReady(peaksKey string) {
	now := time.Now().UTC()
	m.UpdatedAt = now
	m.recordEvent(MediaWaveformReady{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		PeaksKey:  peaksKey,
		Timestamp: now,
	})
}

func (m *MediaFile) RecordTranscribed(wordsKey, language string) {
	now := time.Now().UTC()
	m.UpdatedAt = now
	m.recordEvent(MediaTranscribed{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		WordsKey:  wordsKey,
		Language:  language,
		Timestamp: now,
	})
}

type Asset struct {
	ID          uuid.UUID
	MediaFileID uuid.UUID
	Kind        string
	StorageKey  string
	SizeBytes   *int64
	Meta        json.RawMessage
	CreatedAt   time.Time
}

func NewAsset(mediaFileID uuid.UUID, kind, storageKey string, sizeBytes *int64, meta json.RawMessage) *Asset {
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	return &Asset{
		ID:          uuid.New(),
		MediaFileID: mediaFileID,
		Kind:        kind,
		StorageKey:  storageKey,
		SizeBytes:   sizeBytes,
		Meta:        meta,
		CreatedAt:   time.Now().UTC(),
	}
}

type Configuration struct {
	MediaFileID          uuid.UUID
	SilenceEnabled       bool
	ThresholdMode        string
	SilenceThresholdDB   *float64
	DetectionLevel       string
	SilenceMinDurationMs int
	PaddingBeforeMs      int
	PaddingAfterMs       int
	SpeechMinDurationMs  int
	FillersEnabled       bool
	RepetitionsEnabled   bool
	FalseStartsEnabled   bool
	SponsorsAction       DecisionAction
	AdsAction            DecisionAction
	Language             *string
	MasterSplit          string // "" | parts:N | durationMs:N — one value for master TikTok splits
	BurnSubtitles        bool
	OutputPresetID       *uuid.UUID // last selected render format
	Settings             json.RawMessage
	UpdatedAt            time.Time
}

func DefaultConfiguration(mediaFileID uuid.UUID) *Configuration {
	return &Configuration{
		MediaFileID:          mediaFileID,
		SilenceEnabled:       true,
		ThresholdMode:        "auto",
		DetectionLevel:       "standard",
		SilenceMinDurationMs: 700,
		PaddingBeforeMs:      120,
		PaddingAfterMs:       200,
		SpeechMinDurationMs:  800,
		FillersEnabled:       false,
		RepetitionsEnabled:   false,
		FalseStartsEnabled:   false,
		SponsorsAction:       DecisionRemove,
		AdsAction:            DecisionRemove,
		Language:             nil, // always auto (transcribe language detection)
		MasterSplit:          "",
		BurnSubtitles:        false,
		OutputPresetID:       nil,
		Settings:             json.RawMessage(`{}`),
		UpdatedAt:            time.Now().UTC(),
	}
}

// NoiseFloorOffsetDb returns the auto-mode offset above measured noise floor.
func (c *Configuration) NoiseFloorOffsetDb() float64 {
	switch c.DetectionLevel {
	case "gentle":
		return 6
	case "standard":
		return 9
	default:
		return 12 // aggressive
	}
}

type ProcessingStep struct {
	ID           uuid.UUID
	MediaFileID  uuid.UUID
	Kind         StepKind
	Status       StepStatus
	Attempt      int
	InputHash    *string
	Progress     float64
	ErrorMessage *string
	QueuedAt     *time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
}

func NewProcessingStep(mediaFileID uuid.UUID, kind StepKind) *ProcessingStep {
	return &ProcessingStep{
		ID:          uuid.New(),
		MediaFileID: mediaFileID,
		Kind:        kind,
		Status:      StepStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
}

// ProcessingStepRun is one execution attempt of a pipeline step for a media file.
// processing_step holds current state; runs keep history and measured durations.
type ProcessingStepRun struct {
	ID           uuid.UUID
	MediaFileID  uuid.UUID
	Kind         StepKind
	Attempt      int
	Status       StepStatus
	StartedAt    time.Time
	CompletedAt  *time.Time
	DurationMs   *int64
	ErrorMessage *string
	CreatedAt    time.Time
}

func NewProcessingStepRun(mediaFileID uuid.UUID, kind StepKind, attempt int, startedAt, completedAt time.Time, status StepStatus, errMsg *string) *ProcessingStepRun {
	var dur *int64
	if !completedAt.IsZero() && !startedAt.IsZero() && !completedAt.Before(startedAt) {
		ms := completedAt.Sub(startedAt).Milliseconds()
		dur = &ms
	}
	var completed *time.Time
	if !completedAt.IsZero() {
		c := completedAt
		completed = &c
	}
	return &ProcessingStepRun{
		ID:           uuid.New(),
		MediaFileID:  mediaFileID,
		Kind:         kind,
		Attempt:      attempt,
		Status:       status,
		StartedAt:    startedAt,
		CompletedAt:  completed,
		DurationMs:   dur,
		ErrorMessage: errMsg,
		CreatedAt:    time.Now().UTC(),
	}
}
