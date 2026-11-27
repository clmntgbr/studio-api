package detection

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	mediacmd "go-api/internal/application/command/media"
	domainclip "go-api/internal/domain/clip"
	domaindetection "go-api/internal/domain/detection"
	domainllm "go-api/internal/domain/llmartifact"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type LLMSponsorInterval struct {
	SourceStartMs int64
	SourceEndMs   int64
	FirstWord     *int
	LastWord      *int
	Confidence    *float64
	Reason        string
}

type LLMChapter struct {
	Title         string `json:"title"`
	SourceStartMs int64  `json:"sourceStartMs"`
	SourceEndMs   int64  `json:"sourceEndMs"`
}

type LLMClipCandidate struct {
	SourceStartMs int64
	SourceEndMs   int64
	Score         *float64
	Hook          string
	Reason        string
	Title         string
	Description   string
	Hashtags      []string
}

type ApplyLLMResultCommand struct {
	MediaID      uuid.UUID
	ArtifactsKey string
	Provider     string
	Sponsors     []LLMSponsorInterval
	Chapters     []LLMChapter
	Clips        []LLMClipCandidate
}

type ApplyLLMResultHandler struct {
	media      domainmedia.WriteRepository
	detections domaindetection.WriteRepository
	artifacts  domainllm.WriteRepository
	clips      domainclip.WriteRepository
	outbox     port.OutboxRepository
}

func NewApplyLLMResultHandler(
	media domainmedia.WriteRepository,
	detections domaindetection.WriteRepository,
	artifacts domainllm.WriteRepository,
	clips domainclip.WriteRepository,
	outbox port.OutboxRepository,
) *ApplyLLMResultHandler {
	return &ApplyLLMResultHandler{
		media:      media,
		detections: detections,
		artifacts:  artifacts,
		clips:      clips,
		outbox:     outbox,
	}
}

func (h *ApplyLLMResultHandler) Handle(ctx context.Context, cmd ApplyLLMResultCommand) error {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}
	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepLLM)
	if err != nil {
		return err
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}

	detector := "llm.sponsor@1"
	if cmd.Provider != "" {
		detector = "llm.sponsor@" + cmd.Provider
	}
	now := time.Now().UTC()
	provider := cmd.Provider
	if provider == "" {
		provider = "unknown"
	}

	sponsorItems := make([]*domaindetection.Detection, 0, len(cmd.Sponsors))
	for _, iv := range cmd.Sponsors {
		if iv.SourceEndMs <= iv.SourceStartMs {
			continue
		}
		payload, _ := json.Marshal(map[string]any{
			"reason":       iv.Reason,
			"artifactsKey": cmd.ArtifactsKey,
		})
		d := domaindetection.New(m.ID, domaindetection.KindSponsor, iv.SourceStartMs, iv.SourceEndMs, detector, payload)
		d.FirstWordIndex = iv.FirstWord
		d.LastWordIndex = iv.LastWord
		d.Confidence = iv.Confidence
		sponsorItems = append(sponsorItems, d)
	}

	chaptersPayload, _ := json.Marshal(map[string]any{
		"chapters":     cmd.Chapters,
		"artifactsKey": cmd.ArtifactsKey,
	})
	chapterArtifact := domainllm.New(m.ID, "chapters", chaptersPayload, &provider)

	clipItems := make([]*domainclip.Clip, 0, len(cmd.Clips))
	for _, c := range cmd.Clips {
		if c.SourceEndMs <= c.SourceStartMs {
			continue
		}
		clip := domainclip.New(m.ID, c.SourceStartMs, c.SourceEndMs, "llm")
		clip.Score = c.Score
		if c.Hook != "" {
			h := c.Hook
			clip.Hook = &h
		}
		if c.Reason != "" {
			r := c.Reason
			clip.Reason = &r
		}
		if c.Title != "" {
			t := c.Title
			clip.Title = &t
		}
		if c.Description != "" {
			d := c.Description
			clip.Description = &d
		}
		if len(c.Hashtags) > 0 {
			clip.Hashtags = append([]string(nil), c.Hashtags...)
		}
		clipItems = append(clipItems, clip)
	}

	m.RecordDetectionsUpdated("sponsor")

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.detections.DeleteByMediaAndKind(txCtx, m.ID, domaindetection.KindSponsor); err != nil {
			return err
		}
		if err := h.detections.SaveMany(txCtx, sponsorItems); err != nil {
			return err
		}
		if err := h.artifacts.DeleteByMediaAndKind(txCtx, m.ID, "chapters"); err != nil {
			return err
		}
		if err := h.artifacts.Save(txCtx, chapterArtifact); err != nil {
			return err
		}
		if err := h.clips.DeleteByMediaAndOrigin(txCtx, m.ID, "llm"); err != nil {
			return err
		}
		for _, c := range clipItems {
			if err := h.clips.Save(txCtx, c); err != nil {
				return err
			}
		}
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		if err := mediacmd.CompleteStep(txCtx, h.media, m.ID, domainmedia.StepLLM, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return errors.New("failed to apply llm result")
	}
	return mediacmd.TryMarkMediaReady(ctx, h.media, h.outbox, cmd.MediaID)
}
