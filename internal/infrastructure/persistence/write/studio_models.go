package write

import (
	"time"

	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
)

type ProjectModel struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (ProjectModel) TableName() string { return "project" }

type MediaFileModel struct {
	ID               uuid.UUID `gorm:"column:id;primaryKey"`
	ProjectID        uuid.UUID `gorm:"column:project_id"`
	Position         int       `gorm:"column:position"`
	SourceKind       string    `gorm:"column:source_kind"`
	SourceURL        *string   `gorm:"column:source_url"`
	Title            *string   `gorm:"column:title"`
	OriginalFilename *string   `gorm:"column:original_filename"`
	StorageKey       *string   `gorm:"column:storage_key"`
	MimeType         *string   `gorm:"column:mime_type"`
	DurationMs       *int64    `gorm:"column:duration_ms"`
	Width            *int      `gorm:"column:width"`
	Height           *int      `gorm:"column:height"`
	FPS              *float64  `gorm:"column:fps"`
	VideoCodec       *string   `gorm:"column:video_codec"`
	AudioCodec       *string   `gorm:"column:audio_codec"`
	HasAudio         *bool     `gorm:"column:has_audio"`
	SizeBytes        *int64    `gorm:"column:size_bytes"`
	Status           string    `gorm:"column:status"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (MediaFileModel) TableName() string { return "media_file" }

type MediaAssetModel struct {
	ID          uuid.UUID    `gorm:"column:id;primaryKey"`
	MediaFileID uuid.UUID    `gorm:"column:media_file_id"`
	Kind        string       `gorm:"column:kind"`
	StorageKey  string       `gorm:"column:storage_key"`
	SizeBytes   *int64       `gorm:"column:size_bytes"`
	Meta        dbtype.JSONB `gorm:"column:meta"`
	CreatedAt   time.Time    `gorm:"column:created_at"`
}

func (MediaAssetModel) TableName() string { return "media_asset" }

type MediaConfigurationModel struct {
	MediaFileID          uuid.UUID    `gorm:"column:media_file_id;primaryKey"`
	SilenceEnabled       bool         `gorm:"column:silence_enabled"`
	ThresholdMode        string       `gorm:"column:threshold_mode"`
	SilenceThresholdDB   *float64     `gorm:"column:silence_threshold_db"`
	DetectionLevel       string       `gorm:"column:detection_level"`
	SilenceMinDurationMs int          `gorm:"column:silence_min_duration_ms"`
	PaddingBeforeMs      int          `gorm:"column:padding_before_ms"`
	PaddingAfterMs       int          `gorm:"column:padding_after_ms"`
	SpeechMinDurationMs  int          `gorm:"column:speech_min_duration_ms"`
	FillersEnabled       bool         `gorm:"column:fillers_enabled"`
	RepetitionsEnabled   bool         `gorm:"column:repetitions_enabled"`
	FalseStartsEnabled   bool         `gorm:"column:false_starts_enabled"`
	SponsorsAction       string       `gorm:"column:sponsors_action"`
	AdsAction            string       `gorm:"column:ads_action"`
	Language             *string      `gorm:"column:language"`
	Settings             dbtype.JSONB `gorm:"column:settings"`
	UpdatedAt            time.Time    `gorm:"column:updated_at"`
}

func (MediaConfigurationModel) TableName() string { return "media_configuration" }

type ProcessingStepModel struct {
	ID           uuid.UUID  `gorm:"column:id;primaryKey"`
	MediaFileID  uuid.UUID  `gorm:"column:media_file_id"`
	Kind         string     `gorm:"column:kind"`
	Status       string     `gorm:"column:status"`
	Attempt      int        `gorm:"column:attempt"`
	InputHash    *string    `gorm:"column:input_hash"`
	Progress     float64    `gorm:"column:progress"`
	ErrorMessage *string    `gorm:"column:error_message"`
	QueuedAt     *time.Time `gorm:"column:queued_at"`
	StartedAt    *time.Time `gorm:"column:started_at"`
	CompletedAt  *time.Time `gorm:"column:completed_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

func (ProcessingStepModel) TableName() string { return "processing_step" }

type ProcessingStepRunModel struct {
	ID           uuid.UUID  `gorm:"column:id;primaryKey"`
	MediaFileID  uuid.UUID  `gorm:"column:media_file_id"`
	Kind         string     `gorm:"column:kind"`
	Attempt      int        `gorm:"column:attempt"`
	Status       string     `gorm:"column:status"`
	StartedAt    time.Time  `gorm:"column:started_at"`
	CompletedAt  *time.Time `gorm:"column:completed_at"`
	DurationMs   *int64     `gorm:"column:duration_ms"`
	ErrorMessage *string    `gorm:"column:error_message"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

func (ProcessingStepRunModel) TableName() string { return "processing_step_run" }

type TranscriptModel struct {
	ID            uuid.UUID `gorm:"column:id;primaryKey"`
	MediaFileID   uuid.UUID `gorm:"column:media_file_id"`
	Language      *string   `gorm:"column:language"`
	Text          *string   `gorm:"column:text"`
	Provider      string    `gorm:"column:provider"`
	ProviderJobID *string   `gorm:"column:provider_job_id"`
	Status        string    `gorm:"column:status"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (TranscriptModel) TableName() string { return "transcript" }

type TranscriptWordModel struct {
	ID            int64     `gorm:"column:id;primaryKey"`
	TranscriptID  uuid.UUID `gorm:"column:transcript_id"`
	WordIndex     int       `gorm:"column:word_index"`
	Text          string    `gorm:"column:text"`
	SourceStartMs int64     `gorm:"column:source_start_ms"`
	SourceEndMs   int64     `gorm:"column:source_end_ms"`
	Confidence    *float64  `gorm:"column:confidence"`
	Speaker       *string   `gorm:"column:speaker"`
}

func (TranscriptWordModel) TableName() string { return "transcript_word" }

type DetectionModel struct {
	ID             uuid.UUID    `gorm:"column:id;primaryKey"`
	MediaFileID    uuid.UUID    `gorm:"column:media_file_id"`
	Kind           string       `gorm:"column:kind"`
	SourceStartMs  int64        `gorm:"column:source_start_ms"`
	SourceEndMs    int64        `gorm:"column:source_end_ms"`
	Confidence     *float64     `gorm:"column:confidence"`
	Detector       string       `gorm:"column:detector"`
	FirstWordIndex *int         `gorm:"column:first_word_index"`
	LastWordIndex  *int         `gorm:"column:last_word_index"`
	Payload        dbtype.JSONB `gorm:"column:payload"`
	CreatedAt      time.Time    `gorm:"column:created_at"`
}

func (DetectionModel) TableName() string { return "detection" }

type UserOverrideModel struct {
	ID            uuid.UUID `gorm:"column:id;primaryKey"`
	MediaFileID   uuid.UUID `gorm:"column:media_file_id"`
	Kind          *string   `gorm:"column:kind"`
	Action        string    `gorm:"column:action"`
	SourceStartMs int64     `gorm:"column:source_start_ms"`
	SourceEndMs   int64     `gorm:"column:source_end_ms"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (UserOverrideModel) TableName() string { return "user_override" }

type TimelineModel struct {
	ID          uuid.UUID  `gorm:"column:id;primaryKey"`
	ProjectID   uuid.UUID  `gorm:"column:project_id"`
	Version     int        `gorm:"column:version"`
	Status      string     `gorm:"column:status"`
	Fingerprint string     `gorm:"column:fingerprint"`
	DurationMs  int64      `gorm:"column:duration_ms"`
	ValidatedAt *time.Time `gorm:"column:validated_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func (TimelineModel) TableName() string { return "timeline" }

type TimelineSegmentModel struct {
	ID                  uuid.UUID `gorm:"column:id;primaryKey"`
	TimelineID          uuid.UUID `gorm:"column:timeline_id"`
	SegmentIndex        int       `gorm:"column:segment_index"`
	MediaFileID         uuid.UUID `gorm:"column:media_file_id"`
	SourceStartMs       int64     `gorm:"column:source_start_ms"`
	SourceEndMs         int64     `gorm:"column:source_end_ms"`
	OutputStartMs       int64     `gorm:"column:output_start_ms"`
	OutputEndMs         int64     `gorm:"column:output_end_ms"`
	RemovedBeforeReason *string   `gorm:"column:removed_before_reason"`
}

func (TimelineSegmentModel) TableName() string { return "timeline_segment" }

type TimelineDecisionModel struct {
	TimelineID  uuid.UUID `gorm:"column:timeline_id;primaryKey"`
	DetectionID uuid.UUID `gorm:"column:detection_id;primaryKey"`
	Action      string    `gorm:"column:action"`
	Origin      string    `gorm:"column:origin"`
	Reason      *string   `gorm:"column:reason"`
}

func (TimelineDecisionModel) TableName() string { return "timeline_decision" }

type ClipModel struct {
	ID            uuid.UUID          `gorm:"column:id;primaryKey"`
	ProjectID     uuid.UUID          `gorm:"column:project_id"`
	MediaFileID   uuid.UUID          `gorm:"column:media_file_id"`
	SourceStartMs int64              `gorm:"column:source_start_ms"`
	SourceEndMs   int64              `gorm:"column:source_end_ms"`
	Origin        string             `gorm:"column:origin"`
	Score         *float64           `gorm:"column:score"`
	Hook          *string            `gorm:"column:hook"`
	Reason        *string            `gorm:"column:reason"`
	Title         *string            `gorm:"column:title"`
	Description   *string            `gorm:"column:description"`
	Hashtags      dbtype.StringArray `gorm:"column:hashtags"`
	Selected      bool               `gorm:"column:selected"`
	CreatedAt     time.Time          `gorm:"column:created_at"`
}

func (ClipModel) TableName() string { return "clip" }

type LLMArtifactModel struct {
	ID          uuid.UUID    `gorm:"column:id;primaryKey"`
	MediaFileID uuid.UUID    `gorm:"column:media_file_id"`
	Kind        string       `gorm:"column:kind"`
	Payload     dbtype.JSONB `gorm:"column:payload"`
	Provider    *string      `gorm:"column:provider"`
	CreatedAt   time.Time    `gorm:"column:created_at"`
}

func (LLMArtifactModel) TableName() string { return "llm_artifact" }

type SubtitleStyleModel struct {
	ID             uuid.UUID    `gorm:"column:id;primaryKey"`
	ProjectID      *uuid.UUID   `gorm:"column:project_id"`
	Name           string       `gorm:"column:name"`
	FontFamily     *string      `gorm:"column:font_family"`
	FontSize       *int         `gorm:"column:font_size"`
	BaseColor      *string      `gorm:"column:base_color"`
	HighlightColor *string      `gorm:"column:highlight_color"`
	OutlineColor   *string      `gorm:"column:outline_color"`
	OutlineSize    *int         `gorm:"column:outline_size"`
	WordsPerLine   *int16       `gorm:"column:words_per_line"`
	Position       *string      `gorm:"column:position"`
	Uppercase      bool         `gorm:"column:uppercase"`
	Settings       dbtype.JSONB `gorm:"column:settings"`
	CreatedAt      time.Time    `gorm:"column:created_at"`
}

func (SubtitleStyleModel) TableName() string { return "subtitle_style" }

type OverlayModel struct {
	ID         uuid.UUID    `gorm:"column:id;primaryKey"`
	ProjectID  uuid.UUID    `gorm:"column:project_id"`
	ClipID     *uuid.UUID   `gorm:"column:clip_id"`
	Kind       string       `gorm:"column:kind"`
	Text       *string      `gorm:"column:text"`
	StorageKey *string      `gorm:"column:storage_key"`
	StartMs    int64        `gorm:"column:start_ms"`
	EndMs      int64        `gorm:"column:end_ms"`
	Position   *string      `gorm:"column:position"`
	Settings   dbtype.JSONB `gorm:"column:settings"`
}

func (OverlayModel) TableName() string { return "overlay" }

type OutputPresetModel struct {
	ID           uuid.UUID    `gorm:"column:id;primaryKey"`
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

func (OutputPresetModel) TableName() string { return "output_preset" }

type RenderModel struct {
	ID              uuid.UUID    `gorm:"column:id;primaryKey"`
	ProjectID       uuid.UUID    `gorm:"column:project_id"`
	TimelineID      uuid.UUID    `gorm:"column:timeline_id"`
	Kind            string       `gorm:"column:kind"`
	Strategy        string       `gorm:"column:strategy"`
	ClipID          *uuid.UUID   `gorm:"column:clip_id"`
	OutputPresetID  uuid.UUID    `gorm:"column:output_preset_id"`
	SubtitleStyleID *uuid.UUID   `gorm:"column:subtitle_style_id"`
	Plan            dbtype.JSONB `gorm:"column:plan"`
	Fingerprint     string       `gorm:"column:fingerprint"`
	Status          string       `gorm:"column:status"`
	Progress        float64      `gorm:"column:progress"`
	StorageKey      *string      `gorm:"column:storage_key"`
	DurationMs      *int64       `gorm:"column:duration_ms"`
	SizeBytes       *int64       `gorm:"column:size_bytes"`
	ErrorMessage    *string      `gorm:"column:error_message"`
	CreatedAt       time.Time    `gorm:"column:created_at"`
	StartedAt       *time.Time   `gorm:"column:started_at"`
	CompletedAt     *time.Time   `gorm:"column:completed_at"`
}

func (RenderModel) TableName() string { return "render" }
