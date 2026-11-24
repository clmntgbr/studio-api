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
	items, err := h.listHandler.Handle(c.Context(), querydetection.ListDetectionsQuery{
		MediaID: mediaID,
		Kind:    kind,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list detections"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewDetectionListResponse(items))
}
