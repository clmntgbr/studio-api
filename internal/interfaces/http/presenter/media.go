package presenter

import (
	"time"

	cmdmedia "go-api/internal/application/command/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
)

type MediaStepResponse struct {
	Kind         string     `json:"kind"`
	Status       string     `json:"status"`
	Progress     float64    `json:"progress"`
	ErrorMessage *string    `json:"errorMessage,omitempty"`
	QueuedAt     *time.Time `json:"queuedAt,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	DurationMs   *int64     `json:"durationMs,omitempty"`
}

type MediaStepRunResponse struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"`
	Attempt      int        `json:"attempt"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	DurationMs   *int64     `json:"durationMs,omitempty"`
	ErrorMessage *string    `json:"errorMessage,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type MediaStepsResponse struct {
	Steps []MediaStepResponse    `json:"steps"`
	Runs  []MediaStepRunResponse `json:"runs"`
}

type MediaResponse struct {
	ID               string              `json:"id"`
	ProjectID        string              `json:"projectId"`
	Position         int                 `json:"position"`
	SourceKind       string              `json:"sourceKind"`
	SourceURL        *string             `json:"sourceUrl,omitempty"`
	Title            *string             `json:"title,omitempty"`
	OriginalFilename *string             `json:"originalFilename,omitempty"`
	StorageKey       *string             `json:"storageKey,omitempty"`
	MimeType         *string             `json:"mimeType,omitempty"`
	DurationMs       *int64              `json:"durationMs,omitempty"`
	Width            *int                `json:"width,omitempty"`
	Height           *int                `json:"height,omitempty"`
	Status           string              `json:"status"`
	SizeBytes        *int64              `json:"sizeBytes,omitempty"`
	Steps            []MediaStepResponse `json:"steps"`
	CreatedAt        time.Time           `json:"createdAt"`
	UpdatedAt        time.Time           `json:"updatedAt"`
}

func NewMediaResponseFromView(v domainmedia.FileView) MediaResponse {
	steps := make([]MediaStepResponse, 0, len(v.Steps))
	for _, s := range v.Steps {
		steps = append(steps, NewMediaStepResponse(s))
	}
	return MediaResponse{
		ID:               v.ID.String(),
		ProjectID:        v.ProjectID.String(),
		Position:         v.Position,
		SourceKind:       string(v.SourceKind),
		SourceURL:        v.SourceURL,
		Title:            v.Title,
		OriginalFilename: v.OriginalFilename,
		StorageKey:       v.StorageKey,
		MimeType:         v.MimeType,
		DurationMs:       v.DurationMs,
		Width:            v.Width,
		Height:           v.Height,
		Status:           string(v.Status),
		SizeBytes:        v.SizeBytes,
		Steps:            steps,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
	}
}

