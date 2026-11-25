package render

import (
	"context"
	"encoding/json"
	"errors"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"
	"go-api/internal/renderplan"

	"github.com/google/uuid"
)

type KickoffDeferredHandler struct {
	rendersWrite domainrender.WriteRepository
	rendersRead  domainrender.ReadRepository
	media        domainmedia.WriteRepository
	planDeps     queuedPlanDeps
	jobs         port.JobPublisher
}

func NewKickoffDeferredHandler(
	rendersWrite domainrender.WriteRepository,
	rendersRead domainrender.ReadRepository,
	media domainmedia.WriteRepository,
	planDeps queuedPlanDeps,
	jobs port.JobPublisher,
) *KickoffDeferredHandler {
	return &KickoffDeferredHandler{
		rendersWrite: rendersWrite,
		rendersRead:  rendersRead,
		media:        media,
		planDeps:     planDeps,
		jobs:         jobs,
	}
}

func (h *KickoffDeferredHandler) Handle(ctx context.Context, mediaID uuid.UUID) error {
	if h == nil {
		return nil
	}
	m, err := h.media.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	renders, err := h.rendersRead.ListByProject(ctx, m.ProjectID)
	if err != nil {
		return err
	}

	mediaIDStr := mediaID.String()
	for i := range renders {
		r := &renders[i]
		if r.Status != domainrender.StatusPending {
			continue
		}
		waitMediaID, ok, err := deferredMarkerMediaID(r.Plan)
		if err != nil {
			// Corrupt / non-marker pending row — skip so other deferrals still resume.
			continue
		}
		if !ok || waitMediaID != mediaIDStr {
			continue
		}

		rawPlan, err := buildQueuedPlan(ctx, h.planDeps, queuedPlanRequest{
			ProjectID:       r.ProjectID,
			RenderID:        r.ID,
			TimelineID:      r.TimelineID,
			PresetID:        r.OutputPresetID,
			Kind:            r.Kind,
			SubtitleStyleID: r.SubtitleStyleID,
			ClipID:          r.ClipID,
		})
		if err != nil {
			msg := err.Error()
			r.Status = domainrender.StatusFailed
			r.ErrorMessage = &msg
			if upErr := h.rendersWrite.Update(ctx, r); upErr != nil {
				return upErr
			}
			continue
		}

		r.Plan = rawPlan
		r.Status = domainrender.StatusQueued
		r.ErrorMessage = nil
		if err := h.rendersWrite.Update(ctx, r); err != nil {
			return err
		}
		if err := h.jobs.PublishRenderJob(ctx, r.ID.String(), rawPlan); err != nil {
			msg := "failed to enqueue render"
			r.Status = domainrender.StatusFailed
			r.ErrorMessage = &msg
			_ = h.rendersWrite.Update(ctx, r)
			continue
		}
	}
	return nil
}

func deferredMarkerMediaID(plan json.RawMessage) (mediaID string, awaiting bool, err error) {
	var marker struct {
		AwaitingReframe bool   `json:"awaitingReframe"`
		MediaID         string `json:"mediaId"`
	}
	if err := json.Unmarshal(plan, &marker); err != nil {
		return "", false, err
	}
	if !marker.AwaitingReframe || marker.MediaID == "" {
		return "", false, nil
	}
	return marker.MediaID, true, nil
}

// planHasOutput confirms the plan is a real render job payload, not a defer marker.
func planHasOutput(plan json.RawMessage) bool {
	var p renderplan.Plan
	return json.Unmarshal(plan, &p) == nil && p.Output.Width > 0
}