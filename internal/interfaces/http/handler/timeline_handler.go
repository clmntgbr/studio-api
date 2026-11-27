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
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	view, err := h.getHandler.Handle(c.Context(), querytimeline.GetTimelineQuery{MediaID: mediaID})
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
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	items, err := h.listVersionsHandler.Handle(c.Context(), querytimeline.ListVersionsQuery{MediaID: mediaID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list timeline versions"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewTimelineVersionListResponse(items))
}

func (h *TimelineHandler) Preview(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	var body timelinePreviewBody
	if len(c.Body()) > 0 {
		if err := validation.BindBody(c, &body); err != nil {
			return err
		}
	}
	// Body mediaId is optional; path media is the source of truth.
	_ = body
	result, err := h.previewHandler.Handle(c.Context(), cmdtimeline.PreviewCommand{
		MediaID: mediaID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no media") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "media is not editable") || strings.Contains(err.Error(), "media is not ready") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "duration") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to preview timeline"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewTimelinePreviewResponse(mediaID.String(), result.Output))
}

func (h *TimelineHandler) Validate(c fiber.Ctx) error {
	return c.Status(fiber.StatusGone).JSON(fiber.Map{
		"message": "timeline validate removed; POST /media/:id/renders to render",
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
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
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
		MediaID: mediaID,
		StyleID: styleID,
		Format:  format,
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
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	body, contentType, filename, err := h.getExportHandler.Handle(c.Context(), querytimeline.GetExportQuery{
		MediaID: mediaID,
		Format:  format,
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
