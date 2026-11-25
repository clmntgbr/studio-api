package playbacktest

import (
	"context"
	"io"
	"strings"

	querymedia "go-api/internal/application/query/media"
	"go-api/internal/domain/port"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
)

type mockPlayback struct {
	calls int
	last  querymedia.PlaybackQuery
	out   *querymedia.PlaybackResult
	err   error
}

func (m *mockPlayback) Handle(_ context.Context, q querymedia.PlaybackQuery) (*querymedia.PlaybackResult, error) {
	m.calls++
	m.last = q
	if m.err != nil {
		return nil, m.err
	}
	if m.out != nil {
		return m.out, nil
	}
	return &querymedia.PlaybackResult{
		URL:       "https://cdn.example/proxy.mp4",
		Kind:      "proxy",
		ExpiresIn: 3600,
	}, nil
}

type mockStream struct {
	calls int
	last  querymedia.StreamQuery
	out   *querymedia.StreamResult
	err   error
}

func (m *mockStream) Handle(_ context.Context, q querymedia.StreamQuery) (*querymedia.StreamResult, error) {
	m.calls++
	m.last = q
	if m.err != nil {
		return nil, m.err
	}
	if m.out != nil {
		return m.out, nil
	}
	return &querymedia.StreamResult{
		Kind: "proxy",
		Stream: &port.ObjectStream{
			Body:          io.NopCloser(strings.NewReader("fake-video")),
			ContentType:   "video/mp4",
			ContentLength: 10,
			StatusCode:    200,
		},
	}, nil
}

func mount(playback *mockPlayback, stream *mockStream, withUser bool) *fiber.App {
	if playback == nil {
		playback = &mockPlayback{}
	}
	// Pass untyped nil so Stream sees a nil interface (not a typed-nil *mockStream).
	var h *handler.PlaybackHandler
	if stream == nil {
		h = handler.NewPlaybackHandler(playback, nil)
	} else {
		h = handler.NewPlaybackHandler(playback, stream)
	}
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Get("/api/media/:id/playback", auth, h.Get)
		app.Get("/api/media/:id/stream", auth, h.Stream)
		return app
	}
	app.Get("/api/media/:id/playback", h.Get)
	app.Get("/api/media/:id/stream", h.Stream)
	return app
}
