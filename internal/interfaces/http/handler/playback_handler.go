package handler

import (
	"context"
	"io"
	"strconv"
	"strings"

	querymedia "go-api/internal/application/query/media"
	httpctx "go-api/internal/interfaces/http/context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type mediaPlaybackHandler interface {
	Handle(ctx context.Context, q querymedia.PlaybackQuery) (*querymedia.PlaybackResult, error)
}

type mediaStreamHandler interface {
	Handle(ctx context.Context, q querymedia.StreamQuery) (*querymedia.StreamResult, error)
}

type PlaybackHandler struct {
	playback mediaPlaybackHandler
	stream   mediaStreamHandler
}

func NewPlaybackHandler(playback mediaPlaybackHandler, stream mediaStreamHandler) *PlaybackHandler {
	return &PlaybackHandler{playback: playback, stream: stream}
}

func (h *PlaybackHandler) Get(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	kind := c.Query("kind")
	if kind == "" {
		kind = "auto"
	}
	out, err := h.playback.Handle(c.Context(), querymedia.PlaybackQuery{MediaID: id, Kind: kind})
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		if strings.Contains(msg, "not ready") || strings.Contains(msg, "kind must") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": msg})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get playback URL"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"url":       out.URL,
		"kind":      out.Kind,
		"expiresIn": out.ExpiresIn,
	})
}

// Stream proxies proxy or original source from MinIO through the API (Range-aware).
func (h *PlaybackHandler) Stream(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	if h.stream == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"message": "Stream not configured"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}
	kind := c.Query("kind")
	if kind == "" {
		kind = "auto"
	}

	out, err := h.stream.Handle(c.Context(), querymedia.StreamQuery{
		MediaID:     id,
		Kind:        kind,
		RangeHeader: c.Get("Range"),
	})
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		if strings.Contains(msg, "not ready") || strings.Contains(msg, "kind must") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": msg})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to stream media"})
	}

	stream := out.Stream
	defer stream.Body.Close()

	c.Set("Accept-Ranges", "bytes")
	c.Set("Content-Type", stream.ContentType)
	c.Set("Cache-Control", "private, max-age=60")
	if stream.ContentRange != "" {
		c.Set("Content-Range", stream.ContentRange)
	}
	if stream.ContentLength > 0 {
		c.Set("Content-Length", strconv.FormatInt(stream.ContentLength, 10))
	}

	status := stream.StatusCode
	if status == 0 {
		status = fiber.StatusOK
	}
	c.Status(status)

	_, err = io.Copy(c.Response().BodyWriter(), stream.Body)
	return err
}
