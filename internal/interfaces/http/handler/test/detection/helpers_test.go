package detectiontest

import (
	"context"
	"encoding/json"
	"time"

	querydetection "go-api/internal/application/query/detection"
	domaindetection "go-api/internal/domain/detection"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
)

type mockList struct {
	calls int
	items []domaindetection.Detection
	err   error
}

func (m *mockList) Handle(_ context.Context, _ querydetection.ListDetectionsQuery) ([]domaindetection.Detection, error) {
	m.calls++
	return m.items, m.err
}

func sampleDetection() domaindetection.Detection {
	return domaindetection.Detection{
		ID:            testutil.TestMediaID,
		MediaFileID:   testutil.TestMediaID,
		Kind:          domaindetection.KindSilence,
		SourceStartMs: 1000,
		SourceEndMs:   2500,
		Detector:      "ffmpeg.silencedetect",
		Payload:       json.RawMessage(`{"noiseFloorDb":-42.1}`),
		CreatedAt:     time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
	}
}

func mount(list *mockList, withUser bool) *fiber.App {
	if list == nil {
		list = &mockList{}
	}
	h := handler.NewDetectionHandler(list)
	app := testutil.NewTestApp()
	if withUser {
		app.Get("/api/media/:id/detections", testutil.WithUserWithoutProject(testutil.TestUserID), h.ListByMedia)
		return app
	}
	app.Get("/api/media/:id/detections", h.ListByMedia)
	return app
}
