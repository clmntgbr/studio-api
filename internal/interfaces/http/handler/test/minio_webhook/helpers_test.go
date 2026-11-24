package miniowebhooktest

import (
	"context"

	cmdmedia "go-api/internal/application/command/media"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/middleware"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const testWebhookSecret = "test-minio-webhook-secret"

type mockMarkReady struct {
	calls []cmdmedia.MarkSourceReadyCommand
	err   error
}

func (m *mockMarkReady) Handle(_ context.Context, cmd cmdmedia.MarkSourceReadyCommand) error {
	m.calls = append(m.calls, cmd)
	return m.err
}

func mount(mark *mockMarkReady, withAuth bool) *fiber.App {
	if mark == nil {
		mark = &mockMarkReady{}
	}
	h := handler.NewMinioWebhookHandler(mark, "media")
	app := testutil.NewTestApp()
	if withAuth {
		mw := middleware.NewMinioWebhookMiddleware(testWebhookSecret)
		app.Post("/webhooks/minio/object-created", mw.Protected(), h.ObjectCreated)
		return app
	}
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)
	return app
}

func sampleObjectCreatedBody(mediaID uuid.UUID) []byte {
	key := "media/" + mediaID.String() + "/source/original.mp4"
	return []byte(`{
  "Records": [{
    "eventName": "s3:ObjectCreated:CompleteMultipartUpload",
    "s3": {
      "bucket": {"name": "media"},
      "object": {"key": "` + key + `", "size": 1024}
    }
  }]
}`)
}
