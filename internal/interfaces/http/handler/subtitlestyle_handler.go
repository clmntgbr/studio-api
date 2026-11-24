package handler

import (
	"encoding/json"
	"strings"

	cmdstyle "go-api/internal/application/command/subtitlestyle"
	querystyle "go-api/internal/application/query/subtitlestyle"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type SubtitleStyleHandler struct {
	createHandler subtitleStyleCreateHandler
	updateHandler subtitleStyleUpdateHandler
	deleteHandler subtitleStyleDeleteHandler
	getHandler    subtitleStyleGetHandler
	listHandler   subtitleStyleListHandler
}

func NewSubtitleStyleHandler(
	createHandler subtitleStyleCreateHandler,
	updateHandler subtitleStyleUpdateHandler,
	deleteHandler subtitleStyleDeleteHandler,
	getHandler subtitleStyleGetHandler,
	listHandler subtitleStyleListHandler,
) *SubtitleStyleHandler {
	return &SubtitleStyleHandler{
		createHandler: createHandler,
		updateHandler: updateHandler,
		deleteHandler: deleteHandler,
		getHandler:    getHandler,
		listHandler:   listHandler,
	}
}

type subtitleStyleBody struct {
	ProjectID      string          `json:"projectId" validate:"required,uuid"`
	Name           string          `json:"name" validate:"required,min=1,max=100"`
	FontFamily     *string         `json:"fontFamily" validate:"omitempty,max=255"`
	FontSize       *int            `json:"fontSize" validate:"omitempty,min=8,max=200"`
	BaseColor      *string         `json:"baseColor" validate:"omitempty,max=9"`
	HighlightColor *string         `json:"highlightColor" validate:"omitempty,max=9"`
	OutlineColor   *string         `json:"outlineColor" validate:"omitempty,max=9"`
	OutlineSize    *int            `json:"outlineSize" validate:"omitempty,min=0,max=20"`
	WordsPerLine   *int            `json:"wordsPerLine" validate:"omitempty,min=2,max=4"`
	Position       *string         `json:"position" validate:"omitempty,oneof=top middle bottom"`
	Uppercase      *bool           `json:"uppercase"`
	Settings       json.RawMessage `json:"settings"`
}

type subtitleStyleUpdateBody struct {
	Name           *string         `json:"name" validate:"omitempty,min=1,max=100"`
	FontFamily     *string         `json:"fontFamily" validate:"omitempty,max=255"`
	FontSize       *int            `json:"fontSize" validate:"omitempty,min=8,max=200"`
	BaseColor      *string         `json:"baseColor" validate:"omitempty,max=9"`
	HighlightColor *string         `json:"highlightColor" validate:"omitempty,max=9"`
	OutlineColor   *string         `json:"outlineColor" validate:"omitempty,max=9"`
	OutlineSize    *int            `json:"outlineSize" validate:"omitempty,min=0,max=20"`
	WordsPerLine   *int            `json:"wordsPerLine" validate:"omitempty,min=2,max=4"`
	Position       *string         `json:"position" validate:"omitempty,oneof=top middle bottom"`
	Uppercase      *bool           `json:"uppercase"`
	Settings       json.RawMessage `json:"settings"`
}

func (h *SubtitleStyleHandler) List(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	var projectID *uuid.UUID
	if v := c.Query("projectId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid projectId"})
		}
		projectID = &id
	}
	items, err := h.listHandler.Handle(c.Context(), querystyle.ListQuery{ProjectID: projectID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list subtitle styles"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewSubtitleStyleListResponse(items))
}

func (h *SubtitleStyleHandler) GetByID(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid style id"})
	}
	s, err := h.getHandler.Handle(c.Context(), querystyle.GetQuery{ID: id})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Subtitle style not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get subtitle style"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewSubtitleStyleResponse(s))
}

func (h *SubtitleStyleHandler) Create(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	var body subtitleStyleBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	projectID := uuid.MustParse(body.ProjectID)
	s, err := h.createHandler.Handle(c.Context(), cmdstyle.CreateCommand{
		ProjectID:      &projectID,
		Name:           body.Name,
		FontFamily:     body.FontFamily,
		FontSize:       body.FontSize,
		BaseColor:      body.BaseColor,
		HighlightColor: body.HighlightColor,
		OutlineColor:   body.OutlineColor,
		OutlineSize:    body.OutlineSize,
		WordsPerLine:   body.WordsPerLine,
		Position:       body.Position,
		Uppercase:      body.Uppercase,
		Settings:       body.Settings,
	})
	if err != nil {
		if strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "required") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create subtitle style"})
	}
	return c.Status(fiber.StatusCreated).JSON(presenter.NewSubtitleStyleResponse(s))
}

func (h *SubtitleStyleHandler) Update(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid style id"})
	}
	var body subtitleStyleUpdateBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	s, err := h.updateHandler.Handle(c.Context(), cmdstyle.UpdateCommand{
		ID:             id,
		Name:           body.Name,
		FontFamily:     body.FontFamily,
		FontSize:       body.FontSize,
		BaseColor:      body.BaseColor,
		HighlightColor: body.HighlightColor,
		OutlineColor:   body.OutlineColor,
		OutlineSize:    body.OutlineSize,
		WordsPerLine:   body.WordsPerLine,
		Position:       body.Position,
		Uppercase:      body.Uppercase,
		Settings:       body.Settings,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Subtitle style not found"})
		}
		if strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "required") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update subtitle style"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewSubtitleStyleResponse(s))
}

func (h *SubtitleStyleHandler) Delete(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid style id"})
	}
	if err := h.deleteHandler.Handle(c.Context(), cmdstyle.DeleteCommand{ID: id}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Subtitle style not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to delete subtitle style"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
