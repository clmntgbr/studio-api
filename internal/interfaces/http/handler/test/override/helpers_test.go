package overridetest

import (
	"context"
	"time"

	cmddetection "go-api/internal/application/command/detection"
	domaindetection "go-api/internal/domain/detection"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
)

type mockCreate struct {
	calls int
	last  cmddetection.CreateOverrideCommand
	out   *domaindetection.Override
	err   error
}

func (m *mockCreate) Handle(_ context.Context, cmd cmddetection.CreateOverrideCommand) (*domaindetection.Override, error) {
	m.calls++
	m.last = cmd
	if m.out != nil || m.err != nil {
		return m.out, m.err
	}
	kind := (*domaindetection.Kind)(nil)
	if cmd.Kind != nil {
		k := domaindetection.Kind(*cmd.Kind)
		kind = &k
	}
	return &domaindetection.Override{
		ID:            testutil.TestOverrideID,
		MediaFileID:   cmd.MediaID,
		Kind:          kind,
		Action:        domaindetection.DecisionAction(cmd.Action),
		SourceStartMs: cmd.SourceStartMs,
		SourceEndMs:   cmd.SourceEndMs,
		CreatedAt:     time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
	}, nil
}

type mockDelete struct {
	calls int
	last  cmddetection.DeleteOverrideCommand
	err   error
}

func (m *mockDelete) Handle(_ context.Context, cmd cmddetection.DeleteOverrideCommand) error {
	m.calls++
	m.last = cmd
	return m.err
}

func mount(create *mockCreate, del *mockDelete, withUser bool) *fiber.App {
	if create == nil {
		create = &mockCreate{}
	}
	if del == nil {
		del = &mockDelete{}
	}
	h := handler.NewOverrideHandler(create, del)
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Post("/api/media/:id/overrides", auth, h.Create)
		app.Delete("/api/overrides/:id", auth, h.Delete)
		return app
	}
	app.Post("/api/media/:id/overrides", h.Create)
	app.Delete("/api/overrides/:id", h.Delete)
	return app
}

func validBody() map[string]any {
	return map[string]any{
		"action":        "keep",
		"sourceStartMs": 1000,
		"sourceEndMs":   2500,
		"kind":          "filler",
	}
}