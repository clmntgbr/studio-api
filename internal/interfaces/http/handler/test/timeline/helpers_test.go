package timelinetest

import (
	"context"
	"time"

	cmdtimeline "go-api/internal/application/command/timeline"
	querytimeline "go-api/internal/application/query/timeline"
	domaintimeline "go-api/internal/domain/timeline"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"
	eng "go-api/internal/timeline"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type mockPreview struct {
	calls int
	cmd   cmdtimeline.PreviewCommand
	out   *cmdtimeline.PreviewResult
	err   error
}

func (m *mockPreview) Handle(_ context.Context, cmd cmdtimeline.PreviewCommand) (*cmdtimeline.PreviewResult, error) {
	m.calls++
	m.cmd = cmd
	return m.out, m.err
}

type mockValidate struct {
	calls int
	cmd   cmdtimeline.ValidateCommand
	out   *domaintimeline.Timeline
	err   error
}

func (m *mockValidate) Handle(_ context.Context, cmd cmdtimeline.ValidateCommand) (*domaintimeline.Timeline, error) {
	m.calls++
	m.cmd = cmd
	return m.out, m.err
}

type mockGet struct {
	calls int
	view  *querytimeline.TimelineView
	err   error
}

func (m *mockGet) Handle(_ context.Context, _ querytimeline.GetTimelineQuery) (*querytimeline.TimelineView, error) {
	m.calls++
	return m.view, m.err
}

type mockListVersions struct {
	calls int
	items []domaintimeline.Timeline
	err   error
}

func (m *mockListVersions) Handle(_ context.Context, _ querytimeline.ListVersionsQuery) ([]domaintimeline.Timeline, error) {
	m.calls++
	return m.items, m.err
}

type mockGetSubtitles struct {
	calls       int
	last        querytimeline.GetSubtitlesQuery
	body        string
	contentType string
	err         error
}

func (m *mockGetSubtitles) Handle(_ context.Context, q querytimeline.GetSubtitlesQuery) (string, string, error) {
	m.calls++
	m.last = q
	ct := m.contentType
	if ct == "" {
		ct = "text/x-ssa"
	}
	return m.body, ct, m.err
}

type mockGetExport struct {
	calls    int
	last     querytimeline.GetExportQuery
	body     string
	ctype    string
	filename string
	err      error
}

func (m *mockGetExport) Handle(_ context.Context, q querytimeline.GetExportQuery) (string, string, string, error) {
	m.calls++
	m.last = q
	ct := m.ctype
	if ct == "" {
		ct = "application/xml"
	}
	fn := m.filename
	if fn == "" {
		fn = "timeline." + q.Format
	}
	return m.body, ct, fn, m.err
}

func sampleTimeline() *domaintimeline.Timeline {
	now := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	return &domaintimeline.Timeline{
		ID:          testutil.TestTimelineID,
		ProjectID:   testutil.TestProjectID,
		Version:     1,
		Status:      domaintimeline.StatusDraft,
		Fingerprint: "fp-1",
		DurationMs:  10000,
		CreatedAt:   now,
	}
}

func sampleView() *querytimeline.TimelineView {
	t := sampleTimeline()
	return &querytimeline.TimelineView{
		Timeline: t,
		Segments: []domaintimeline.Segment{{
			ID:            uuid.New(),
			TimelineID:    t.ID,
			SegmentIndex:  0,
			MediaFileID:   testutil.TestMediaID,
			SourceStartMs: 0,
			SourceEndMs:   10000,
			OutputStartMs: 0,
			OutputEndMs:   10000,
		}},
		Decisions: nil,
	}
}

func samplePreview() *cmdtimeline.PreviewResult {
	return &cmdtimeline.PreviewResult{
		Output: eng.Output{
			Fingerprint: "preview-fp",
			DurationMs:  9000,
			Segments: []eng.Segment{{
				SegmentIndex:  0,
				MediaFileID:   testutil.TestMediaID,
				SourceStartMs: 0,
				SourceEndMs:   9000,
				OutputStartMs: 0,
				OutputEndMs:   9000,
			}},
		},
	}
}

func newHandler(preview *mockPreview, validate *mockValidate, get *mockGet) *handler.TimelineHandler {
	return newHandlerFull(preview, validate, get, nil, nil, nil)
}

func newHandlerFull(
	preview *mockPreview,
	validate *mockValidate,
	get *mockGet,
	list *mockListVersions,
	subs *mockGetSubtitles,
	exp *mockGetExport,
) *handler.TimelineHandler {
	if preview == nil {
		preview = &mockPreview{}
	}
	if validate == nil {
		validate = &mockValidate{}
	}
	if get == nil {
		get = &mockGet{}
	}
	if list == nil {
		list = &mockListVersions{}
	}
	if subs == nil {
		subs = &mockGetSubtitles{}
	}
	if exp == nil {
		exp = &mockGetExport{}
	}
	return handler.NewTimelineHandler(preview, validate, get, list, subs, exp)
}

func mount(h *handler.TimelineHandler, withUser bool) *fiber.App {
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Get("/api/projects/:id/timeline", auth, h.Get)
		app.Get("/api/projects/:id/timeline/versions", auth, h.ListVersions)
		app.Get("/api/projects/:id/timeline/subtitles.ass", auth, h.GetSubtitlesASS)
		app.Get("/api/projects/:id/timeline/subtitles.srt", auth, h.GetSubtitlesSRT)
		app.Get("/api/projects/:id/timeline/export.fcpxml", auth, h.GetExportFCPXML)
		app.Get("/api/projects/:id/timeline/export.edl", auth, h.GetExportEDL)
		app.Get("/api/projects/:id/timeline/export.otio", auth, h.GetExportOTIO)
		app.Get("/api/projects/:id/timeline/export.premiere.xml", auth, h.GetExportPremiere)
		app.Post("/api/projects/:id/timeline/preview", auth, h.Preview)
		app.Post("/api/projects/:id/timeline/validate", auth, h.Validate)
		return app
	}
	app.Get("/api/projects/:id/timeline", h.Get)
	app.Get("/api/projects/:id/timeline/versions", h.ListVersions)
	app.Get("/api/projects/:id/timeline/subtitles.ass", h.GetSubtitlesASS)
	app.Get("/api/projects/:id/timeline/subtitles.srt", h.GetSubtitlesSRT)
	app.Get("/api/projects/:id/timeline/export.fcpxml", h.GetExportFCPXML)
	app.Get("/api/projects/:id/timeline/export.edl", h.GetExportEDL)
	app.Get("/api/projects/:id/timeline/export.otio", h.GetExportOTIO)
	app.Get("/api/projects/:id/timeline/export.premiere.xml", h.GetExportPremiere)
	app.Post("/api/projects/:id/timeline/preview", h.Preview)
	app.Post("/api/projects/:id/timeline/validate", h.Validate)
	return app
}
