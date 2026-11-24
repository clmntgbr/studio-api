package presenter

import (
	"time"

	cmdmedia "go-api/internal/application/command/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
)

type MediaResponse struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	Position         int       `json:"position"`
	SourceKind       string    `json:"sourceKind"`
	SourceURL        *string   `json:"sourceUrl,omitempty"`
	Title            *string   `json:"title,omitempty"`
	OriginalFilename *string   `json:"originalFilename,omitempty"`
	StorageKey       *string   `json:"storageKey,omitempty"`
	MimeType         *string   `json:"mimeType,omitempty"`
	DurationMs       *int64    `json:"durationMs,omitempty"`
	Width            *int      `json:"width,omitempty"`
	Height           *int      `json:"height,omitempty"`
	Status           string    `json:"status"`
	SizeBytes        *int64    `json:"sizeBytes,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func NewMediaResponseFromView(v domainmedia.FileView) MediaResponse {
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
