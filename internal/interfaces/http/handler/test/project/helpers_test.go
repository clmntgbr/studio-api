package projecttest

import (
	"context"
	"time"

	cmdproject "go-api/internal/application/command/project"
	queryproject "go-api/internal/application/query/project"
	domainproject "go-api/internal/domain/project"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
)

type mockCreate struct {
	calls int
	last  cmdproject.CreateProjectCommand
	out   *domainproject.Project
	err   error
}

func (m *mockCreate) Handle(_ context.Context, cmd cmdproject.CreateProjectCommand) (*domainproject.Project, error) {
	m.calls++
	m.last = cmd
	return m.out, m.err
}

type mockUpdate struct {
	calls int
	err   error
	out   *domainproject.Project
}

func (m *mockUpdate) Handle(_ context.Context, _ cmdproject.UpdateProjectCommand) (*domainproject.Project, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.out, nil
}

type mockDelete struct {
	calls int
	err   error
}

func (m *mockDelete) Handle(_ context.Context, _ cmdproject.DeleteProjectCommand) error {
	m.calls++
	return m.err
}

type mockGet struct {
	calls int
	view  *domainproject.View
	err   error
}

func (m *mockGet) Handle(_ context.Context, _ queryproject.GetProjectQuery) (*domainproject.View, error) {
	m.calls++
	return m.view, m.err
}

type mockList struct {
	calls int
	views []domainproject.View
	total int64
	err   error
}

func (m *mockList) Handle(_ context.Context, _ queryproject.ListProjectsQuery) ([]domainproject.View, int64, error) {
	m.calls++
	return m.views, m.total, m.err
}

func sampleProject() *domainproject.Project {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	return &domainproject.Project{
		ID:        testutil.TestProjectID,
		Name:      "Demo",
		Status:    domainproject.StatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func sampleView() *domainproject.View {
	p := sampleProject()
	return &domainproject.View{
		ID: p.ID, Name: p.Name, Status: p.Status, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func mount(h *handler.ProjectHandler, withUser bool) *fiber.App {
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Post("/api/projects", auth, h.Create)
		app.Get("/api/projects", auth, h.List)
		app.Get("/api/projects/:id", auth, h.GetByID)
		app.Patch("/api/projects/:id", auth, h.Update)
		app.Delete("/api/projects/:id", auth, h.Delete)
		return app
	}
	app.Post("/api/projects", h.Create)
	app.Get("/api/projects", h.List)
	app.Get("/api/projects/:id", h.GetByID)
	app.Patch("/api/projects/:id", h.Update)
	app.Delete("/api/projects/:id", h.Delete)
	return app
}

func newHandler(create *mockCreate, update *mockUpdate, del *mockDelete, get *mockGet, list *mockList) *handler.ProjectHandler {
	if create == nil {
		create = &mockCreate{}
	}
	if update == nil {
		update = &mockUpdate{}
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
	return handler.NewProjectHandler(create, update, del, get, list)
}
