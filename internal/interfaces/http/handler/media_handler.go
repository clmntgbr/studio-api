package handler

import (
	"strings"

	cmdmedia "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
	"go-api/internal/domain/paginate"
	"go-api/internal/domain/port"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"
	"go-api/internal/pipeline"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type MediaHandler struct {
	initiateHandler     mediaInitiateUploadHandler
	presignHandler      mediaPresignPartsHandler
	completeHandler     mediaCompleteUploadHandler
	fromURLHandler      mediaFromURLHandler
	deleteHandler       mediaDeleteHandler
	getHandler          mediaGetHandler
	listHandler         mediaListHandler
	updateConfigHandler mediaUpdateConfigurationHandler
	getConfigHandler    mediaGetConfigurationHandler
	reanalyzeHandler    mediaReanalyzeHandler
	listStepRunsHandler mediaListStepRunsHandler
}

func NewMediaHandler(
	initiateHandler mediaInitiateUploadHandler,
	presignHandler mediaPresignPartsHandler,
	completeHandler mediaCompleteUploadHandler,
	fromURLHandler mediaFromURLHandler,
	deleteHandler mediaDeleteHandler,
	getHandler mediaGetHandler,
	listHandler mediaListHandler,
	updateConfigHandler mediaUpdateConfigurationHandler,
	getConfigHandler mediaGetConfigurationHandler,
	reanalyzeHandler mediaReanalyzeHandler,
	listStepRunsHandler mediaListStepRunsHandler,
) *MediaHandler {
	return &MediaHandler{
		initiateHandler:     initiateHandler,
		presignHandler:      presignHandler,
		completeHandler:     completeHandler,
		fromURLHandler:      fromURLHandler,
		deleteHandler:       deleteHandler,
		getHandler:          getHandler,
		listHandler:         listHandler,
		updateConfigHandler: updateConfigHandler,
		getConfigHandler:    getConfigHandler,
		reanalyzeHandler:    reanalyzeHandler,
		listStepRunsHandler: listStepRunsHandler,
	}
}

type initiateUploadBody struct {
	Filename  string `json:"filename" validate:"required,min=1,max=512"`
	MimeType  string `json:"mimeType" validate:"omitempty,max=100"`
	SizeBytes int64  `json:"sizeBytes" validate:"omitempty,min=0"`
	PartCount int32  `json:"partCount" validate:"omitempty,min=1,max=10000"`
}

type presignPartsBody struct {
	UploadID    string  `json:"uploadId" validate:"required"`
	PartNumbers []int32 `json:"partNumbers" validate:"required,min=1,dive,min=1"`
}

type completeUploadBody struct {
	UploadID string `json:"uploadId" validate:"required"`
	Parts    []struct {
		PartNumber int32  `json:"partNumber" validate:"required,min=1"`
		ETag       string `json:"etag" validate:"required"`
	} `json:"parts" validate:"required,min=1,dive"`
}

type fromURLBody struct {
	URL string `json:"url" validate:"required,url"`
}

func (h *MediaHandler) InitiateUpload(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var body initiateUploadBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	if body.PartCount == 0 {
		body.PartCount = 1
	}
	result, err := h.initiateHandler.Handle(c.Context(), cmdmedia.InitiateUploadCommand{
		ProjectID: projectID,
		Filename:  body.Filename,
		MimeType:  body.MimeType,
		SizeBytes: body.SizeBytes,
		PartCount: body.PartCount,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to initiate upload"})
	}
	return c.Status(fiber.StatusCreated).JSON(presenter.NewInitiateUploadResponse(result))
}

func (h *MediaHandler) PresignParts(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	var body presignPartsBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	parts, err := h.presignHandler.Handle(c.Context(), cmdmedia.PresignPartsCommand{
		MediaID:     mediaID,
		UploadID:    body.UploadID,
		PartNumbers: body.PartNumbers,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to presign parts"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"parts": presenter.NewUploadPartsResponse(parts)})
}

func (h *MediaHandler) CompleteUpload(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	var body completeUploadBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	parts := make([]port.CompletedUploadPart, 0, len(body.Parts))
	for _, p := range body.Parts {
		parts = append(parts, port.CompletedUploadPart{PartNumber: p.PartNumber, ETag: p.ETag})
	}
	m, err := h.completeHandler.Handle(c.Context(), cmdmedia.CompleteUploadCommand{
		MediaID:  mediaID,
		UploadID: body.UploadID,
		Parts:    parts,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to complete upload"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewMediaResponseFromEntity(m))
}

func (h *MediaHandler) FromURL(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var body fromURLBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	if err := pipeline.ValidatePublicHTTPURL(body.URL); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid or blocked url"})
	}
	m, err := h.fromURLHandler.Handle(c.Context(), cmdmedia.FromURLCommand{
		ProjectID: projectID,
		URL:       body.URL,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create media from url"})
	}
	return c.Status(fiber.StatusCreated).JSON(presenter.NewMediaResponseFromEntity(m))
}

func (h *MediaHandler) ListByProject(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var listQuery paginate.PaginateQuery
	if err := c.Bind().Query(&listQuery); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid query parameters",
			"errors":  err.Error(),
		})
	}
	sortBy, orderBy := listQuery.SortBy, listQuery.OrderBy
	listQuery.Normalize()
	if sortBy == "" {
		listQuery.SortBy = "position"
	}
	if orderBy == "" {
		listQuery.OrderBy = paginate.OrderByAsc
	}
	views, total, err := h.listHandler.Handle(c.Context(), querymedia.ListMediaQuery{
		ProjectID: projectID,
		Query:     listQuery,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list media"})
	}
	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewMediaListResponseFromViews(views),
		int(total),
		listQuery,
	))
}

func (h *MediaHandler) GetByID(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	view, err := h.getHandler.Handle(c.Context(), querymedia.GetMediaQuery{ID: id})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get media"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewMediaResponseFromView(*view))
}

func (h *MediaHandler) Delete(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	if err := h.deleteHandler.Handle(c.Context(), cmdmedia.DeleteMediaCommand{ID: id}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to delete media"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type updateConfigurationBody struct {
	SilenceEnabled       *bool    `json:"silenceEnabled"`
	ThresholdMode        *string  `json:"thresholdMode" validate:"omitempty,oneof=auto manual"`
	SilenceThresholdDB   *float64 `json:"silenceThresholdDb"`
	DetectionLevel       *string  `json:"detectionLevel" validate:"omitempty,oneof=gentle standard aggressive"`
	SilenceMinDurationMs *int     `json:"silenceMinDurationMs" validate:"omitempty,min=0"`
	PaddingBeforeMs      *int     `json:"paddingBeforeMs" validate:"omitempty,min=0"`
	PaddingAfterMs       *int     `json:"paddingAfterMs" validate:"omitempty,min=0"`
	SpeechMinDurationMs  *int     `json:"speechMinDurationMs" validate:"omitempty,min=0"`
	FillersEnabled       *bool    `json:"fillersEnabled"`
	RepetitionsEnabled   *bool    `json:"repetitionsEnabled"`
	FalseStartsEnabled *bool   `json:"falseStartsEnabled"`
	MasterSplit        *string `json:"masterSplit" validate:"omitempty,max=64"`
	BurnSubtitles        *bool    `json:"burnSubtitles"`
	// Empty string clears the preset; non-empty must be a UUID (validated below).
	OutputPresetID *string `json:"outputPresetId" validate:"omitempty"`
}

type reanalyzeBody struct {
	Steps []string `json:"steps" validate:"omitempty,dive,oneof=silence text llm scenes reframe"`
}

func (h *MediaHandler) ListSteps(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	out, err := h.listStepRunsHandler.Handle(c.Context(), querymedia.ListStepRunsQuery{MediaID: id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list media steps"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewMediaStepsResponse(out.Steps, out.Runs))
}

func (h *MediaHandler) GetConfiguration(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	cfg, err := h.getConfigHandler.Handle(c.Context(), querymedia.GetConfigurationQuery{MediaID: id})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Configuration not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get configuration"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewMediaConfigurationResponse(cfg))
}

func (h *MediaHandler) UpdateConfiguration(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	var body updateConfigurationBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	var presetID *uuid.UUID
	clearPreset := false
	if body.OutputPresetID != nil {
		if *body.OutputPresetID == "" {
			clearPreset = true
		} else {
			parsed, err := uuid.Parse(*body.OutputPresetID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid outputPresetId"})
			}
			presetID = &parsed
		}
	}
	cfg, err := h.updateConfigHandler.Handle(c.Context(), cmdmedia.UpdateConfigurationCommand{
		MediaID:              id,
		SilenceEnabled:       body.SilenceEnabled,
		ThresholdMode:        body.ThresholdMode,
		SilenceThresholdDB:   body.SilenceThresholdDB,
		DetectionLevel:       body.DetectionLevel,
		SilenceMinDurationMs: body.SilenceMinDurationMs,
		PaddingBeforeMs:      body.PaddingBeforeMs,
		PaddingAfterMs:       body.PaddingAfterMs,
		SpeechMinDurationMs:  body.SpeechMinDurationMs,
		FillersEnabled:       body.FillersEnabled,
		RepetitionsEnabled:   body.RepetitionsEnabled,
		FalseStartsEnabled: body.FalseStartsEnabled,
		MasterSplit:        body.MasterSplit,
		BurnSubtitles:        body.BurnSubtitles,
		OutputPresetID:       presetID,
		ClearOutputPresetID:  clearPreset,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		if strings.Contains(err.Error(), "media is not ready") || strings.Contains(err.Error(), "media is not editable") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "masterSplit") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update configuration"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewMediaConfigurationResponse(cfg))
}

func (h *MediaHandler) Reanalyze(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	var body reanalyzeBody
	if len(c.Body()) > 0 {
		if err := validation.BindBody(c, &body); err != nil {
			return err
		}
	}
	if err := h.reanalyzeHandler.Handle(c.Context(), cmdmedia.ReanalyzeCommand{
		MediaID: id,
		Steps:   body.Steps,
	}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		if strings.Contains(err.Error(), "media is not editable") || strings.Contains(err.Error(), "media is not ready") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		if strings.Contains(err.Error(), "not ready") || strings.Contains(err.Error(), "unsupported") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to reanalyze"})
	}
	return c.SendStatus(fiber.StatusAccepted)
}
