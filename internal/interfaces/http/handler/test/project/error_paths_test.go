package projecttest

import (
	"errors"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/testutil"
)

func TestProjectHandler_GetByID_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_GetByID_HandlerError_Internal(t *testing.T) {
	get := &mockGet{err: errors.New("db down")}
	h := newHandler(nil, nil, nil, get, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Update_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/projects/"+testutil.TestProjectID.String(), map[string]any{"name": "X"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Update_InvalidID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/projects/not-a-uuid", map[string]any{"name": "X"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Update_InvalidBody(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/projects/"+testutil.TestProjectID.String(), map[string]any{"name": ""})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Update_HandlerError_Internal(t *testing.T) {
	update := &mockUpdate{err: errors.New("db down")}
	h := newHandler(nil, update, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPatch, "/api/projects/"+testutil.TestProjectID.String(), map[string]any{"name": "X"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Delete_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/projects/"+testutil.TestProjectID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Delete_InvalidID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/projects/not-a-uuid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_Delete_NotFound(t *testing.T) {
	del := &mockDelete{err: errors.New("project not found")}
	h := newHandler(nil, nil, del, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/projects/"+testutil.TestProjectID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestProjectHandler_List_HandlerError_Internal(t *testing.T) {
	list := &mockList{err: errors.New("db down")}
	h := newHandler(nil, nil, nil, nil, list)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
