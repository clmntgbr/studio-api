package subtitlestyletest

import (
	"errors"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/testutil"
)

func TestSubtitleStyleHandler_List_Success(t *testing.T) {
	list := &mockList{}
	app := mount(nil, nil, nil, nil, list, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/subtitle-styles", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if list.calls != 1 {
		t.Fatalf("calls=%d", list.calls)
	}
}

func TestSubtitleStyleHandler_List_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/subtitle-styles", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}


func TestSubtitleStyleHandler_List_HandlerError(t *testing.T) {
	list := &mockList{err: errors.New("boom")}
	app := mount(nil, nil, nil, nil, list, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/subtitle-styles", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_GetByID_Success(t *testing.T) {
	get := &mockGet{}
	app := mount(nil, nil, nil, get, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/subtitle-styles/"+testStyleID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_GetByID_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/subtitle-styles/"+testStyleID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_GetByID_InvalidID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/subtitle-styles/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_GetByID_NotFound(t *testing.T) {
	get := &mockGet{err: errors.New("subtitle style not found")}
	app := mount(nil, nil, nil, get, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/subtitle-styles/"+testStyleID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_GetByID_HandlerError(t *testing.T) {
	get := &mockGet{err: errors.New("boom")}
	app := mount(nil, nil, nil, get, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/subtitle-styles/"+testStyleID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Create_Success(t *testing.T) {
	create := &mockCreate{}
	app := mount(create, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/subtitle-styles", validCreateBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if create.calls != 1 || create.last.Name != "Karaoke" {
		t.Fatalf("cmd=%+v", create.last)
	}
}

func TestSubtitleStyleHandler_Create_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/subtitle-styles", validCreateBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Create_InvalidInput(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/subtitle-styles", map[string]any{"name": ""})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Create_HandlerError_BadRequest(t *testing.T) {
	create := &mockCreate{err: errors.New("wordsPerLine must be between 2 and 4")}
	app := mount(create, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/subtitle-styles", validCreateBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Create_HandlerError_Internal(t *testing.T) {
	create := &mockCreate{err: errors.New("boom")}
	app := mount(create, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/subtitle-styles", validCreateBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Update_Success(t *testing.T) {
	update := &mockUpdate{}
	app := mount(nil, update, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/subtitle-styles/"+testStyleID.String(), map[string]any{
		"name": "Neon",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if update.calls != 1 || update.last.Name == nil || *update.last.Name != "Neon" {
		t.Fatalf("cmd=%+v", update.last)
	}
}

func TestSubtitleStyleHandler_Update_InvalidInput(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/subtitle-styles/"+testStyleID.String(), map[string]any{
		"position": "side",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Update_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/subtitle-styles/"+testStyleID.String(), map[string]any{"name": "x"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Update_InvalidID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/subtitle-styles/bad", map[string]any{"name": "x"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Update_NotFound(t *testing.T) {
	update := &mockUpdate{err: errors.New("subtitle style not found")}
	app := mount(nil, update, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/subtitle-styles/"+testStyleID.String(), map[string]any{"name": "x"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Update_HandlerError_BadRequest(t *testing.T) {
	update := &mockUpdate{err: errors.New("position must be top, middle or bottom")}
	app := mount(nil, update, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/subtitle-styles/"+testStyleID.String(), map[string]any{"name": "x"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Update_HandlerError_Internal(t *testing.T) {
	update := &mockUpdate{err: errors.New("boom")}
	app := mount(nil, update, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/subtitle-styles/"+testStyleID.String(), map[string]any{"name": "x"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Delete_Success(t *testing.T) {
	del := &mockDelete{}
	app := mount(nil, nil, del, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/subtitle-styles/"+testStyleID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if del.calls != 1 || del.last.ID != testStyleID {
		t.Fatalf("cmd=%+v", del.last)
	}
}

func TestSubtitleStyleHandler_Delete_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/subtitle-styles/"+testStyleID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Delete_InvalidID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/subtitle-styles/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Delete_NotFound(t *testing.T) {
	del := &mockDelete{err: errors.New("subtitle style not found")}
	app := mount(nil, nil, del, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/subtitle-styles/"+testStyleID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSubtitleStyleHandler_Delete_HandlerError_Internal(t *testing.T) {
	del := &mockDelete{err: errors.New("boom")}
	app := mount(nil, nil, del, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/subtitle-styles/"+testStyleID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
