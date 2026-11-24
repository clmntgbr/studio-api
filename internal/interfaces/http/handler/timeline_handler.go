package handler

import (
	"errors"
	"strings"

	cmdtimeline "go-api/internal/application/command/timeline"
	querytimeline "go-api/internal/application/query/timeline"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type TimelineHandler struct {
	previewHandler      timelinePreviewHandler
	validateHandler     timelineValidateHandler
	getHandler          timelineGetHandler
	listVersionsHandler timelineListVersionsHandler
}

func NewTimelineHandler(
	previewHandler timelinePreviewHandler,
	validateHandler timelineValidateHandler,
	getHandler timelineGetHandler,
	listVersionsHandler timelineListVersionsHandler,
) *TimelineHandler {
	return &TimelineHandler{
		previewHandler:      previewHandler,
		validateHandler:     validateHandler,
		getHandler:          getHandler,
		listVersionsHandler: listVersionsHandler,
	}
}

type timelinePreviewBody struct {
	MediaID string `json:"mediaId" validate:"omitempty,uuid"`
}

type timelineValidateBody struct {
	Version int `json:"version" validate:"required,min=1"`
}

func (h *TimelineHandler) Get(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	view, err := h.getHandler.Handle(c.Context(), querytimeline.GetTimelineQuery{ProjectID: projectID})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Timeline not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get timeline"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewTimelineResponseFromView(view))
}

func (h *TimelineHandler) ListVersions(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	items, err := h.listVersionsHandler.Handle(c.Context(), querytimeline.ListVersionsQuery{ProjectID: projectID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list timeline versions"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewTimelineVersionListResponse(items))
}

func (h *TimelineHandler) Preview(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var body timelinePreviewBody
	if len(c.Body()) > 0 {
		if err := validation.BindBody(c, &body); err != nil {
			return err
		}
	}
	var mediaID uuid.UUID
	if body.MediaID != "" {
		// Validated as uuid by BindBody above.
		mediaID = uuid.MustParse(body.MediaID)
	}
	result, err := h.previewHandler.Handle(c.Context(), cmdtimeline.PreviewCommand{
		ProjectID: projectID,
		MediaID:   mediaID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no media") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "duration") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to preview timeline"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewTimelinePreviewResponse(projectID.String(), result.Output))
}

func (h *TimelineHandler) Validate(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var body timelineValidateBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	t, err := h.validateHandler.Handle(c.Context(), cmdtimeline.ValidateCommand{
		ProjectID: projectID,
		Version:   body.Version,
	})
	if err != nil {
		if errors.Is(err, cmdtimeline.ErrStaleVersion) || strings.Contains(err.Error(), "stale") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"message": "Timeline version is stale",
				"code":    "STALE_TIMELINE_VERSION",
			})
		}
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Timeline not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to validate timeline"})
	}
	view, getErr := h.getHandler.Handle(c.Context(), querytimeline.GetTimelineQuery{ProjectID: projectID})
	if getErr != nil {
		return c.Status(fiber.StatusOK).JSON(presenter.NewTimelineResponseFromView(&querytimeline.TimelineView{
			Timeline: t,
		}))
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewTimelineResponseFromView(view))
}
