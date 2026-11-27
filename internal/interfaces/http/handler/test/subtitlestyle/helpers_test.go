package subtitlestyletest

import (
	"context"
	"encoding/json"
	"time"

	cmdstyle "go-api/internal/application/command/subtitlestyle"
	querystyle "go-api/internal/application/query/subtitlestyle"
	domainstyle "go-api/internal/domain/subtitlestyle"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var testStyleID = uuid.MustParse("01960000-0000-7000-8000-000000000006")

type mockCreate struct {
	calls int
	last  cmdstyle.CreateCommand
	out   *domainstyle.Style
	err   error
}

func (m *mockCreate) Handle(_ context.Context, cmd cmdstyle.CreateCommand) (*domainstyle.Style, error) {
	m.calls++
	m.last = cmd
	if m.out != nil || m.err != nil {
		return m.out, m.err
	}
	return sampleStyle(cmd.Name), nil
}

type mockUpdate struct {
	calls int
	last  cmdstyle.UpdateCommand
	out   *domainstyle.Style
	err   error
}

func (m *mockUpdate) Handle(_ context.Context, cmd cmdstyle.UpdateCommand) (*domainstyle.Style, error) {
	m.calls++
	m.last = cmd
	if m.out != nil || m.err != nil {
		return m.out, m.err
	}
	s := sampleStyle("updated")
	s.ID = cmd.ID
	return s, nil
}

type mockDelete struct {
	calls int
	last  cmdstyle.DeleteCommand
	err   error
}

func (m *mockDelete) Handle(_ context.Context, cmd cmdstyle.DeleteCommand) error {
	m.calls++
	m.last = cmd
	return m.err
}

type mockGet struct {
	calls int
	out   *domainstyle.Style
	err   error
}

func (m *mockGet) Handle(_ context.Context, _ querystyle.GetQuery) (*domainstyle.Style, error) {
	m.calls++
	if m.out != nil || m.err != nil {
		return m.out, m.err
	}
	return sampleStyle("default"), nil
}

type mockList struct {
	calls int
	items []domainstyle.Style
	err   error
}

func (m *mockList) Handle(_ context.Context, _ querystyle.ListQuery) ([]domainstyle.Style, error) {
	m.calls++
	if m.items != nil || m.err != nil {
		return m.items, m.err
	}
	return []domainstyle.Style{*sampleStyle("default")}, nil
}

func sampleStyle(name string) *domainstyle.Style {
	font := "Montserrat"
	size := 64
	base := "#FFFFFF"
	hi := "#FFE600"
	pos := "bottom"
	wpl := 3
	return &domainstyle.Style{
		ID:             testStyleID,
		Name:           name,
		FontFamily:     &font,
		FontSize:       &size,
		BaseColor:      &base,
		HighlightColor: &hi,
		WordsPerLine:   &wpl,
		Position:       &pos,
		Uppercase:      true,
		Settings:       json.RawMessage(`{"animation":"pop"}`),
		CreatedAt:      time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
	}
}

func mount(create *mockCreate, update *mockUpdate, del *mockDelete, get *mockGet, list *mockList, withUser bool) *fiber.App {
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
	h := handler.NewSubtitleStyleHandler(create, update, del, get, list)
	app := testutil.NewTestApp()
	auth := testutil.WithUserWithoutProject(testutil.TestUserID)
	if withUser {
		app.Get("/api/subtitle-styles", auth, h.List)
		app.Post("/api/subtitle-styles", auth, h.Create)
		app.Get("/api/subtitle-styles/:id", auth, h.GetByID)
		app.Patch("/api/subtitle-styles/:id", auth, h.Update)
		app.Delete("/api/subtitle-styles/:id", auth, h.Delete)
		return app
	}
	app.Get("/api/subtitle-styles", h.List)
	app.Post("/api/subtitle-styles", h.Create)
	app.Get("/api/subtitle-styles/:id", h.GetByID)
	app.Patch("/api/subtitle-styles/:id", h.Update)
	app.Delete("/api/subtitle-styles/:id", h.Delete)
	return app
}

func validCreateBody() map[string]any {
	return map[string]any{
		"name":           "Karaoke",
		"fontFamily":     "Montserrat",
		"fontSize":       64,
		"baseColor":      "#FFFFFF",
		"highlightColor": "#FFE600",
		"wordsPerLine":   3,
		"position":       "bottom",
		"uppercase":      true,
		"settings":       map[string]any{"animation": "pop"},
	}
}
