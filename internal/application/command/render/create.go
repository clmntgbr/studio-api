package render

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	querytimeline "go-api/internal/application/query/timeline"
	domainclip "go-api/internal/domain/clip"
	domainmedia "go-api/internal/domain/media"
	domainpreset "go-api/internal/domain/outputpreset"
	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"
	domaintimeline "go-api/internal/domain/timeline"
	"go-api/internal/renderplan"
	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

type CreateCommand struct {
	ProjectID       uuid.UUID
	PresetID        uuid.UUID
	Kind            string
	SubtitleStyleID *uuid.UUID
	ClipID          *uuid.UUID
}

type assBuilder interface {
	Handle(ctx context.Context, q querytimeline.GetSubtitlesQuery) (string, string, error)
}

type CreateHandler struct {
	renders      domainrender.WriteRepository
	timelines    domaintimeline.ReadRepository
	presets      domainpreset.ReadRepository
	media        domainmedia.WriteRepository
	clips        domainclip.ReadRepository
	jobs         port.JobPublisher
	storage      port.ObjectStorage
	ass          assBuilder
	renderBucket string
}

func NewCreateHandler(
	renders domainrender.WriteRepository,
	timelines domaintimeline.ReadRepository,
	presets domainpreset.ReadRepository,
	media domainmedia.WriteRepository,
	clips domainclip.ReadRepository,
	jobs port.JobPublisher,
	storage port.ObjectStorage,
	ass assBuilder,
	renderBucket string,
) *CreateHandler {
	return &CreateHandler{
		renders:      renders,
		timelines:    timelines,
		presets:      presets,
		media:        media,
		clips:        clips,
		jobs:         jobs,
		storage:      storage,
		ass:          ass,
		renderBucket: renderBucket,
	}
}

