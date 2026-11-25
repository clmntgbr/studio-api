package render

import (
	"context"
	"errors"
	"time"

	"go-api/internal/application/realtime"
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
	publisher *realtime.Publisher
}

func NewApplyResultHandler(renders domainrender.WriteRepository, transport port.RealtimePublisher) *ApplyResultHandler {
	var pub *realtime.Publisher
	if transport != nil {
		pub = realtime.NewPublisher(transport)
	}
	return &ApplyResultHandler{renders: renders, publisher: pub}
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
	h.publish(ctx, realtime.ActionRenderCompleted, r, map[string]any{
		"renderId": r.ID.String(),
	})
	return nil
}

func (h *ApplyResultHandler) publish(ctx context.Context, action string, r *domainrender.Render, payload map[string]any) {
	if h.publisher == nil || r == nil {
		return
	}
	_ = h.publisher.ToProject(ctx, realtime.EntityRender, action, r.ProjectID.String(), payload)
}
