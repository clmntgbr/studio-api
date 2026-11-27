package timeline

import (
	"context"
	"errors"
	"fmt"

	domainmedia "go-api/internal/domain/media"
	domaintimeline "go-api/internal/domain/timeline"
	"go-api/internal/export"

	"github.com/google/uuid"
)

type GetExportQuery struct {
	MediaID uuid.UUID
	Format  string // fcpxml | edl | otio | premiere
}

type GetExportHandler struct {
	timelines domaintimeline.ReadRepository
	media     domainmedia.WriteRepository
}

func NewGetExportHandler(
	timelines domaintimeline.ReadRepository,
	media domainmedia.WriteRepository,
) *GetExportHandler {
	return &GetExportHandler{timelines: timelines, media: media}
}

func (h *GetExportHandler) Handle(ctx context.Context, q GetExportQuery) (body string, contentType string, filename string, err error) {
	format := q.Format
	switch format {
	case "fcpxml", "edl", "otio", "premiere":
	default:
		return "", "", "", errors.New("format must be fcpxml, edl, otio or premiere")
	}

	t, err := h.timelines.FindHeadByMedia(ctx, q.MediaID)
	if err != nil {
		return "", "", "", err
	}
	if t == nil {
		return "", "", "", errors.New("timeline not found")
	}
	if t.Status != domaintimeline.StatusValidated && t.Status != domaintimeline.StatusDraft {
		return "", "", "", errors.New("timeline not found")
	}
	segments, err := h.timelines.ListSegments(ctx, t.ID)
	if err != nil {
		return "", "", "", err
	}
	if len(segments) == 0 {
		return "", "", "", errors.New("timeline has no segments")
	}

	projectName := "Studio"
	if m, err := h.media.GetByID(ctx, q.MediaID); err == nil && m != nil {
		if m.Title != nil && *m.Title != "" {
			projectName = *m.Title
		} else if m.OriginalFilename != nil && *m.OriginalFilename != "" {
			projectName = *m.OriginalFilename
		}
	}

	in := export.TimelineInput{
		ProjectName: projectName,
		Version:     t.Version,
		DurationMs:  t.DurationMs,
		Width:       1920,
		Height:      1080,
		FPS:         30,
		Segments:    make([]export.Segment, 0, len(segments)),
	}

	mediaCache := map[uuid.UUID]*domainmedia.MediaFile{}
	for i, s := range segments {
		m := mediaCache[s.MediaFileID]
		if m == nil {
			m, err = h.media.GetByID(ctx, s.MediaFileID)
			if err != nil {
				return "", "", "", err
			}
			mediaCache[s.MediaFileID] = m
		}
		name := s.MediaFileID.String()
		path := "file://localhost/media/" + s.MediaFileID.String() + "/source/original.mp4"
		if m != nil {
			if m.OriginalFilename != nil && *m.OriginalFilename != "" {
				name = *m.OriginalFilename
			}
			if m.StorageKey != nil && *m.StorageKey != "" {
				path = "file://localhost/" + *m.StorageKey
			}
			if m.Width != nil && *m.Width > 0 {
				in.Width = *m.Width
			}
			if m.Height != nil && *m.Height > 0 {
				in.Height = *m.Height
			}
			if m.FPS != nil && *m.FPS > 0 {
				in.FPS = *m.FPS
			}
		}
		in.Segments = append(in.Segments, export.Segment{
			Index:         i,
			MediaFileID:   s.MediaFileID,
			MediaName:     name,
			MediaPath:     path,
			SourceStartMs: s.SourceStartMs,
			SourceEndMs:   s.SourceEndMs,
			OutputStartMs: s.OutputStartMs,
			OutputEndMs:   s.OutputEndMs,
			FPS:           in.FPS,
		})
	}

	switch format {
	case "edl":
		return export.ToEDL(in), "application/x-edl", "timeline.edl", nil
	case "otio":
		doc, err := export.ToOTIO(in)
		if err != nil {
			return "", "", "", fmt.Errorf("otio: %w", err)
		}
		return doc, "application/otio+json", "timeline.otio", nil
	case "premiere":
		return export.ToPremiereXML(in), "application/xml", "timeline.premiere.xml", nil
	default:
		return export.ToFCPXML(in), "application/xml", "timeline.fcpxml", nil
	}
}