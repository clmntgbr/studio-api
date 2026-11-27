package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
)

type FromURLCommand struct {
	URL string
}

type FromURLHandler struct {
	media  domainmedia.WriteRepository
	outbox port.OutboxRepository
	jobs   port.JobPublisher
}

func NewFromURLHandler(
	media domainmedia.WriteRepository,
	outbox port.OutboxRepository,
	jobs port.JobPublisher,
) *FromURLHandler {
	return &FromURLHandler{media: media, outbox: outbox, jobs: jobs}
}

func (h *FromURLHandler) Handle(ctx context.Context, cmd FromURLCommand) (*domainmedia.MediaFile, error) {
	if cmd.URL == "" {
		return nil, errors.New("url is required")
	}

	m := domainmedia.NewFromURL(cmd.URL, 0)
	cfg := domainmedia.DefaultConfiguration(m.ID)
	step := domainmedia.NewProcessingStep(m.ID, domainmedia.StepFetch)

	err := h.media.WithTransaction(ctx, func(txCtx context.Context) error {
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