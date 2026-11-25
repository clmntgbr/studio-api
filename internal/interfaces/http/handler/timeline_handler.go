package handler

import (
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
	getSubtitlesHandler timelineGetSubtitlesHandler
	getExportHandler    timelineGetExportHandler
}

func NewTimelineHandler(
	previewHandler timelinePreviewHandler,
	validateHandler timelineValidateHandler,
	getHandler timelineGetHandler,
	listVersionsHandler timelineListVersionsHandler,
	getSubtitlesHandler timelineGetSubtitlesHandler,
	getExportHandler timelineGetExportHandler,
) *TimelineHandler {
	return &TimelineHandler{
		previewHandler:      previewHandler,
		validateHandler:     validateHandler,
		getHandler:          getHandler,
		listVersionsHandler: listVersionsHandler,
		getSubtitlesHandler: getSubtitlesHandler,
		getExportHandler:    getExportHandler,
	}
}

type timelinePreviewBody struct {
	MediaID string `json:"mediaId" validate:"omitempty,uuid"`
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
	return c.Status(fiber.StatusGone).JSON(fiber.Map{
		"message": "timeline validate removed; POST /projects/:id/renders to ship",
	})
}

func (h *TimelineHandler) GetSubtitlesASS(c fiber.Ctx) error {
	return h.getSubtitles(c, "ass")
}

func (h *TimelineHandler) GetSubtitlesSRT(c fiber.Ctx) error {
	return h.getSubtitles(c, "srt")
}

func (h *TimelineHandler) getSubtitles(c fiber.Ctx, format string) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var styleID *uuid.UUID
	if v := c.Query("styleId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid styleId"})
		}
		styleID = &id
	}
	body, contentType, err := h.getSubtitlesHandler.Handle(c.Context(), querytimeline.GetSubtitlesQuery{
		ProjectID: projectID,
		StyleID:   styleID,
		Format:    format,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no segments") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "must be") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to generate subtitles"})
	}
	filename := "subtitles." + format
	c.Set("Content-Type", contentType+"; charset=utf-8")
	c.Set("Content-Disposition", `inline; filename="`+filename+`"`)
	return c.Status(fiber.StatusOK).SendString(body)
}

func (h *TimelineHandler) GetExportFCPXML(c fiber.Ctx) error {
	return h.getExport(c, "fcpxml")
}

func (h *TimelineHandler) GetExportEDL(c fiber.Ctx) error {
	return h.getExport(c, "edl")
}

func (h *TimelineHandler) GetExportOTIO(c fiber.Ctx) error {
	return h.getExport(c, "otio")
}

func (h *TimelineHandler) GetExportPremiere(c fiber.Ctx) error {
	return h.getExport(c, "premiere")
}

func (h *TimelineHandler) getExport(c fiber.Ctx, format string) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	body, contentType, filename, err := h.getExportHandler.Handle(c.Context(), querytimeline.GetExportQuery{
		ProjectID: projectID,
		Format:    format,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no segments") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "must be") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to export timeline"})
	}
	c.Set("Content-Type", contentType+"; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	return c.Status(fiber.StatusOK).SendString(body)
}
