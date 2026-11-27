package timelinetest

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/testutil"
)

func TestTimelineHandler_GetExportFCPXML_Success(t *testing.T) {
	exp := &mockGetExport{body: "<fcpxml/>", ctype: "application/xml", filename: "timeline.fcpxml"}
	h := newHandlerFull(nil, nil, nil, nil, nil, exp)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/export.fcpxml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if exp.calls != 1 || exp.last.Format != "fcpxml" {
		t.Fatalf("q=%+v", exp.last)
	}
	raw, _ := io.ReadAll(resp.Body)
	if string(raw) != "<fcpxml/>" {
		t.Fatalf("body=%q", raw)
	}
}

func TestTimelineHandler_GetExportEDL_Success(t *testing.T) {
	exp := &mockGetExport{body: "TITLE: x\n", ctype: "application/x-edl"}
	h := newHandlerFull(nil, nil, nil, nil, nil, exp)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/export.edl", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || exp.last.Format != "edl" {
		t.Fatalf("status=%d format=%s", resp.StatusCode, exp.last.Format)
	}
}

func TestTimelineHandler_GetExportOTIO_Success(t *testing.T) {
	exp := &mockGetExport{body: "{}", ctype: "application/otio+json"}
	h := newHandlerFull(nil, nil, nil, nil, nil, exp)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/export.otio", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || exp.last.Format != "otio" {
		t.Fatalf("status=%d format=%s", resp.StatusCode, exp.last.Format)
	}
}

func TestTimelineHandler_GetExportPremiere_Success(t *testing.T) {
	exp := &mockGetExport{body: "<xmeml/>", ctype: "application/xml", filename: "timeline.premiere.xml"}
	h := newHandlerFull(nil, nil, nil, nil, nil, exp)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/export.premiere.xml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || exp.last.Format != "premiere" {
		t.Fatalf("status=%d format=%s", resp.StatusCode, exp.last.Format)
	}
}

func TestTimelineHandler_GetExportFCPXML_Unauthorized(t *testing.T) {
	h := newHandlerFull(nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/export.fcpxml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetExportFCPXML_InvalidMediaID(t *testing.T) {
	h := newHandlerFull(nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad/timeline/export.fcpxml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetExportFCPXML_NotFound(t *testing.T) {
	exp := &mockGetExport{err: errors.New("timeline not found")}
	h := newHandlerFull(nil, nil, nil, nil, nil, exp)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/export.fcpxml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetExportFCPXML_HandlerError_Internal(t *testing.T) {
	exp := &mockGetExport{err: errors.New("boom")}
	h := newHandlerFull(nil, nil, nil, nil, nil, exp)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/export.fcpxml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTimelineHandler_GetExportFCPXML_HandlerError_BadRequest(t *testing.T) {
	exp := &mockGetExport{err: errors.New("format must be fcpxml, edl or otio")}
	h := newHandlerFull(nil, nil, nil, nil, nil, exp)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/timeline/export.fcpxml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
