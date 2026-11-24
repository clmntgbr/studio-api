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

func newHandler(
	init *mockInitiate,
	presign *mockPresign,
	complete *mockComplete,
	fromURL *mockFromURL,
	del *mockDelete,
	get *mockGet,
	list *mockList,
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
	return handler.NewMediaHandler(init, presign, complete, fromURL, del, get, list)
}

func mount(h *handler.MediaHandler, withUser bool) *fiber.App {
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Post("/api/projects/:id/media/uploads", auth, h.InitiateUpload)
		app.Post("/api/projects/:id/media/from-url", auth, h.FromURL)
		app.Get("/api/projects/:id/media", auth, h.ListByProject)
		app.Get("/api/media/:id", auth, h.GetByID)
		app.Delete("/api/media/:id", auth, h.Delete)
		app.Post("/api/media/:id/uploads/parts", auth, h.PresignParts)
		app.Post("/api/media/:id/uploads/complete", auth, h.CompleteUpload)
		return app
	}
	app.Post("/api/projects/:id/media/uploads", h.InitiateUpload)
	app.Post("/api/projects/:id/media/from-url", h.FromURL)
	app.Get("/api/projects/:id/media", h.ListByProject)
	app.Get("/api/media/:id", h.GetByID)
	app.Delete("/api/media/:id", h.Delete)
	app.Post("/api/media/:id/uploads/parts", h.PresignParts)
	app.Post("/api/media/:id/uploads/complete", h.CompleteUpload)
	return app
}

var _ = uuid.Nil
