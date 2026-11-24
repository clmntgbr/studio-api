package handler

import (
	"strings"

	querytranscript "go-api/internal/application/query/transcript"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type TranscriptHandler struct {
	getHandler    transcriptGetHandler
	getSRTHandler transcriptGetSRTHandler
}

func NewTranscriptHandler(getHandler transcriptGetHandler, getSRTHandler transcriptGetSRTHandler) *TranscriptHandler {
	return &TranscriptHandler{getHandler: getHandler, getSRTHandler: getSRTHandler}
}

func (h *TranscriptHandler) GetByMedia(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	var fromMs, toMs *int64
	if v := c.Query("fromMs"); v != "" {
		n, err := parseInt64Query(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid fromMs"})
		}
		fromMs = &n
	}
	if v := c.Query("toMs"); v != "" {
		n, err := parseInt64Query(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid toMs"})
		}
		toMs = &n
	}
	if fromMs != nil && toMs != nil && *fromMs > *toMs {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "fromMs must be <= toMs"})
	}
	view, err := h.getHandler.Handle(c.Context(), querytranscript.GetTranscriptQuery{
		MediaID: mediaID,
		FromMs:  fromMs,
		ToMs:    toMs,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Transcript not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get transcript"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewTranscriptResponse(view))
}

func (h *TranscriptHandler) GetSRT(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	srt, err := h.getSRTHandler.Handle(c.Context(), querytranscript.GetSRTQuery{MediaID: mediaID})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Transcript not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to export transcript srt"})
	}
	c.Set("Content-Type", "application/x-subrip; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="transcript.srt"`)
	return c.Status(fiber.StatusOK).SendString(srt)
}
