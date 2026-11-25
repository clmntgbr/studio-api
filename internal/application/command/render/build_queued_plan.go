package render

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

type queuedPlanDeps struct {
	timelines    domaintimeline.ReadRepository
	presets      domainpreset.ReadRepository
	media        domainmedia.WriteRepository
	clips        domainclip.ReadRepository
	ass          assBuilder
	storage      port.ObjectStorage
	renderBucket string
}

// QueuedPlanDeps wires dependencies for building a queued render plan (create + kickoff).
func QueuedPlanDeps(
	timelines domaintimeline.ReadRepository,
	presets domainpreset.ReadRepository,
	media domainmedia.WriteRepository,
	clips domainclip.ReadRepository,
	ass assBuilder,
	storage port.ObjectStorage,
	renderBucket string,
) queuedPlanDeps {
	return queuedPlanDeps{
		timelines:    timelines,
		presets:      presets,
		media:        media,
		clips:        clips,
		ass:          ass,
		storage:      storage,
		renderBucket: renderBucket,
	}
}

type queuedPlanRequest struct {
	ProjectID       uuid.UUID
	RenderID        uuid.UUID
	TimelineID      uuid.UUID
	PresetID        uuid.UUID
	Kind            domainrender.Kind
	SubtitleStyleID *uuid.UUID
	ClipID          *uuid.UUID
}

func buildQueuedPlan(ctx context.Context, d queuedPlanDeps, req queuedPlanRequest) (json.RawMessage, error) {
	segments, err := d.timelines.ListSegments(ctx, req.TimelineID)
	if err != nil {
		return nil, err
	}
	if len(segments) == 0 {
		return nil, errors.New("timeline has no segments")
	}

	preset, err := d.presets.FindByID(ctx, req.PresetID)
	if err != nil {
		return nil, err
	}
	if preset == nil {
		return nil, errors.New("output preset not found")
	}

	var clip *domainclip.Clip
	if req.ClipID != nil {
		if d.clips == nil {
			return nil, errors.New("clip repository is not configured")
		}
		clip, err = d.clips.FindByID(ctx, *req.ClipID)
		if err != nil {
			return nil, err
		}
		if clip == nil || clip.ProjectID != req.ProjectID {
			return nil, errors.New("clip not found")
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
		m, err := d.media.GetByID(ctx, s.MediaFileID)
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
			if asset, err := d.media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindCropPath); err == nil && asset != nil && asset.StorageKey != "" {
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

	outputKey := fmt.Sprintf("projects/%s/renders/%s/out.mp4", req.ProjectID.String(), req.RenderID.String())

	plan := renderplan.Plan{
		RenderID: req.RenderID.String(),
		Kind:     string(req.Kind),
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

	if req.SubtitleStyleID != nil && req.Kind != domainrender.KindAudio {
		if d.ass == nil || d.storage == nil {
			return nil, errors.New("subtitle burn-in is not configured")
		}
		q := querytimeline.GetSubtitlesQuery{
			ProjectID: req.ProjectID,
			StyleID:   req.SubtitleStyleID,
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
		body, contentType, err := d.ass.Handle(ctx, q)
		if err != nil {
			return nil, err
		}
		assKey := fmt.Sprintf("projects/%s/renders/%s/subs.ass", req.ProjectID.String(), req.RenderID.String())
		if err := d.storage.PutObject(ctx, d.renderBucket, assKey, []byte(body), contentType); err != nil {
			return nil, errors.New("failed to upload subtitle track")
		}
		plan.Subtitles = &renderplan.Subtitles{ASSKey: assKey, FontsDir: "/fonts"}
	}

	return plan.Marshal()
}