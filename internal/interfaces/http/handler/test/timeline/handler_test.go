package timelinetest

import (
	"errors"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

func TestTimelineHandler_Get_Success(t *testing.T) {
	get := &mockGet{view: sampleView()}
	h := newHandler(nil, nil, get)
	app := mount(h, true)

	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out presenter.TimelineResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.ProjectID != testutil.TestProjectID.String() || out.Fingerprint != "fp-1" {
		t.Fatalf("%+v", out)
	}
}

func TestTimelineHandler_Get_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Get_InvalidProjectID(t *testing.T) {
	h := newHandler(nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/bad/timeline", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Get_NotFound(t *testing.T) {
	get := &mockGet{err: errors.New("timeline not found")}
	h := newHandler(nil, nil, get)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Get_HandlerError_Internal(t *testing.T) {
	get := &mockGet{err: errors.New("boom")}
	h := newHandler(nil, nil, get)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Preview_Success(t *testing.T) {
	preview := &mockPreview{out: samplePreview()}
	h := newHandler(preview, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/preview", map[string]any{
		"mediaId": testutil.TestMediaID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out presenter.TimelineResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.Fingerprint != "preview-fp" || out.DurationMs != 9000 {
		t.Fatalf("%+v", out)
	}
	if preview.cmd.MediaID != testutil.TestMediaID {
		t.Fatalf("mediaId=%s", preview.cmd.MediaID)
	}
}

func TestTimelineHandler_Preview_EmptyBody_Success(t *testing.T) {
	preview := &mockPreview{out: samplePreview()}
	h := newHandler(preview, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/preview", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Preview_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/preview", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Preview_InvalidProjectID(t *testing.T) {
	h := newHandler(nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/bad/timeline/preview", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Preview_InvalidMediaID(t *testing.T) {
	h := newHandler(nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/preview", map[string]any{
		"mediaId": "not-a-uuid",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Preview_NotFound(t *testing.T) {
	preview := &mockPreview{err: errors.New("media not found")}
	h := newHandler(preview, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/preview", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Preview_DurationError(t *testing.T) {
	preview := &mockPreview{err: errors.New("duration unknown")}
	h := newHandler(preview, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/preview", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Preview_HandlerError_Internal(t *testing.T) {
	preview := &mockPreview{err: errors.New("boom")}
	h := newHandler(preview, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/preview", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_Validate_Gone(t *testing.T) {
	h := newHandler(nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/validate", map[string]any{
		"version": 1,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusGone {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body map[string]any
	testutil.DecodeJSON(t, resp, &body)
	want := "timeline validate removed; POST /projects/:id/renders to ship"
	if body["message"] != want {
		t.Fatalf("message=%v", body["message"])
	}
}
