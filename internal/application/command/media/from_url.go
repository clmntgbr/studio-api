package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	domainproject "go-api/internal/domain/project"

	"github.com/google/uuid"
)

type FromURLCommand struct {
	ProjectID uuid.UUID
	URL       string
}

type FromURLHandler struct {
	projects domainproject.WriteRepository
	media    domainmedia.WriteRepository
	outbox   port.OutboxRepository
	jobs     port.JobPublisher
}

func NewFromURLHandler(
	projects domainproject.WriteRepository,
	media domainmedia.WriteRepository,
	outbox port.OutboxRepository,
	jobs port.JobPublisher,
) *FromURLHandler {
	return &FromURLHandler{projects: projects, media: media, outbox: outbox, jobs: jobs}
}

func (h *FromURLHandler) Handle(ctx context.Context, cmd FromURLCommand) (*domainmedia.MediaFile, error) {
	if cmd.URL == "" {
		return nil, errors.New("url is required")
	}
	project, err := h.projects.GetByID(ctx, cmd.ProjectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	m := domainmedia.NewFromURL(cmd.ProjectID, cmd.URL, 0)
	cfg := domainmedia.DefaultConfiguration(m.ID)
	step := domainmedia.NewProcessingStep(m.ID, domainmedia.StepFetch)

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.media.Save(txCtx, m); err != nil {
			return err
		}
		if err := h.media.SaveConfiguration(txCtx, cfg); err != nil {
			return err
		}
		if err := h.media.SaveStep(txCtx, step); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return nil, errors.New("failed to create media from url")
	}

	if err := h.jobs.PublishFetchJob(ctx, m.ID.String(), cmd.URL); err != nil {
		return nil, errors.New("failed to enqueue fetch job")
	}
	return m, nil
}
