package media

import (
	"context"
	"encoding/json"

	"go-api/internal/application/messaging"
	"go-api/internal/application/realtime"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
)

type PublishRealtimeHandler struct {
	publisher *realtime.Publisher
}

func NewPublishRealtimeHandler(transport port.RealtimePublisher) *PublishRealtimeHandler {
	return &PublishRealtimeHandler{publisher: realtime.NewPublisher(transport)}
}

func (h *PublishRealtimeHandler) OnSourceReady(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaSourceReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionStarted, evt.ProjectID, map[string]any{
		"mediaId": evt.MediaID,
		"step":    "probe",
	})
}

func (h *PublishRealtimeHandler) OnProbed(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaProbed
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionCompleted, evt.ProjectID, map[string]any{
		"mediaId": evt.MediaID,
		"step":    "probe",
	})
}

func (h *PublishRealtimeHandler) OnAudioReady(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaAudioReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionCompleted, evt.ProjectID, map[string]any{
		"mediaId": evt.MediaID,
		"step":    "audio",
	})
}

func (h *PublishRealtimeHandler) OnProxyReady(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaProxyReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionCompleted, evt.ProjectID, map[string]any{
		"mediaId": evt.MediaID,
		"step":    "proxy",
	})
}

func (h *PublishRealtimeHandler) OnWaveformReady(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaWaveformReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionCompleted, evt.ProjectID, map[string]any{
		"mediaId": evt.MediaID,
		"step":    "waveform",
	})
}

func (h *PublishRealtimeHandler) OnReady(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionUpdated, evt.ProjectID, map[string]any{
		"mediaId": evt.MediaID,
		"status":  "ready",
	})
}

func (h *PublishRealtimeHandler) OnDetectionsUpdated(ctx context.Context, payload []byte) error {
	var evt domainmedia.DetectionsUpdated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToProject(ctx, "detections", "updated", evt.ProjectID, map[string]any{
		"mediaId": evt.MediaID,
		"kind":    evt.Kind,
	})
}

func (h *PublishRealtimeHandler) OnTranscribed(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaTranscribed
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionCompleted, evt.ProjectID, map[string]any{
		"mediaId":  evt.MediaID,
		"step":     "transcribe",
		"wordsKey": evt.WordsKey,
		"language": evt.Language,
	})
}