func NewMediaResponseFromEntity(m *domainmedia.MediaFile) MediaResponse {
	return MediaResponse{
		ID:               m.ID.String(),
		ProjectID:        m.ProjectID.String(),
		Position:         m.Position,
		SourceKind:       string(m.SourceKind),
		SourceURL:        m.SourceURL,
		Title:            m.Title,
		OriginalFilename: m.OriginalFilename,
		StorageKey:       m.StorageKey,
		MimeType:         m.MimeType,
		DurationMs:       m.DurationMs,
		Width:            m.Width,
		Height:           m.Height,
		Status:           string(m.Status),
		SizeBytes:        m.SizeBytes,
		Steps:            make([]MediaStepResponse, 0),
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func NewMediaListResponseFromViews(views []domainmedia.FileView) []MediaResponse {
	out := make([]MediaResponse, 0, len(views))
	for _, v := range views {
		out = append(out, NewMediaResponseFromView(v))
	}
	return out
}

func NewMediaStepResponse(s domainmedia.StepView) MediaStepResponse {
	return MediaStepResponse{
		Kind:         string(s.Kind),
		Status:       string(s.Status),
		Progress:     s.Progress,
		ErrorMessage: s.ErrorMessage,
		QueuedAt:     s.QueuedAt,
		StartedAt:    s.StartedAt,
		CompletedAt:  s.CompletedAt,
		DurationMs:   s.DurationMs,
	}
}

func NewMediaStepsResponse(steps []domainmedia.StepView, runs []domainmedia.StepRunView) MediaStepsResponse {
	stepOut := make([]MediaStepResponse, 0, len(steps))
	for _, s := range steps {
		stepOut = append(stepOut, NewMediaStepResponse(s))
	}
	runOut := make([]MediaStepRunResponse, 0, len(runs))
	for _, r := range runs {
		runOut = append(runOut, MediaStepRunResponse{
			ID:           r.ID.String(),
			Kind:         string(r.Kind),
			Attempt:      r.Attempt,
			Status:       string(r.Status),
			StartedAt:    r.StartedAt,
			CompletedAt:  r.CompletedAt,
			DurationMs:   r.DurationMs,
			ErrorMessage: r.ErrorMessage,
			CreatedAt:    r.CreatedAt,
		})
	}
	return MediaStepsResponse{Steps: stepOut, Runs: runOut}
}

type UploadPartResponse struct {
	PartNumber int32  `json:"partNumber"`
	URL        string `json:"url"`
}

type InitiateUploadResponse struct {
	MediaID  string               `json:"mediaId"`
	UploadID string               `json:"uploadId"`
	Bucket   string               `json:"bucket"`
	Key      string               `json:"key"`
	Parts    []UploadPartResponse `json:"parts"`
}

func NewInitiateUploadResponse(r *cmdmedia.InitiateUploadResult) InitiateUploadResponse {
	parts := make([]UploadPartResponse, 0, len(r.Parts))
	for _, p := range r.Parts {
		parts = append(parts, UploadPartResponse{PartNumber: p.PartNumber, URL: p.URL})
	}
	return InitiateUploadResponse{
		MediaID:  r.MediaID.String(),
		UploadID: r.UploadID,
		Bucket:   r.Bucket,
		Key:      r.Key,
		Parts:    parts,
	}
}

func NewUploadPartsResponse(parts []port.UploadPartURL) []UploadPartResponse {
	out := make([]UploadPartResponse, 0, len(parts))
	for _, p := range parts {
		out = append(out, UploadPartResponse{PartNumber: p.PartNumber, URL: p.URL})
	}
	return out
}

type MediaConfigurationResponse struct {
	MediaFileID          string   `json:"mediaFileId"`
	SilenceEnabled       bool     `json:"silenceEnabled"`
	ThresholdMode        string   `json:"thresholdMode"`
	SilenceThresholdDB   *float64 `json:"silenceThresholdDb,omitempty"`
	DetectionLevel       string   `json:"detectionLevel"`
	SilenceMinDurationMs int      `json:"silenceMinDurationMs"`
	PaddingBeforeMs      int      `json:"paddingBeforeMs"`
	PaddingAfterMs       int      `json:"paddingAfterMs"`
	SpeechMinDurationMs  int      `json:"speechMinDurationMs"`
	FillersEnabled       bool     `json:"fillersEnabled"`
	RepetitionsEnabled   bool     `json:"repetitionsEnabled"`
	FalseStartsEnabled   bool     `json:"falseStartsEnabled"`
	SponsorsAction       string    `json:"sponsorsAction"`
	AdsAction            string    `json:"adsAction"`
	Language             *string   `json:"language,omitempty"`
	MasterSplit          string    `json:"masterSplit"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

func NewMediaConfigurationResponse(cfg *domainmedia.Configuration) MediaConfigurationResponse {
	return MediaConfigurationResponse{
		MediaFileID:          cfg.MediaFileID.String(),
		SilenceEnabled:       cfg.SilenceEnabled,
		ThresholdMode:        cfg.ThresholdMode,
		SilenceThresholdDB:   cfg.SilenceThresholdDB,
		DetectionLevel:       cfg.DetectionLevel,
		SilenceMinDurationMs: cfg.SilenceMinDurationMs,
		PaddingBeforeMs:      cfg.PaddingBeforeMs,
		PaddingAfterMs:       cfg.PaddingAfterMs,
		SpeechMinDurationMs:  cfg.SpeechMinDurationMs,
		FillersEnabled:       cfg.FillersEnabled,
		RepetitionsEnabled:   cfg.RepetitionsEnabled,
		FalseStartsEnabled:   cfg.FalseStartsEnabled,
		SponsorsAction:       string(cfg.SponsorsAction),
		AdsAction:            string(cfg.AdsAction),
		Language:             cfg.Language,
		MasterSplit:          cfg.MasterSplit,
		UpdatedAt:            cfg.UpdatedAt,
	}
}
