package outputpresettest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	domainpreset "go-api/internal/domain/outputpreset"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type mockList struct {
	calls int
	items []domainpreset.Preset
	err   error
}

func (m *mockList) Handle(_ context.Context) ([]domainpreset.Preset, error) {
	m.calls++
	return m.items, m.err
}

func mount(list *mockList, withUser bool) *fiber.App {
	if list == nil {
		list = &mockList{items: []domainpreset.Preset{{
			ID: uuid.MustParse("01960000-0000-7000-8000-0000000000a1"), Name: "Master 16:9", Width: 1920, Height: 1080,
		}}}
	}
	h := handler.NewOutputPresetHandler(list)
	app := testutil.NewTestApp()
	if withUser {
		app.Get("/api/output-presets", testutil.WithUserWithoutProject(testutil.TestUserID), h.List)
		return app
	}
	app.Get("/api/output-presets", h.List)
	return app
}

func TestOutputPresetHandler_List_Success(t *testing.T) {
	list := &mockList{items: []domainpreset.Preset{{ID: testutil.TestMediaID, Name: "Master"}}}
	app := mount(list, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/output-presets", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || list.calls != 1 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, list.calls)
	}
}

func TestOutputPresetHandler_List_Unauthorized(t *testing.T) {
	app := mount(nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/output-presets", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOutputPresetHandler_List_HandlerError(t *testing.T) {
	list := &mockList{err: errors.New("boom")}
	app := mount(list, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/output-presets", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