func (h *CreateHandler) Handle(ctx context.Context, cmd CreateCommand) (*domainrender.Render, error) {
	kind := domainrender.Kind(cmd.Kind)
	if kind == "" {
		if cmd.ClipID != nil {
			kind = domainrender.KindClip
		} else {
			kind = domainrender.KindMaster
		}
	}
	switch kind {
	case domainrender.KindMaster, domainrender.KindClip, domainrender.KindCustom:
	case domainrender.KindAudio:
		return nil, errors.New("kind must be master, clip or custom (audio not supported yet)")
	default:
		return nil, errors.New("kind must be master, clip, audio or custom")
	}
	if kind == domainrender.KindClip && cmd.ClipID == nil {
		return nil, errors.New("clipId is required for kind clip")
	}

	t, err := h.timelines.FindHeadByProject(ctx, cmd.ProjectID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("timeline not found")
	}
	segments, err := h.timelines.ListSegments(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	if len(segments) == 0 {
		return nil, errors.New("timeline has no segments")
	}

	preset, err := h.presets.FindByID(ctx, cmd.PresetID)
	if err != nil {
		return nil, err
	}
	if preset == nil {
		return nil, errors.New("output preset not found")
	}

	var clip *domainclip.Clip
	if cmd.ClipID != nil {
		if h.clips == nil {
			return nil, errors.New("clip repository is not configured")
		}
		clip, err = h.clips.FindByID(ctx, *cmd.ClipID)
		if err != nil {
			return nil, err
		}
		if clip == nil || clip.ProjectID != cmd.ProjectID {
			return nil, errors.New("clip not found")
		}
	}

	fp := renderplan.Fingerprint(t.ID, preset.ID, cmd.SubtitleStyleID, cmd.ClipID, string(kind), preset.CropMode)
	if existing, err := h.renders.FindByFingerprint(ctx, cmd.ProjectID, fp); err != nil {
		return nil, err
	} else if existing != nil {
		switch existing.Status {
		case domainrender.StatusQueued, domainrender.StatusRunning, domainrender.StatusCompleted, domainrender.StatusPending:
			return existing, nil
		}
		existing.Fingerprint = existing.Fingerprint + ":superseded:" + existing.ID.String()
		if err := h.renders.Update(ctx, existing); err != nil {
			return nil, errors.New("failed to supersede previous render")
		}
	}

	workSegs := segments
	if clip != nil {
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
		window := eng.ClipWindow(engSegs, clip.MediaFileID, clip.SourceStartMs, clip.SourceEndMs)
		if len(window) == 0 {
			return nil, errors.New("clip has no overlapping timeline segments")
		}
		workSegs = make([]domaintimeline.Segment, 0, len(window))
		for _, s := range window {
			workSegs = append(workSegs, domaintimeline.Segment{
				MediaFileID:   s.MediaFileID,
				SourceStartMs: s.SourceStartMs,
				SourceEndMs:   s.SourceEndMs,
				OutputStartMs: s.OutputStartMs,
				OutputEndMs:   s.OutputEndMs,
			})
		}
	}

	planSegs := make([]renderplan.Segment, 0, len(workSegs))
	var cropPathKey *string
	for _, s := range workSegs {
		m, err := h.media.GetByID(ctx, s.MediaFileID)
		if err != nil {
			return nil, err
		}
		if m == nil || m.StorageKey == nil || *m.StorageKey == "" {
			return nil, errors.New("media source not ready")
		}
		planSegs = append(planSegs, renderplan.Segment{
			MediaKey:   *m.StorageKey,
			SrcStartMs: s.SourceStartMs,
			SrcEndMs:   s.SourceEndMs,
		})
		if cropPathKey == nil && (preset.CropMode == "face_track" || preset.CropMode == "speaker_track") {
			if asset, err := h.media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindCropPath); err == nil && asset != nil && asset.StorageKey != "" {
				key := asset.StorageKey
				cropPathKey = &key
			}
		}
	}

	crf, ffmpegPreset, fadeMs := 20, "veryfast", 15
	hookMs := int64(3000)
	if len(preset.Settings) > 0 {
		var settings map[string]any
		_ = json.Unmarshal(preset.Settings, &settings)
		if v, ok := settings["crf"].(float64); ok {
			crf = int(v)
		}
		if v, ok := settings["preset"].(string); ok && v != "" {
			ffmpegPreset = v
		}
		if v, ok := settings["fadeMs"].(float64); ok {
			fadeMs = int(v)
		}
		if v, ok := settings["hookMs"].(float64); ok && v > 0 {
			hookMs = int64(v)
		}
	}
	fps := 30.0
	if preset.FPS != nil && *preset.FPS > 0 {
		fps = *preset.FPS
	}
	loudness := -14.0
	if preset.LoudnessLUFS != nil {
		loudness = *preset.LoudnessLUFS
	}

	renderID := uuid.New()
	outputKey := fmt.Sprintf("projects/%s/renders/%s/out.mp4", cmd.ProjectID.String(), renderID.String())

	plan := renderplan.Plan{
		RenderID: renderID.String(),
		Kind:     string(kind),
		Segments: planSegs,
		Crop:     renderplan.Crop{Mode: preset.CropMode, PathKey: cropPathKey},
		Output: renderplan.Output{
			Width:  preset.Width,
			Height: preset.Height,
			FPS:    fps,
			Codec:  preset.VideoCodec,
			CRF:    crf,
			Preset: ffmpegPreset,
		},
		Audio: renderplan.Audio{
			Codec:        preset.AudioCodec,
			LoudnessLUFS: loudness,
			FadeMs:       fadeMs,
		},
		OutputKey: outputKey,
	}

	if clip != nil && clip.Hook != nil && *clip.Hook != "" {
		end := hookMs
		if end <= 0 {
			end = 3000
		}
		plan.Overlays = append(plan.Overlays, renderplan.Overlay{
			Kind:    "headline",
			Text:    *clip.Hook,
			StartMs: 0,
			EndMs:   end,
		})
	}

	if cmd.SubtitleStyleID != nil && kind != domainrender.KindAudio {
		if h.ass == nil || h.storage == nil {
			return nil, errors.New("subtitle burn-in is not configured")
		}
		q := querytimeline.GetSubtitlesQuery{
			ProjectID: cmd.ProjectID,
			StyleID:   cmd.SubtitleStyleID,
			Format:    "ass",
			PlayResX:  preset.Width,
			PlayResY:  preset.Height,
		}
		if clip != nil {
			mid := clip.MediaFileID
			start, end := clip.SourceStartMs, clip.SourceEndMs
			q.ClipMediaID = &mid
			q.ClipStartMs = &start
			q.ClipEndMs = &end
		}
		body, contentType, err := h.ass.Handle(ctx, q)
		if err != nil {
			return nil, err
		}
		assKey := fmt.Sprintf("projects/%s/renders/%s/subs.ass", cmd.ProjectID.String(), renderID.String())
		if err := h.storage.PutObject(ctx, h.renderBucket, assKey, []byte(body), contentType); err != nil {
			return nil, errors.New("failed to upload subtitle track")
		}
		plan.Subtitles = &renderplan.Subtitles{ASSKey: assKey, FontsDir: "/fonts"}
	}

	rawPlan, err := plan.Marshal()
	if err != nil {
		return nil, err
	}

	r := domainrender.New(cmd.ProjectID, t.ID, preset.ID, kind, fp, rawPlan)
	r.ID = renderID
	r.SubtitleStyleID = cmd.SubtitleStyleID
	r.ClipID = cmd.ClipID
	r.Status = domainrender.StatusQueued
	now := time.Now().UTC()
	r.StartedAt = nil
	_ = now

	if err := h.renders.Save(ctx, r); err != nil {
		return nil, errors.New("failed to create render")
	}
	if err := h.jobs.PublishRenderJob(ctx, r.ID.String(), rawPlan); err != nil {
		r.Status = domainrender.StatusFailed
		msg := "failed to enqueue render"
		r.ErrorMessage = &msg
		_ = h.renders.Update(ctx, r)
		return nil, errors.New("failed to enqueue render")
	}
	return r, nil
}
