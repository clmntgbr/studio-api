package timelinetest

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/testutil"
)

func TestTimelineHandler_GetSubtitlesASS_Success(t *testing.T) {
	subs := &mockGetSubtitles{body: "[Script Info]\n", contentType: "text/x-ssa"}
	h := newHandlerFull(nil, nil, nil, nil, subs, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/subtitles.ass?styleId="+testutil.TestOverrideID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if subs.calls != 1 || subs.last.Format != "ass" || subs.last.StyleID == nil {
		t.Fatalf("q=%+v", subs.last)
	}
	raw, _ := io.ReadAll(resp.Body)
	if string(raw) != "[Script Info]\n" {
		t.Fatalf("body=%q", raw)
	}
}

func TestTimelineHandler_GetSubtitlesSRT_Success(t *testing.T) {
	subs := &mockGetSubtitles{body: "1\n00:00:00,000 --> 00:00:01,000\nhi\n", contentType: "application/x-subrip"}
	h := newHandlerFull(nil, nil, nil, nil, subs, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/subtitles.srt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if subs.last.Format != "srt" {
		t.Fatalf("format=%s", subs.last.Format)
	}
}

func TestTimelineHandler_GetSubtitlesASS_Unauthorized(t *testing.T) {
	h := newHandlerFull(nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/subtitles.ass", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetSubtitlesASS_InvalidProjectID(t *testing.T) {
	h := newHandlerFull(nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/bad/timeline/subtitles.ass", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetSubtitlesASS_InvalidStyleID(t *testing.T) {
	h := newHandlerFull(nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/subtitles.ass?styleId=bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetSubtitlesASS_NotFound(t *testing.T) {
	subs := &mockGetSubtitles{err: errors.New("timeline not found")}
	h := newHandlerFull(nil, nil, nil, nil, subs, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/subtitles.ass", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetSubtitlesASS_HandlerError_Internal(t *testing.T) {
	subs := &mockGetSubtitles{err: errors.New("boom")}
	h := newHandlerFull(nil, nil, nil, nil, subs, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/subtitles.ass", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetSubtitlesASS_HandlerError_BadRequest(t *testing.T) {
	subs := &mockGetSubtitles{err: errors.New("format must be ass or srt")}
	h := newHandlerFull(nil, nil, nil, nil, subs, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/timeline/subtitles.ass", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
