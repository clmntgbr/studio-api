package overridetest

import (
	"errors"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/testutil"
)

func TestOverrideHandler_Create_Success(t *testing.T) {
	create := &mockCreate{}
	app := mount(create, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/overrides", validBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if create.calls != 1 || create.last.Action != "keep" || create.last.SourceStartMs != 1000 {
		t.Fatalf("cmd=%+v", create.last)
	}
	body := testutil.DecodeJSONMap(t, resp)
	if body["id"] != testutil.TestOverrideID.String() || body["action"] != "keep" {
		t.Fatalf("body=%v", body)
	}
}

func TestOverrideHandler_Create_Unauthorized(t *testing.T) {
	app := mount(nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/overrides", validBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Create_InvalidMediaID(t *testing.T) {
	app := mount(nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/bad/overrides", validBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Create_InvalidInput(t *testing.T) {
	app := mount(nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/overrides", map[string]any{
		"action":        "maybe",
		"sourceStartMs": 1000,
		"sourceEndMs":   2000,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Create_NotFound(t *testing.T) {
	create := &mockCreate{err: errors.New("media not found")}
	app := mount(create, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/overrides", validBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Create_HandlerError_BadRequest(t *testing.T) {
	create := &mockCreate{err: errors.New("action must be keep or remove")}
	app := mount(create, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/overrides", validBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Create_HandlerError_Internal(t *testing.T) {
	create := &mockCreate{err: errors.New("boom")}
	app := mount(create, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/overrides", validBody())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Delete_Success(t *testing.T) {
	del := &mockDelete{}
	app := mount(nil, del, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/overrides/"+testutil.TestOverrideID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if del.calls != 1 || del.last.OverrideID != testutil.TestOverrideID {
		t.Fatalf("cmd=%+v", del.last)
	}
}

func TestOverrideHandler_Delete_Unauthorized(t *testing.T) {
	app := mount(nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/overrides/"+testutil.TestOverrideID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Delete_InvalidID(t *testing.T) {
	app := mount(nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/overrides/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Delete_NotFound(t *testing.T) {
	del := &mockDelete{err: errors.New("override not found")}
	app := mount(nil, del, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/overrides/"+testutil.TestOverrideID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOverrideHandler_Delete_HandlerError_Internal(t *testing.T) {
	del := &mockDelete{err: errors.New("boom")}
	app := mount(nil, del, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/overrides/"+testutil.TestOverrideID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
