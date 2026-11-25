package rendertest

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	cmdrender "go-api/internal/application/command/render"
	queryrender "go-api/internal/application/query/render"
	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var testRenderID = uuid.MustParse("01960000-0000-7000-8000-0000000000b1")
var testPresetID = uuid.MustParse("01960000-0000-7000-8000-0000000000a1")

type mockCreate struct {
	calls int
	last  cmdrender.CreateCommand
	out   *domainrender.Render
	err   error
}

func (m *mockCreate) Handle(_ context.Context, cmd cmdrender.CreateCommand) (*domainrender.Render, error) {
	m.calls++
	m.last = cmd
	if m.out != nil || m.err != nil {
		return m.out, m.err
	}
	return sampleRender(), nil
}

type mockCancel struct {
	calls int
	err   error
}

func (m *mockCancel) Handle(_ context.Context, _ cmdrender.CancelCommand) (*domainrender.Render, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	r := sampleRender()
	r.Status = domainrender.StatusCancelled
	return r, nil
}

type mockGet struct {
	out *domainrender.Render
	err error
}

func (m *mockGet) Handle(_ context.Context, _ queryrender.GetQuery) (*domainrender.Render, error) {
	if m.out != nil || m.err != nil {
		return m.out, m.err
	}
	return sampleRender(), nil
}

type mockList struct {
	items []domainrender.Render
	err   error
}

func (m *mockList) Handle(_ context.Context, _ queryrender.ListQuery) ([]domainrender.Render, error) {
	if m.items != nil || m.err != nil {
		return m.items, m.err
	}
	return []domainrender.Render{*sampleRender()}, nil
}

type mockDownload struct {
	out *queryrender.DownloadResult
	err error
}

func (m *mockDownload) Handle(_ context.Context, _ queryrender.DownloadQuery) (*queryrender.DownloadResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.out != nil {
		return m.out, nil
	}
	return &queryrender.DownloadResult{
		Stream: &port.ObjectStream{
			Body:          io.NopCloser(strings.NewReader("fake-mp4")),
			ContentType:   "video/mp4",
			ContentLength: 8,
			StatusCode:    200,
		},
		Filename: "render-" + testRenderID.String() + ".mp4",
	}, nil
}

func sampleRender() *domainrender.Render {
	return &domainrender.Render{
		ID:             testRenderID,
		ProjectID:      testutil.TestProjectID,
		TimelineID:     testutil.TestTimelineID,
		Kind:           domainrender.KindMaster,
		Strategy:       domainrender.StrategyFullRender,
		OutputPresetID: testPresetID,
		Plan:           json.RawMessage(`{"renderId":"` + testRenderID.String() + `"}`),
		Fingerprint:    "fp",
		Status:         domainrender.StatusQueued,
		CreatedAt:      time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
	}
}

func mount(create *mockCreate, cancel *mockCancel, get *mockGet, list *mockList, download *mockDownload, withUser bool) *fiber.App {
	if create == nil {
		create = &mockCreate{}
	}
	if cancel == nil {
		cancel = &mockCancel{}
	}
	if get == nil {
		get = &mockGet{}
	}
	if list == nil {
		list = &mockList{}
	}
	if download == nil {
		download = &mockDownload{}
	}
	h := handler.NewRenderHandler(create, cancel, get, list, download)
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Post("/api/projects/:id/renders", auth, h.Create)
		app.Get("/api/projects/:id/renders", auth, h.ListByProject)
		app.Get("/api/renders/:id", auth, h.GetByID)
		app.Post("/api/renders/:id/cancel", auth, h.Cancel)
		app.Get("/api/renders/:id/download", auth, h.Download)
		return app
	}
	app.Post("/api/projects/:id/renders", h.Create)
	app.Get("/api/projects/:id/renders", h.ListByProject)
	app.Get("/api/renders/:id", h.GetByID)
	app.Post("/api/renders/:id/cancel", h.Cancel)
	app.Get("/api/renders/:id/download", h.Download)
	return app
}
