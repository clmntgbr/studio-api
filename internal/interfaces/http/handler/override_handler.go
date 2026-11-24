package handler

import (
	"strings"

	cmddetection "go-api/internal/application/command/detection"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type OverrideHandler struct {
	createHandler overrideCreateHandler
	deleteHandler overrideDeleteHandler
}

func NewOverrideHandler(createHandler overrideCreateHandler, deleteHandler overrideDeleteHandler) *OverrideHandler {
	return &OverrideHandler{createHandler: createHandler, deleteHandler: deleteHandler}
}

type createOverrideBody struct {
	Action        string  `json:"action" validate:"required,oneof=keep remove"`
	SourceStartMs int64   `json:"sourceStartMs" validate:"min=0"`
	SourceEndMs   int64   `json:"sourceEndMs" validate:"required,gtfield=SourceStartMs"`
	Kind          *string `json:"kind" validate:"omitempty,oneof=silence filler repetition false_start sponsor ad"`
}

func (h *OverrideHandler) Create(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	var body createOverrideBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	o, err := h.createHandler.Handle(c.Context(), cmddetection.CreateOverrideCommand{
		MediaID:       mediaID,
		Action:        body.Action,
		SourceStartMs: body.SourceStartMs,
		SourceEndMs:   body.SourceEndMs,
		Kind:          body.Kind,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		if strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "invalid") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create override"})
	}
	return c.Status(fiber.StatusCreated).JSON(presenter.NewOverrideResponse(o))
}

func (h *OverrideHandler) Delete(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid override id"})
	}
	if err := h.deleteHandler.Handle(c.Context(), cmddetection.DeleteOverrideCommand{OverrideID: id}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Override not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to delete override"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
