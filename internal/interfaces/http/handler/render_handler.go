package handler

import (
	"strings"

	cmdrender "go-api/internal/application/command/render"
	queryrender "go-api/internal/application/query/render"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type RenderHandler struct {
	createHandler   renderCreateHandler
	cancelHandler   renderCancelHandler
	getHandler      renderGetHandler
	listHandler     renderListHandler
	downloadHandler renderDownloadHandler
}

func NewRenderHandler(
	createHandler renderCreateHandler,
	cancelHandler renderCancelHandler,
	getHandler renderGetHandler,
	listHandler renderListHandler,
	downloadHandler renderDownloadHandler,
) *RenderHandler {
	return &RenderHandler{
		createHandler:   createHandler,
		cancelHandler:   cancelHandler,
		getHandler:      getHandler,
		listHandler:     listHandler,
		downloadHandler: downloadHandler,
	}
}

type createRenderBody struct {
	PresetID        string  `json:"presetId" validate:"required,uuid"`
	Kind            string  `json:"kind" validate:"omitempty,oneof=master clip audio custom"`
	SubtitleStyleID *string `json:"subtitleStyleId" validate:"omitempty,uuid"`
	ClipID          *string `json:"clipId" validate:"omitempty,uuid"`
}

func (h *RenderHandler) Create(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var body createRenderBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	presetID := uuid.MustParse(body.PresetID)
	var styleID *uuid.UUID
	if body.SubtitleStyleID != nil && *body.SubtitleStyleID != "" {
		id := uuid.MustParse(*body.SubtitleStyleID)
		styleID = &id
	}
	var clipID *uuid.UUID
	if body.ClipID != nil && *body.ClipID != "" {
		id := uuid.MustParse(*body.ClipID)
		clipID = &id
	}
	r, err := h.createHandler.Handle(c.Context(), cmdrender.CreateCommand{
		ProjectID:       projectID,
		PresetID:        presetID,
		Kind:            body.Kind,
		SubtitleStyleID: styleID,
		ClipID:          clipID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "must be validated") {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not ready") || strings.Contains(err.Error(), "no segments") || strings.Contains(err.Error(), "no overlapping") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "required") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create render"})
	}
	return c.Status(fiber.StatusAccepted).JSON(presenter.NewRenderResponse(r))
}

func (h *RenderHandler) ListByProject(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	items, err := h.listHandler.Handle(c.Context(), queryrender.ListQuery{ProjectID: projectID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list renders"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewRenderListResponse(items))
}

func (h *RenderHandler) GetByID(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid render id"})
	}
	r, err := h.getHandler.Handle(c.Context(), queryrender.GetQuery{ID: id})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Render not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get render"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewRenderResponse(r))
}

func (h *RenderHandler) Cancel(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid render id"})
	}
	r, err := h.cancelHandler.Handle(c.Context(), cmdrender.CancelCommand{RenderID: id})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Render not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to cancel render"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewRenderResponse(r))
}

func (h *RenderHandler) Download(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid render id"})
	}
	out, err := h.downloadHandler.Handle(c.Context(), queryrender.DownloadQuery{ID: id})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Render not found"})
		}
		if strings.Contains(err.Error(), "not ready") || strings.Contains(err.Error(), "missing") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create download URL"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"url":       out.URL,
		"expiresIn": out.ExpiresIn,
	})
}
