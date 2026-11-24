package handler

import (
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"

	"github.com/gofiber/fiber/v3"
)

type OutputPresetHandler struct {
	listHandler outputPresetListHandler
}

func NewOutputPresetHandler(listHandler outputPresetListHandler) *OutputPresetHandler {
	return &OutputPresetHandler{listHandler: listHandler}
}

func (h *OutputPresetHandler) List(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	items, err := h.listHandler.Handle(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list output presets"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewOutputPresetListResponse(items))
}
