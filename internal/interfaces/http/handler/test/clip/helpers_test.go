package cliptest

import (
	"context"
	"encoding/json"
	"time"

	cmdclip "go-api/internal/application/command/clip"
	cmdrender "go-api/internal/application/command/render"
	queryclip "go-api/internal/application/query/clip"
	queryllm "go-api/internal/application/query/llmartifact"
	domainclip "go-api/internal/domain/clip"
	domainllm "go-api/internal/domain/llmartifact"
	domainrender "go-api/internal/domain/render"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var testClipID = uuid.MustParse("01960000-0000-7000-8000-0000000000c1")
var testPresetID = uuid.MustParse("01960000-0000-7000-8000-0000000000a2")

type mockList struct {
	items []domainclip.Clip
	err   error
}

func (m *mockList) Handle(_ context.Context, _ queryclip.ListQuery) ([]domainclip.Clip, error) {
	if m.items != nil || m.err != nil {
		return m.items, m.err
	}
	return []domainclip.Clip{*sampleClip()}, nil
}

type mockCreate struct {
	calls int
	err   error
}

func (m *mockCreate) Handle(_ context.Context, _ cmdclip.CreateCommand) (*domainclip.Clip, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return sampleClip(), nil
}

type mockUpdate struct {
	calls int
	err   error
}

func (m *mockUpdate) Handle(_ context.Context, _ cmdclip.UpdateCommand) (*domainclip.Clip, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	c := sampleClip()
	c.Selected = true
	return c, nil
}

type mockAnalyze struct {
	calls int
	err   error
}

func (m *mockAnalyze) Handle(_ context.Context, _ cmdclip.AnalyzeCommand) error {
	m.calls++
	return m.err
}

type mockArtifacts struct {
	items []domainllm.Artifact
	err   error
}

func (m *mockArtifacts) Handle(_ context.Context, _ queryllm.ListQuery) ([]domainllm.Artifact, error) {
	if m.items != nil || m.err != nil {
		return m.items, m.err
	}
	return []domainllm.Artifact{{
		ID:          uuid.MustParse("01960000-0000-7000-8000-0000000000d1"),
		MediaFileID: testutil.TestMediaID,
		Kind:        "chapters",
		Payload:     json.RawMessage(`{"chapters":[]}`),
		CreatedAt:   time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
	}}, nil
}

type mockGet struct {
	out *domainclip.Clip
	err error
}

func (m *mockGet) Handle(_ context.Context, _ queryclip.GetQuery) (*domainclip.Clip, error) {
	if m.out != nil || m.err != nil {
		return m.out, m.err
	}
	return sampleClip(), nil
}

type mockRender struct {
	calls int
	last  cmdrender.CreateCommand
	err   error
}

func (m *mockRender) Handle(_ context.Context, cmd cmdrender.CreateCommand) (*domainrender.Render, error) {
	m.calls++
	m.last = cmd
	if m.err != nil {
		return nil, m.err
	}
	return &domainrender.Render{
		ID:             uuid.MustParse("01960000-0000-7000-8000-0000000000e1"),
		ProjectID:      testutil.TestProjectID,
		TimelineID:     testutil.TestTimelineID,
		Kind:           domainrender.KindClip,
		Strategy:       domainrender.StrategyFullRender,
		ClipID:         cmd.ClipID,
		OutputPresetID: cmd.PresetID,
		Status:         domainrender.StatusQueued,
		Plan:           json.RawMessage(`{}`),
		CreatedAt:      time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
	}, nil
}

func sampleClip() *domainclip.Clip {
	score := 0.8
	hook := "hook"
	return &domainclip.Clip{
		ID:            testClipID,
		ProjectID:     testutil.TestProjectID,
		MediaFileID:   testutil.TestMediaID,
		SourceStartMs: 1000,
		SourceEndMs:   5000,
		Origin:        "llm",
		Score:         &score,
		Hook:          &hook,
		Hashtags:      []string{},
		Selected:      false,
		CreatedAt:     time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
	}
}

func mount(list *mockList, create *mockCreate, update *mockUpdate, analyze *mockAnalyze, artifacts *mockArtifacts, get *mockGet, render *mockRender, withUser bool) *fiber.App {
	if list == nil {
		list = &mockList{}
	}
	if create == nil {
		create = &mockCreate{}
	}
	if update == nil {
		update = &mockUpdate{}
	}
	if analyze == nil {
		analyze = &mockAnalyze{}
	}
	if artifacts == nil {
		artifacts = &mockArtifacts{}
	}
	if get == nil {
		get = &mockGet{}
	}
	if render == nil {
		render = &mockRender{}
	}
	h := handler.NewClipHandler(list, create, update, analyze, artifacts, get, render)
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Get("/api/projects/:id/clips", auth, h.ListByProject)
		app.Post("/api/projects/:id/clips", auth, h.Create)
		app.Post("/api/projects/:id/clips/analyze", auth, h.Analyze)
		app.Patch("/api/clips/:id", auth, h.Update)
		app.Post("/api/clips/:id/render", auth, h.Render)
		app.Get("/api/media/:id/llm-artifacts", auth, h.ListArtifactsByMedia)
		return app
	}
	app.Get("/api/projects/:id/clips", h.ListByProject)
	app.Post("/api/projects/:id/clips", h.Create)
	app.Post("/api/projects/:id/clips/analyze", h.Analyze)
	app.Patch("/api/clips/:id", h.Update)
	app.Post("/api/clips/:id/render", h.Render)
	app.Get("/api/media/:id/llm-artifacts", h.ListArtifactsByMedia)
	return app
}
