package handler

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"

	cmdmedia "go-api/internal/application/command/media"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Minio object-created notification (S3-compatible). Vendor shape stays at the HTTP boundary.
type minioEventEnvelope struct {
	Records []minioEventRecord `json:"Records"`
	Key     string             `json:"Key"`
}

type minioEventRecord struct {
	EventName string `json:"eventName"`
	S3        struct {
		Bucket struct {
			Name string `json:"name"`
		} `json:"bucket"`
		Object struct {
			Key       string `json:"key"`
			Size      int64  `json:"size"`
			Sequencer string `json:"sequencer"`
		} `json:"object"`
	} `json:"s3"`
}

type MinioWebhookHandler struct {
	markReady minioMarkSourceReadyHandler
	bucket    string
}

func NewMinioWebhookHandler(markReady minioMarkSourceReadyHandler, bucket string) *MinioWebhookHandler {
	return &MinioWebhookHandler{markReady: markReady, bucket: bucket}
}

func (h *MinioWebhookHandler) ObjectCreated(c fiber.Ctx) error {
	var env minioEventEnvelope
	if err := json.Unmarshal(c.Body(), &env); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	handled := 0
	for _, rec := range env.Records {
		if rec.EventName != "" && !strings.Contains(rec.EventName, "ObjectCreated") {
			continue
		}
		key, err := url.QueryUnescape(rec.S3.Object.Key)
		if err != nil {
			key = rec.S3.Object.Key
		}
		key = strings.TrimPrefix(key, "/")
		// MinIO may send "<bucket>/<objectKey>"; object keys already start with "media/…".
		if h.bucket != "" && strings.HasPrefix(key, h.bucket+"/") {
			if stripped := strings.TrimPrefix(key, h.bucket+"/"); parseMediaIDFromSourceKeyOK(stripped) {
				key = stripped
			}
		}

		mediaID, ok := parseMediaIDFromSourceKey(key)
		if !ok {
			continue
		}
		if h.bucket != "" && rec.S3.Bucket.Name != "" && rec.S3.Bucket.Name != h.bucket {
			continue
		}

		var sizePtr *int64
		if rec.S3.Object.Size > 0 {
			sz := rec.S3.Object.Size
			sizePtr = &sz
		}
		if err := h.markReady.Handle(c.Context(), cmdmedia.MarkSourceReadyCommand{
			MediaID:   mediaID,
			SourceKey: key,
			SizeBytes: sizePtr,
		}); err != nil {
			log.Printf("minio webhook mark source ready media=%s: %v", mediaID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to process object created event",
			})
		}
		handled++
	}

	// Empty Records is possible for MinIO test pings — acknowledge.
	_ = handled
	return c.SendStatus(fiber.StatusNoContent)
}

// parseMediaIDFromSourceKey expects media/<uuid>/source/<filename>.
func parseMediaIDFromSourceKey(key string) (uuid.UUID, bool) {
	parts := strings.Split(key, "/")
	if len(parts) < 4 {
		return uuid.Nil, false
	}
	if parts[0] != "media" || parts[2] != "source" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func parseMediaIDFromSourceKeyOK(key string) bool {
	_, ok := parseMediaIDFromSourceKey(key)
	return ok
}
