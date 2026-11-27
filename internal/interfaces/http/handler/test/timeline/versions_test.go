package timelinetest

import (
	"errors"
	"net/http"
	"testing"

	domaintimeline "go-api/internal/domain/timeline"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

func TestTimelineHandler_ListVersions_Success(t *testing.T) {
	list := &mockListVersions{items: []domaintimeline.Timeline{*sampleTimeline()}}
	h := newHandlerFull(nil, nil, nil, list, nil, nil)
	app := mount(h, true)

	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/versions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out []presenter.TimelineVersionResponse
	testutil.DecodeJSON(t, resp, &out)
	if len(out) != 1 || out[0].Version != 1 || out[0].Fingerprint != "fp-1" {
		t.Fatalf("%+v", out)
	}
	if list.calls != 1 {
		t.Fatalf("calls=%d", list.calls)
	}
}

func TestTimelineHandler_ListVersions_Empty(t *testing.T) {
	list := &mockListVersions{items: nil}
	h := newHandlerFull(nil, nil, nil, list, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/versions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out []presenter.TimelineVersionResponse
	testutil.DecodeJSON(t, resp, &out)
	if out == nil {
		t.Fatal("expected [] not null")
	}
	if len(out) != 0 {
		t.Fatalf("len=%d", len(out))
	}
}

func TestTimelineHandler_ListVersions_Unauthorized(t *testing.T) {
	h := newHandlerFull(nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/versions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_ListVersions_InvalidMediaID(t *testing.T) {
	h := newHandlerFull(nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad/timeline/versions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_ListVersions_HandlerError_Internal(t *testing.T) {
	list := &mockListVersions{err: errors.New("boom")}
	h := newHandlerFull(nil, nil, nil, list, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/versions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
