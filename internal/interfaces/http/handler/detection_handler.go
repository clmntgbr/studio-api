package handler

import (
	querydetection "go-api/internal/application/query/detection"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type DetectionHandler struct {
	listHandler detectionListHandler
}

func NewDetectionHandler(listHandler detectionListHandler) *DetectionHandler {
	return &DetectionHandler{listHandler: listHandler}
}

func (h *DetectionHandler) ListByMedia(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	kind := c.Query("kind")
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
	items, err := h.listHandler.Handle(c.Context(), querydetection.ListDetectionsQuery{
		MediaID: mediaID,
		Kind:    kind,
		FromMs:  fromMs,
		ToMs:    toMs,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list detections"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewDetectionListResponse(items))
}
