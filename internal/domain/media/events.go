package media

import "time"

const (
	EventTypeMediaUploaded          = "media.uploaded.v1"
	EventTypeMediaFetchRequested    = "media.fetch_requested.v1"
	EventTypeMediaSourceReady       = "media.source_ready.v1"
	EventTypeMediaProbed            = "media.probed.v1"
	EventTypeMediaAudioReady        = "media.audio_ready.v1"
	EventTypeMediaProxyReady        = "media.proxy_ready.v1"
	EventTypeMediaWaveformReady     = "media.waveform_ready.v1"
	EventTypeMediaReady             = "media.ready.v1"
	EventTypeMediaDetectionsUpdated = "media.detections_updated.v1"
	EventTypeMediaTranscribed       = "media.transcribed.v1"
)

type MediaUploaded struct {
	ID        string    `json:"eventId"`
	MediaID   string    `json:"mediaId"`
	ProjectID string    `json:"projectId"`
	Timestamp time.Time `json:"timestamp"`
}

func (e MediaUploaded) EventID() string       { return e.ID }
func (e MediaUploaded) EventType() string     { return EventTypeMediaUploaded }
func (e MediaUploaded) AggregateID() string   { return e.MediaID }
func (e MediaUploaded) OccurredAt() time.Time { return e.Timestamp }

type MediaFetchRequested struct {
	ID        string    `json:"eventId"`
	MediaID   string    `json:"mediaId"`
	ProjectID string    `json:"projectId"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

func (e MediaFetchRequested) EventID() string       { return e.ID }
func (e MediaFetchRequested) EventType() string     { return EventTypeMediaFetchRequested }
func (e MediaFetchRequested) AggregateID() string   { return e.MediaID }
func (e MediaFetchRequested) OccurredAt() time.Time { return e.Timestamp }

// MediaSourceReady is emitted when the original file is in MinIO
// (presigned upload completed or fetch finished). Downstream is identical.
type MediaSourceReady struct {
	ID        string    `json:"eventId"`
	MediaID   string    `json:"mediaId"`
	ProjectID string    `json:"projectId"`
	SourceKey string    `json:"sourceKey"`
	Timestamp time.Time `json:"timestamp"`
}

func (e MediaSourceReady) EventID() string       { return e.ID }
func (e MediaSourceReady) EventType() string     { return EventTypeMediaSourceReady }
func (e MediaSourceReady) AggregateID() string   { return e.MediaID }
func (e MediaSourceReady) OccurredAt() time.Time { return e.Timestamp }

type MediaProbed struct {
	ID         string    `json:"eventId"`
	MediaID    string    `json:"mediaId"`
	ProjectID  string    `json:"projectId"`
	SourceKey  string    `json:"sourceKey"`
	HasAudio   bool      `json:"hasAudio"`
	DurationMs int64     `json:"durationMs"`
	Width      int       `json:"width,omitempty"`
	Height     int       `json:"height,omitempty"`
	SkipProxy  bool      `json:"skipProxy,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e MediaProbed) EventID() string       { return e.ID }
func (e MediaProbed) EventType() string     { return EventTypeMediaProbed }
func (e MediaProbed) AggregateID() string   { return e.MediaID }
func (e MediaProbed) OccurredAt() time.Time { return e.Timestamp }

type MediaAudioReady struct {
	ID               string    `json:"eventId"`
	MediaID          string    `json:"mediaId"`
	ProjectID        string    `json:"projectId"`
	AnalysisKey      string    `json:"analysisKey"`
	TranscriptionKey string    `json:"transcriptionKey"`
	MasterKey        string    `json:"masterKey"`
	Timestamp        time.Time `json:"timestamp"`
}

func (e MediaAudioReady) EventID() string       { return e.ID }
func (e MediaAudioReady) EventType() string     { return EventTypeMediaAudioReady }
func (e MediaAudioReady) AggregateID() string   { return e.MediaID }
func (e MediaAudioReady) OccurredAt() time.Time { return e.Timestamp }

type MediaProxyReady struct {
	ID        string    `json:"eventId"`
	MediaID   string    `json:"mediaId"`
	ProjectID string    `json:"projectId"`
	ProxyKey  string    `json:"proxyKey"`
	Timestamp time.Time `json:"timestamp"`
}

func (e MediaProxyReady) EventID() string       { return e.ID }
func (e MediaProxyReady) EventType() string     { return EventTypeMediaProxyReady }
func (e MediaProxyReady) AggregateID() string   { return e.MediaID }
func (e MediaProxyReady) OccurredAt() time.Time { return e.Timestamp }

type MediaWaveformReady struct {
	ID        string    `json:"eventId"`
	MediaID   string    `json:"mediaId"`
	ProjectID string    `json:"projectId"`
	PeaksKey  string    `json:"peaksKey"`
	Timestamp time.Time `json:"timestamp"`
}

func (e MediaWaveformReady) EventID() string       { return e.ID }
func (e MediaWaveformReady) EventType() string     { return EventTypeMediaWaveformReady }
func (e MediaWaveformReady) AggregateID() string   { return e.MediaID }
func (e MediaWaveformReady) OccurredAt() time.Time { return e.Timestamp }

type MediaReady struct {
	ID        string    `json:"eventId"`
	MediaID   string    `json:"mediaId"`
	ProjectID string    `json:"projectId"`
	Timestamp time.Time `json:"timestamp"`
}

func (e MediaReady) EventID() string       { return e.ID }
func (e MediaReady) EventType() string     { return EventTypeMediaReady }
func (e MediaReady) AggregateID() string   { return e.MediaID }
func (e MediaReady) OccurredAt() time.Time { return e.Timestamp }

type DetectionsUpdated struct {
	ID        string    `json:"eventId"`
	MediaID   string    `json:"mediaId"`
	ProjectID string    `json:"projectId"`
	Kind      string    `json:"kind"`
	Timestamp time.Time `json:"timestamp"`
}

func (e DetectionsUpdated) EventID() string       { return e.ID }
func (e DetectionsUpdated) EventType() string     { return EventTypeMediaDetectionsUpdated }
func (e DetectionsUpdated) AggregateID() string   { return e.MediaID }
func (e DetectionsUpdated) OccurredAt() time.Time { return e.Timestamp }

type MediaTranscribed struct {
	ID        string    `json:"eventId"`
	MediaID   string    `json:"mediaId"`
	ProjectID string    `json:"projectId"`
	WordsKey  string    `json:"wordsKey"`
	Language  string    `json:"language,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (e MediaTranscribed) EventID() string       { return e.ID }
func (e MediaTranscribed) EventType() string     { return EventTypeMediaTranscribed }
func (e MediaTranscribed) AggregateID() string   { return e.MediaID }
func (e MediaTranscribed) OccurredAt() time.Time { return e.Timestamp }
