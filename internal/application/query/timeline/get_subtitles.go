package timeline

import (
	"context"
	"errors"

	domainstyle "go-api/internal/domain/subtitlestyle"
	domaintimeline "go-api/internal/domain/timeline"
	domaintranscript "go-api/internal/domain/transcript"
	"go-api/internal/subtitles"
	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

type GetSubtitlesQuery struct {
	ProjectID uuid.UUID
	StyleID   *uuid.UUID
	Format    string // ass | srt
	// Optional PlayRes override (render burn-in uses the output preset size).
	PlayResX int
	PlayResY int
	// Optional SourceTime window for clip renders (rebased to output 0).
	ClipMediaID *uuid.UUID
	ClipStartMs *int64
	ClipEndMs   *int64
}

type GetSubtitlesHandler struct {
	timelines   domaintimeline.ReadRepository
	styles      domainstyle.ReadRepository
	transcripts domaintranscript.ReadRepository
}

func NewGetSubtitlesHandler(
	timelines domaintimeline.ReadRepository,
	styles domainstyle.ReadRepository,
	transcripts domaintranscript.ReadRepository,
) *GetSubtitlesHandler {
	return &GetSubtitlesHandler{
		timelines:   timelines,
		styles:      styles,
		transcripts: transcripts,
	}
}

func (h *GetSubtitlesHandler) Handle(ctx context.Context, q GetSubtitlesQuery) (string, string, error) {
	format := q.Format
	if format == "" {
		format = "ass"
	}
	if format != "ass" && format != "srt" {
		return "", "", errors.New("format must be ass or srt")
	}

	t, err := h.timelines.FindHeadByProject(ctx, q.ProjectID)
	if err != nil {
		return "", "", err
	}
	if t == nil {
		return "", "", errors.New("timeline not found")
	}
	segments, err := h.timelines.ListSegments(ctx, t.ID)
	if err != nil {
		return "", "", err
	}
	if len(segments) == 0 {
		return "", "", errors.New("timeline has no segments")
	}

	var style *domainstyle.Style
	if q.StyleID != nil {
		style, err = h.styles.FindByID(ctx, *q.StyleID)
		if err != nil {
			return "", "", err
		}
		if style == nil {
			return "", "", errors.New("subtitle style not found")
		}
		if style.ProjectID != nil && *style.ProjectID != q.ProjectID {
			return "", "", errors.New("subtitle style not found")
		}
	}

	engSegs := make([]eng.Segment, 0, len(segments))
	for _, s := range segments {
		engSegs = append(engSegs, eng.Segment{
			MediaFileID:   s.MediaFileID,
			SourceStartMs: s.SourceStartMs,
			SourceEndMs:   s.SourceEndMs,
			OutputStartMs: s.OutputStartMs,
			OutputEndMs:   s.OutputEndMs,
		})
	}
	if q.ClipMediaID != nil && q.ClipStartMs != nil && q.ClipEndMs != nil {
		// Match clip encode: continuous source window (not silence KEEP cuts).
		engSegs = eng.ContinuousClip(*q.ClipMediaID, *q.ClipStartMs, *q.ClipEndMs)
		if len(engSegs) == 0 {
			return "", "", errors.New("clip has no overlapping timeline segments")
		}
	}
	mediaSeen := make(map[uuid.UUID]struct{})
	mediaIDs := make([]uuid.UUID, 0)
	for _, s := range engSegs {
		if _, ok := mediaSeen[s.MediaFileID]; !ok {
			mediaSeen[s.MediaFileID] = struct{}{}
			mediaIDs = append(mediaIDs, s.MediaFileID)
		}
	}

	words := make([]subtitles.Word, 0)
	foundTranscript := false
	for _, mediaID := range mediaIDs {
		tr, err := h.transcripts.FindByMediaID(ctx, mediaID)
		if err != nil {
			return "", "", err
		}
		if tr == nil {
			continue
		}
		foundTranscript = true
		list, err := h.transcripts.ListWords(ctx, tr.ID)
		if err != nil {
			return "", "", err
		}
		for _, w := range list {
			words = append(words, subtitles.Word{
				MediaFileID:   mediaID,
				Text:          w.Text,
				SourceStartMs: w.SourceStartMs,
				SourceEndMs:   w.SourceEndMs,
			})
		}
	}
	if !foundTranscript {
		return "", "", errors.New("transcript not found")
	}

	timed := subtitles.RemapWords(words, engSegs)
	genStyle := subtitles.FromDomain(style)
	if q.PlayResX > 0 {
		genStyle.PlayResX = q.PlayResX
	}
	if q.PlayResY > 0 {
		genStyle.PlayResY = q.PlayResY
	}
	var body string
	var contentType string
	switch format {
	case "srt":
		body = subtitles.GenerateSRT(timed, genStyle)
		contentType = "application/x-subrip"
	default:
		body = subtitles.GenerateASS(timed, genStyle)
		contentType = "text/x-ssa"
	}
	return body, contentType, nil
}
