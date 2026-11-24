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
	initiateHandler mediaInitiateUploadHandler
	presignHandler  mediaPresignPartsHandler
	completeHandler mediaCompleteUploadHandler
	fromURLHandler  mediaFromURLHandler
	deleteHandler   mediaDeleteHandler
	getHandler      mediaGetHandler
	listHandler     mediaListHandler
}

func NewMediaHandler(
	initiateHandler mediaInitiateUploadHandler,
	presignHandler mediaPresignPartsHandler,
	completeHandler mediaCompleteUploadHandler,
	fromURLHandler mediaFromURLHandler,
	deleteHandler mediaDeleteHandler,
	getHandler mediaGetHandler,
	listHandler mediaListHandler,
) *MediaHandler {
	return &MediaHandler{
		initiateHandler: initiateHandler,
		presignHandler:  presignHandler,
		completeHandler: completeHandler,
		fromURLHandler:  fromURLHandler,
		deleteHandler:   deleteHandler,
		getHandler:      getHandler,
		listHandler:     listHandler,
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
