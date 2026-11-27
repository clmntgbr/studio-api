package render

import (
	"context"
	"errors"
	"time"

	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/application/realtime"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

type ApplyResultCommand struct {
	RenderID    uuid.UUID
	OutputKey   string
	OutputParts []string
	DurationMs  int64
	SizeBytes   int64
}

type ApplyResultHandler struct {
	renders   domainrender.WriteRepository
	media     domainmedia.WriteRepository
	publisher *realtime.Publisher
}

func NewApplyResultHandler(
	renders domainrender.WriteRepository,
	media domainmedia.WriteRepository,
	transport port.RealtimePublisher,
) *ApplyResultHandler {
	var pub *realtime.Publisher
	if transport != nil {
		pub = realtime.NewPublisher(transport)
	}
	return &ApplyResultHandler{renders: renders, media: media, publisher: pub}
}

type FailCommand struct {
	RenderID uuid.UUID
	Message  string
}

func (h *ApplyResultHandler) Fail(ctx context.Context, cmd FailCommand) error {
	r, err := h.renders.GetByID(ctx, cmd.RenderID)
	if err != nil {
		return err
	}
	if r == nil {
		return errors.New("render not found")
	}
	if r.Status == domainrender.StatusCompleted || r.Status == domainrender.StatusCancelled || r.Status == domainrender.StatusFailed {
		return nil
	}
	now := time.Now().UTC()
	r.Status = domainrender.StatusFailed
	msg := cmd.Message
	if msg == "" {
		msg = "render failed"
	}
	r.ErrorMessage = &msg
	r.CompletedAt = &now
	if err := h.renders.Update(ctx, r); err != nil {
		return err
	}
	if r.Kind == domainrender.KindMaster && h.media != nil && r.MediaFileID != uuid.Nil {
		_ = mediacmd.FailStep(ctx, h.media, r.MediaFileID, domainmedia.StepRender, now, msg)
	}
	_ = mediacmd.ReturnMediaToReadyIfIdle(ctx, h.media, h.renders.HasInFlightByMedia, r.MediaFileID)
	h.publish(ctx, realtime.ActionRenderFailed, r, map[string]any{
		"renderId": r.ID.String(),
		"message":  msg,
	})
	return nil
}

func (h *ApplyResultHandler) Handle(ctx context.Context, cmd ApplyResultCommand) error {
	r, err := h.renders.GetByID(ctx, cmd.RenderID)
	if err != nil {
		return err
	}
	if r == nil {
		return errors.New("render not found")
	}
	if r.Status == domainrender.StatusCompleted || r.Status == domainrender.StatusCancelled {
		return nil
	}
	now := time.Now().UTC()
	r.Status = domainrender.StatusCompleted
	r.Progress = 100
	key := cmd.OutputKey
	r.StorageKey = &key
	r.OutputParts = append([]string(nil), cmd.OutputParts...)
	if cmd.DurationMs > 0 {
		d := cmd.DurationMs
		r.DurationMs = &d
	}
	if cmd.SizeBytes > 0 {
		s := cmd.SizeBytes
		r.SizeBytes = &s
	}
	r.CompletedAt = &now
	if r.StartedAt == nil {
		r.StartedAt = &now
	}
	if err := h.renders.Update(ctx, r); err != nil {
		return errors.New("failed to apply render result")
	}
	if r.Kind == domainrender.KindMaster && h.media != nil && r.MediaFileID != uuid.Nil {
		_ = mediacmd.CompleteStep(ctx, h.media, r.MediaFileID, domainmedia.StepRender, now)
	}
	_ = mediacmd.ReturnMediaToReadyIfIdle(ctx, h.media, h.renders.HasInFlightByMedia, r.MediaFileID)
	h.publish(ctx, realtime.ActionRenderCompleted, r, map[string]any{
		"renderId": r.ID.String(),
	})
	return nil
}

func (h *ApplyResultHandler) publish(ctx context.Context, action string, r *domainrender.Render, payload map[string]any) {
	if h.publisher == nil || r == nil {
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["mediaFileId"] = r.MediaFileID.String()
	_ = h.publisher.ToMedia(ctx, realtime.EntityRender, action, r.MediaFileID.String(), payload)
}
