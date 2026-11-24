package handler

import (
	"strings"

	cmdclip "go-api/internal/application/command/clip"
	cmdrender "go-api/internal/application/command/render"
	queryclip "go-api/internal/application/query/clip"
	queryllm "go-api/internal/application/query/llmartifact"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ClipHandler struct {
	listHandler      clipListHandler
	createHandler    clipCreateHandler
	updateHandler    clipUpdateHandler
	analyzeHandler   clipAnalyzeHandler
	artifactsHandler llmArtifactListHandler
	getHandler       clipGetHandler
	renderHandler    clipRenderHandler
}

func NewClipHandler(
	listHandler clipListHandler,
	createHandler clipCreateHandler,
	updateHandler clipUpdateHandler,
	analyzeHandler clipAnalyzeHandler,
	artifactsHandler llmArtifactListHandler,
	getHandler clipGetHandler,
	renderHandler clipRenderHandler,
) *ClipHandler {
	return &ClipHandler{
		listHandler:      listHandler,
		createHandler:    createHandler,
		updateHandler:    updateHandler,
		analyzeHandler:   analyzeHandler,
		artifactsHandler: artifactsHandler,
		getHandler:       getHandler,
		renderHandler:    renderHandler,
	}
}

type createClipBody struct {
	MediaFileID   string  `json:"mediaFileId" validate:"required,uuid"`
	SourceStartMs int64   `json:"sourceStartMs" validate:"gte=0"`
	SourceEndMs   int64   `json:"sourceEndMs" validate:"required,gt=0"`
	Title         *string `json:"title" validate:"omitempty,max=200"`
}

type updateClipBody struct {
	Selected      *bool     `json:"selected"`
	SourceStartMs *int64    `json:"sourceStartMs" validate:"omitempty,gte=0"`
	SourceEndMs   *int64    `json:"sourceEndMs" validate:"omitempty,gt=0"`
	Title         *string   `json:"title" validate:"omitempty,max=200"`
	Description   *string   `json:"description" validate:"omitempty,max=4000"`
	Hashtags      *[]string `json:"hashtags" validate:"omitempty,dive,max=64"`
}

type analyzeClipsBody struct {
	MediaID string `json:"mediaId" validate:"required,uuid"`
}

type renderClipBody struct {
	PresetID        string  `json:"presetId" validate:"omitempty,uuid"`
	SubtitleStyleID *string `json:"subtitleStyleId" validate:"omitempty,uuid"`
}

// Default vertical preset seed id (migrations/00005_output_presets_seed.sql).
var defaultVerticalPresetID = uuid.MustParse("01960000-0000-7000-8000-0000000000a2")

func (h *ClipHandler) ListByProject(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	items, err := h.listHandler.Handle(c.Context(), queryclip.ListQuery{ProjectID: projectID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list clips"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewClipListResponse(items))
}

func (h *ClipHandler) Create(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var body createClipBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	out, err := h.createHandler.Handle(c.Context(), cmdclip.CreateCommand{
		ProjectID:     projectID,
		MediaFileID:   uuid.MustParse(body.MediaFileID),
		SourceStartMs: body.SourceStartMs,
		SourceEndMs:   body.SourceEndMs,
		Title:         body.Title,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "must be") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create clip"})
	}
	return c.Status(fiber.StatusCreated).JSON(presenter.NewClipResponse(out))
}

func (h *ClipHandler) Update(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid clip id"})
	}
	var body updateClipBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	out, err := h.updateHandler.Handle(c.Context(), cmdclip.UpdateCommand{
		ClipID:        id,
		Selected:      body.Selected,
		SourceStartMs: body.SourceStartMs,
		SourceEndMs:   body.SourceEndMs,
		Title:         body.Title,
		Description:   body.Description,
		Hashtags:      body.Hashtags,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Clip not found"})
		}
		if strings.Contains(err.Error(), "must be") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update clip"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewClipResponse(out))
}

func (h *ClipHandler) Analyze(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var body analyzeClipsBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	if err := h.analyzeHandler.Handle(c.Context(), cmdclip.AnalyzeCommand{
		ProjectID: projectID,
		MediaID:   uuid.MustParse(body.MediaID),
	}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "not ready") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to start clip analysis"})
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"message": "LLM analysis queued"})
}

func (h *ClipHandler) Render(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	clipID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid clip id"})
	}
	var body renderClipBody
	if len(c.Body()) > 0 {
		if err := validation.BindBody(c, &body); err != nil {
			return err
		}
	}
	presetID := defaultVerticalPresetID
	if body.PresetID != "" {
		presetID = uuid.MustParse(body.PresetID)
	}
	var styleID *uuid.UUID
	if body.SubtitleStyleID != nil && *body.SubtitleStyleID != "" {
		id := uuid.MustParse(*body.SubtitleStyleID)
		styleID = &id
	}
	clip, err := h.getHandler.Handle(c.Context(), queryclip.GetQuery{ID: clipID})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Clip not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to load clip"})
	}
	r, err := h.renderHandler.Handle(c.Context(), cmdrender.CreateCommand{
		ProjectID:       clip.ProjectID,
		PresetID:        presetID,
		Kind:            "clip",
		SubtitleStyleID: styleID,
		ClipID:          &clipID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "must be validated") {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not ready") || strings.Contains(err.Error(), "no overlapping") || strings.Contains(err.Error(), "no segments") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "required") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create clip render"})
	}
	return c.Status(fiber.StatusAccepted).JSON(presenter.NewRenderResponse(r))
}

func (h *ClipHandler) ListArtifactsByMedia(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	kind := c.Query("kind")
	items, err := h.artifactsHandler.Handle(c.Context(), queryllm.ListQuery{MediaID: mediaID, Kind: kind})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list LLM artifacts"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewLLMArtifactListResponse(items))
}
