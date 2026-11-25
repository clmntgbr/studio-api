package render

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	mediacmd "go-api/internal/application/command/media"
	querytimeline "go-api/internal/application/query/timeline"
	domainclip "go-api/internal/domain/clip"
	domainmedia "go-api/internal/domain/media"
	domainpreset "go-api/internal/domain/outputpreset"
	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"
	domaintimeline "go-api/internal/domain/timeline"
	"go-api/internal/mastersplit"
	"go-api/internal/renderplan"
	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

type CreateCommand struct {
	ProjectID       uuid.UUID
	PresetID        uuid.UUID
	Kind            string
	SubtitleStyleID *uuid.UUID
	BurnSubtitles   bool // burn transcript captions (default style if SubtitleStyleID nil)
	ClipID          *uuid.UUID
	Force           bool // supersede any prior fingerprint (Ship again = full re-encode)
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

	burnSubs := cmd.BurnSubtitles || cmd.SubtitleStyleID != nil
	masterSplit := ""
	if kind == domainrender.KindMaster && len(segments) > 0 {
		if cfg, err := h.media.GetConfiguration(ctx, segments[0].MediaFileID); err != nil {
			return nil, err
		} else if cfg != nil {
			masterSplit = strings.TrimSpace(cfg.MasterSplit)
			if _, err := mastersplit.Parse(masterSplit); err != nil {
				return nil, err
			}
		}
	}
	fp := renderplan.Fingerprint(t.ID, preset.ID, cmd.SubtitleStyleID, cmd.ClipID, string(kind), preset.CropMode, burnSubs, masterSplit)
	if existing, err := h.renders.FindByFingerprint(ctx, cmd.ProjectID, fp); err != nil {
		return nil, err
	} else if existing != nil {
		reuse := !cmd.Force && (existing.Status == domainrender.StatusQueued ||
			existing.Status == domainrender.StatusRunning ||
			existing.Status == domainrender.StatusCompleted ||
			existing.Status == domainrender.StatusPending)
		if reuse {
			return existing, nil
		}
		// fingerprint is varchar(64); keep superseded marker short and unique.
		existing.Fingerprint = "s-" + existing.ID.String()
		if err := h.renders.Update(ctx, existing); err != nil {
			return nil, errors.New("failed to supersede previous render")
		}
	}

	workSegs := segments
	if clip != nil {
		window := eng.ContinuousClip(clip.MediaFileID, clip.SourceStartMs, clip.SourceEndMs)
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

	needsTrack := preset.CropMode == "face_track" || preset.CropMode == "speaker_track"
	if needsTrack && cropPathKey == nil {
		firstMediaID := workSegs[0].MediaFileID
		proxyAsset, err := h.media.GetAssetByKind(ctx, firstMediaID, domainmedia.AssetKindProxy)
		if err != nil {
			return nil, err
		}
		if proxyAsset == nil || proxyAsset.StorageKey == "" {
			return nil, errors.New("proxy not ready for reframe")
		}

		renderID := uuid.New()
		markerPlan, err := json.Marshal(map[string]any{
			"awaitingReframe": true,
			"mediaId":         firstMediaID.String(),
			"burnSubtitles":   burnSubs,
			"masterSplit":     masterSplit,
		})
		if err != nil {
			return nil, err
		}

		r := domainrender.New(cmd.ProjectID, t.ID, preset.ID, kind, fp, markerPlan)
		r.ID = renderID
		r.SubtitleStyleID = cmd.SubtitleStyleID
		r.ClipID = cmd.ClipID
		r.Status = domainrender.StatusPending

		if err := h.renders.Save(ctx, r); err != nil {
			return nil, errors.New("failed to create render")
		}
		if err := mediacmd.QueueStep(ctx, h.media, firstMediaID, domainmedia.StepReframe, time.Now().UTC()); err != nil {
			r.Status = domainrender.StatusFailed
			msg := "failed to queue reframe"
			r.ErrorMessage = &msg
			_ = h.renders.Update(ctx, r)
			return nil, err
		}
		if err := h.jobs.PublishReframeJob(ctx, firstMediaID.String(), proxyAsset.StorageKey, preset.Width, preset.Height); err != nil {
			r.Status = domainrender.StatusFailed
			msg := "failed to enqueue reframe"
			r.ErrorMessage = &msg
			_ = h.renders.Update(ctx, r)
			return nil, errors.New("failed to enqueue reframe")
		}
		return r, nil
	}

	renderID := uuid.New()
	rawPlan, err := buildQueuedPlan(ctx, queuedPlanDeps{
		timelines:    h.timelines,
		presets:      h.presets,
		media:        h.media,
		clips:        h.clips,
		ass:          h.ass,
		storage:      h.storage,
		renderBucket: h.renderBucket,
	}, queuedPlanRequest{
		ProjectID:       cmd.ProjectID,
		RenderID:        renderID,
		TimelineID:      t.ID,
		PresetID:        cmd.PresetID,
		Kind:            kind,
		SubtitleStyleID: cmd.SubtitleStyleID,
		BurnSubtitles:   burnSubs,
		ClipID:          cmd.ClipID,
		MasterSplit:     masterSplit,
	})
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
