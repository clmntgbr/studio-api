package mediatest

import (
	"context"
	"time"

	cmdmedia "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type mockInitiate struct {
	calls int
	out   *cmdmedia.InitiateUploadResult
	err   error
}

func (m *mockInitiate) Handle(_ context.Context, _ cmdmedia.InitiateUploadCommand) (*cmdmedia.InitiateUploadResult, error) {
	m.calls++
	return m.out, m.err
}

type mockPresign struct {
	calls int
	out   []port.UploadPartURL
	err   error
}

func (m *mockPresign) Handle(_ context.Context, _ cmdmedia.PresignPartsCommand) ([]port.UploadPartURL, error) {
	m.calls++
	return m.out, m.err
}

type mockComplete struct {
	calls int
	out   *domainmedia.MediaFile
	err   error
}

func (m *mockComplete) Handle(_ context.Context, _ cmdmedia.CompleteUploadCommand) (*domainmedia.MediaFile, error) {
	m.calls++
	return m.out, m.err
}

type mockFromURL struct {
	calls int
	out   *domainmedia.MediaFile
	err   error
}

func (m *mockFromURL) Handle(_ context.Context, _ cmdmedia.FromURLCommand) (*domainmedia.MediaFile, error) {
	m.calls++
	return m.out, m.err
}

type mockDelete struct {
	calls int
	err   error
}

func (m *mockDelete) Handle(_ context.Context, _ cmdmedia.DeleteMediaCommand) error {
	m.calls++
	return m.err
}

type mockGet struct {
	calls int
	view  *domainmedia.FileView
	err   error
}

func (m *mockGet) Handle(_ context.Context, _ querymedia.GetMediaQuery) (*domainmedia.FileView, error) {
	m.calls++
	return m.view, m.err
}

type mockList struct {
	calls int
	views []domainmedia.FileView
	total int64
	err   error
}

func (m *mockList) Handle(_ context.Context, _ querymedia.ListMediaQuery) ([]domainmedia.FileView, int64, error) {
	m.calls++
	return m.views, m.total, m.err
}

type mockUpdateConfig struct {
	calls int
	cmd   cmdmedia.UpdateConfigurationCommand
	out   *domainmedia.Configuration
	err   error
}

func (m *mockUpdateConfig) Handle(_ context.Context, cmd cmdmedia.UpdateConfigurationCommand) (*domainmedia.Configuration, error) {
	m.calls++
	m.cmd = cmd
	return m.out, m.err
}

type mockGetConfig struct {
	calls int
	out   *domainmedia.Configuration
	err   error
}

func (m *mockGetConfig) Handle(_ context.Context, _ querymedia.GetConfigurationQuery) (*domainmedia.Configuration, error) {
	m.calls++
	return m.out, m.err
}

type mockReanalyze struct {
	calls int
	cmd   cmdmedia.ReanalyzeCommand
	err   error
}

func (m *mockReanalyze) Handle(_ context.Context, cmd cmdmedia.ReanalyzeCommand) error {
	m.calls++
	m.cmd = cmd
	return m.err
}

type mockListStepRuns struct {
	calls int
	out   *querymedia.ListStepRunsResult
	err   error
}

func (m *mockListStepRuns) Handle(_ context.Context, _ querymedia.ListStepRunsQuery) (*querymedia.ListStepRunsResult, error) {
	m.calls++
	return m.out, m.err
}

func sampleMedia() *domainmedia.MediaFile {
	now := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	key := "media/" + testutil.TestMediaID.String() + "/source/original.mp4"
	fn := "clip.mp4"
	return &domainmedia.MediaFile{
		ID:               testutil.TestMediaID,
		ProjectID:        testutil.TestProjectID,
		SourceKind:       domainmedia.SourceKindUpload,
		OriginalFilename: &fn,
		StorageKey:       &key,
		Status:           domainmedia.StatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func sampleConfig() *domainmedia.Configuration {
	cfg := domainmedia.DefaultConfiguration(testutil.TestMediaID)
	cfg.UpdatedAt = time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	return cfg
}

func newHandler(
	init *mockInitiate,
	presign *mockPresign,
	complete *mockComplete,
	fromURL *mockFromURL,
	del *mockDelete,
	get *mockGet,
	list *mockList,
) *handler.MediaHandler {
	return newHandlerWithConfig(init, presign, complete, fromURL, del, get, list, nil, nil, nil, nil)
}

func newHandlerWithConfig(
	init *mockInitiate,
	presign *mockPresign,
	complete *mockComplete,
	fromURL *mockFromURL,
	del *mockDelete,
	get *mockGet,
	list *mockList,
	updateCfg *mockUpdateConfig,
	getCfg *mockGetConfig,
	reanalyze *mockReanalyze,
	listSteps *mockListStepRuns,
) *handler.MediaHandler {
	if init == nil {
		init = &mockInitiate{}
	}
	if presign == nil {
		presign = &mockPresign{}
	}
	if complete == nil {
		complete = &mockComplete{}
	}
	if fromURL == nil {
		fromURL = &mockFromURL{}
	}
	if del == nil {
		del = &mockDelete{}
	}
	if get == nil {
		get = &mockGet{}
	}
	if list == nil {
		list = &mockList{}
	}
	if updateCfg == nil {
		updateCfg = &mockUpdateConfig{}
	}
	if getCfg == nil {
		getCfg = &mockGetConfig{}
	}
	if reanalyze == nil {
		reanalyze = &mockReanalyze{}
	}
	if listSteps == nil {
		listSteps = &mockListStepRuns{}
	}
	return handler.NewMediaHandler(init, presign, complete, fromURL, del, get, list, updateCfg, getCfg, reanalyze, listSteps)
}

func mount(h *handler.MediaHandler, withUser bool) *fiber.App {
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	register := func(method, path string, fn fiber.Handler) {
		if withUser {
			app.Add([]string{method}, path, auth, fn)
			return
		}
		app.Add([]string{method}, path, fn)
	}
	register(fiber.MethodPost, "/api/projects/:id/media/uploads", h.InitiateUpload)
	register(fiber.MethodPost, "/api/projects/:id/media/from-url", h.FromURL)
	register(fiber.MethodGet, "/api/projects/:id/media", h.ListByProject)
	register(fiber.MethodGet, "/api/media/:id", h.GetByID)
	register(fiber.MethodDelete, "/api/media/:id", h.Delete)
	register(fiber.MethodPost, "/api/media/:id/uploads/parts", h.PresignParts)
	register(fiber.MethodPost, "/api/media/:id/uploads/complete", h.CompleteUpload)
	register(fiber.MethodGet, "/api/media/:id/configuration", h.GetConfiguration)
	register(fiber.MethodPatch, "/api/media/:id/configuration", h.UpdateConfiguration)
	register(fiber.MethodPost, "/api/media/:id/reanalyze", h.Reanalyze)
	register(fiber.MethodGet, "/api/media/:id/steps", h.ListSteps)
	return app
}

var _ = uuid.Nil
