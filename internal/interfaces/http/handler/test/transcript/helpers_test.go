package transcripttest

import (
	"context"
	"time"

	querytranscript "go-api/internal/application/query/transcript"
	domaintranscript "go-api/internal/domain/transcript"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
)

type mockGet struct {
	calls int
	last  querytranscript.GetTranscriptQuery
	view  *querytranscript.TranscriptView
	err   error
}

func (m *mockGet) Handle(_ context.Context, q querytranscript.GetTranscriptQuery) (*querytranscript.TranscriptView, error) {
	m.calls++
	m.last = q
	return m.view, m.err
}

type mockGetSRT struct {
	calls int
	out   string
	err   error
}

func (m *mockGetSRT) Handle(_ context.Context, _ querytranscript.GetSRTQuery) (string, error) {
	m.calls++
	return m.out, m.err
}

func sampleView() *querytranscript.TranscriptView {
	lang := "fr"
	text := "bonjour le monde"
	now := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	conf := 0.98
	return &querytranscript.TranscriptView{
		Transcript: &domaintranscript.Transcript{
			ID:          testutil.TestMediaID,
			MediaFileID: testutil.TestMediaID,
			Language:    &lang,
			Text:        &text,
			Provider:    "assemblyai",
			Status:      domaintranscript.StatusCompleted,
			CreatedAt:   now,
		},
		Words: []domaintranscript.Word{{
			TranscriptID:  testutil.TestMediaID,
			WordIndex:     0,
			Text:          "bonjour",
			SourceStartMs: 0,
			SourceEndMs:   400,
			Confidence:    &conf,
		}, {
			TranscriptID:  testutil.TestMediaID,
			WordIndex:     1,
			Text:          "monde",
			SourceStartMs: 500,
			SourceEndMs:   900,
			Confidence:    &conf,
		}},
	}
}

func mount(get *mockGet, withUser bool) *fiber.App {
	return mountFull(get, nil, withUser)
}

func mountFull(get *mockGet, srt *mockGetSRT, withUser bool) *fiber.App {
	if get == nil {
		get = &mockGet{}
	}
	if srt == nil {
		srt = &mockGetSRT{}
	}
	h := handler.NewTranscriptHandler(get, srt)
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Get("/api/media/:id/transcript", auth, h.GetByMedia)
		app.Get("/api/media/:id/transcript.srt", auth, h.GetSRT)
		return app
	}
	app.Get("/api/media/:id/transcript", h.GetByMedia)
	app.Get("/api/media/:id/transcript.srt", h.GetSRT)
	return app
}
