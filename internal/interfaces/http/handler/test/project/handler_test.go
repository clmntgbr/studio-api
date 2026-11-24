package projecttest

import (
	"errors"
	"net/http"
	"testing"

	domainproject "go-api/internal/domain/project"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

func TestProjectHandler_Create_Success(t *testing.T) {
	create := &mockCreate{out: sampleProject()}
	h := newHandler(create, nil, nil, nil, nil)
	app := mount(h, true)

	req, err := testutil.JSONRequest(http.MethodPost, "/api/projects", map[string]any{"name": "Demo"})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if create.calls != 1 || create.last.Name != "Demo" {
		t.Fatalf("create calls=%d name=%q", create.calls, create.last.Name)
	}
	var out presenter.ProjectResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.Name != "Demo" {
		t.Fatalf("name=%q", out.Name)
	}
}

func TestProjectHandler_Create_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects", map[string]any{"name": "Demo"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Create_InvalidBody(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects", map[string]any{"name": ""})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_List_Success(t *testing.T) {
	list := &mockList{views: []domainproject.View{*sampleView()}, total: 1}
	h := newHandler(nil, nil, nil, nil, list)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if list.calls != 1 {
		t.Fatalf("list calls=%d", list.calls)
	}
}

func TestProjectHandler_GetByID_Success(t *testing.T) {
	get := &mockGet{view: sampleView()}
	h := newHandler(nil, nil, nil, get, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_GetByID_NotFound(t *testing.T) {
	get := &mockGet{err: errors.New("project not found")}
	h := newHandler(nil, nil, nil, get, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_GetByID_InvalidID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/not-a-uuid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Update_Success(t *testing.T) {
	update := &mockUpdate{out: sampleProject()}
	h := newHandler(nil, update, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/projects/"+testutil.TestProjectID.String(), map[string]any{"name": "Renamed"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Delete_Success(t *testing.T) {
	del := &mockDelete{}
	h := newHandler(nil, nil, del, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/projects/"+testutil.TestProjectID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Delete_HandlerError_Internal(t *testing.T) {
	del := &mockDelete{err: errors.New("db down")}
	h := newHandler(nil, nil, del, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/projects/"+testutil.TestProjectID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Update_NotFound(t *testing.T) {
	update := &mockUpdate{err: errors.New("project not found")}
	h := newHandler(nil, update, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/projects/"+testutil.TestProjectID.String(), map[string]any{"name": "X"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_List_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Create_HandlerError_Internal(t *testing.T) {
	create := &mockCreate{err: errors.New("failed to create project")}
	h := newHandler(create, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects", map[string]any{"name": "Demo"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
