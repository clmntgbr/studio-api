package media

import (
	"encoding/json"
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusFetching   Status = "fetching"
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusFailed     Status = "failed"
)

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
	AssetKindCropPath      = "crop_path"
)

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
	ProjectID        uuid.UUID
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

func NewUpload(projectID uuid.UUID, filename string, position int) *MediaFile {
	now := time.Now().UTC()
	fn := filename
	m := &MediaFile{
		ID:               uuid.New(),
		ProjectID:        projectID,
		Position:         position,
		SourceKind:       SourceKindUpload,
		OriginalFilename: &fn,
		Status:           StatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	m.recordEvent(MediaUploaded{
		ID:        uuid.New().String(),
		MediaID:   m.ID.String(),
		ProjectID: projectID.String(),
		Timestamp: now,
	})
	return m
}

func NewFromURL(projectID uuid.UUID, rawURL string, position int) *MediaFile {
	now := time.Now().UTC()
	u := rawURL
	m := &MediaFile{
		ID:         uuid.New(),
		ProjectID:  projectID,
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
		ProjectID: projectID.String(),
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
	m.Status = StatusReady
	m.UpdatedAt = time.Now().UTC()
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
		ProjectID: m.ProjectID.String(),
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
		ProjectID:  m.ProjectID.String(),
		SourceKey:  sourceKey,
		HasAudio:   hasAudio,
		DurationMs: durationMs,
		Timestamp:  now,
	})
}

func (m *MediaFile) RecordAudioReady(analysisKey, transcriptionKey, masterKey string) {
	now := time.Now().UTC()
	m.UpdatedAt = now
	m.recordEvent(MediaAudioReady{
		ID:               uuid.New().String(),
		MediaID:          m.ID.String(),
		ProjectID:        m.ProjectID.String(),
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
		ProjectID: m.ProjectID.String(),
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
		ProjectID: m.ProjectID.String(),
		PeaksKey:  peaksKey,
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
	Settings             json.RawMessage
	UpdatedAt            time.Time
}

func DefaultConfiguration(mediaFileID uuid.UUID) *Configuration {
	return &Configuration{
		MediaFileID:          mediaFileID,
		SilenceEnabled:       true,
		ThresholdMode:        "auto",
		DetectionLevel:       "aggressive",
		SilenceMinDurationMs: 500,
		PaddingBeforeMs:      50,
		PaddingAfterMs:       150,
		SpeechMinDurationMs:  300,
		FillersEnabled:       true,
		RepetitionsEnabled:   true,
		FalseStartsEnabled:   false,
		SponsorsAction:       DecisionSuggest,
		AdsAction:            DecisionSuggest,
		Settings:             json.RawMessage(`{}`),
		UpdatedAt:            time.Now().UTC(),
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
