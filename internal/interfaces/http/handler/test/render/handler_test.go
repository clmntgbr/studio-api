package rendertest

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	queryrender "go-api/internal/application/query/render"
	"go-api/internal/domain/port"
	"go-api/internal/interfaces/http/testutil"
)

func TestRenderHandler_Create_Success(t *testing.T) {
	create := &mockCreate{}
	app := mount(create, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{
		"presetId": testPresetID.String(),
		"kind":     "master",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if create.calls != 1 || create.last.PresetID != testPresetID {
		t.Fatalf("cmd=%+v", create.last)
	}
}

func TestRenderHandler_Create_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{
		"presetId": testPresetID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Create_InvalidProjectID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/bad/renders", map[string]any{"presetId": testPresetID.String()})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Create_InvalidInput(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Create_WithStyleID_Success(t *testing.T) {
	create := &mockCreate{}
	app := mount(create, nil, nil, nil, nil, true)
	styleID := testutil.TestOverrideID.String()
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{
		"presetId":        testPresetID.String(),
		"subtitleStyleId": styleID,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if create.last.SubtitleStyleID == nil || create.last.SubtitleStyleID.String() != styleID {
		t.Fatalf("cmd=%+v", create.last)
	}
}

func TestRenderHandler_Create_WithClipID_Success(t *testing.T) {
	create := &mockCreate{}
	app := mount(create, nil, nil, nil, nil, true)
	clipID := testutil.TestOverrideID.String()
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{
		"presetId": testPresetID.String(),
		"kind":     "clip",
		"clipId":   clipID,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if create.last.ClipID == nil || create.last.ClipID.String() != clipID || create.last.Kind != "clip" {
		t.Fatalf("cmd=%+v", create.last)
	}
}

func TestRenderHandler_Create_BadRequest_Kind(t *testing.T) {
	create := &mockCreate{err: errors.New("kind must be master, clip, audio or custom")}
	app := mount(create, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{
		"presetId": testPresetID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Create_DraftTimeline_OK(t *testing.T) {
	create := &mockCreate{}
	app := mount(create, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{
		"presetId": testPresetID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if create.calls != 1 {
		t.Fatalf("calls=%d", create.calls)
	}
}

func TestRenderHandler_Create_NotFound(t *testing.T) {
	create := &mockCreate{err: errors.New("timeline not found")}
	app := mount(create, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{
		"presetId": testPresetID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Create_HandlerError_Internal(t *testing.T) {
	create := &mockCreate{err: errors.New("boom")}
	app := mount(create, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/renders", map[string]any{
		"presetId": testPresetID.String(),
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_ListByProject_Success(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/renders", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_ListByProject_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/renders", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_ListByProject_InvalidProjectID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/bad/renders", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_ListByProject_HandlerError(t *testing.T) {
	list := &mockList{err: errors.New("boom")}
	app := mount(nil, nil, nil, list, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/renders", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_GetByID_Success(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_GetByID_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_GetByID_InvalidID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_GetByID_NotFound(t *testing.T) {
	get := &mockGet{err: errors.New("render not found")}
	app := mount(nil, nil, get, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_GetByID_HandlerError(t *testing.T) {
	get := &mockGet{err: errors.New("boom")}
	app := mount(nil, nil, get, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Cancel_Success(t *testing.T) {
	cancel := &mockCancel{}
	app := mount(nil, cancel, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/renders/"+testRenderID.String()+"/cancel", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || cancel.calls != 1 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, cancel.calls)
	}
}

func TestRenderHandler_Cancel_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/renders/"+testRenderID.String()+"/cancel", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Cancel_InvalidID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/renders/bad/cancel", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Cancel_NotFound(t *testing.T) {
	cancel := &mockCancel{err: errors.New("render not found")}
	app := mount(nil, cancel, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/renders/"+testRenderID.String()+"/cancel", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Cancel_HandlerError(t *testing.T) {
	cancel := &mockCancel{err: errors.New("boom")}
	app := mount(nil, cancel, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/renders/"+testRenderID.String()+"/cancel", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Download_Success(t *testing.T) {
	app := mount(nil, nil, nil, nil, &mockDownload{}, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "video/mp4" {
		t.Fatalf("content-type=%q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Fatalf("content-disposition=%q", cd)
	}
}

func TestRenderHandler_Download_Unauthorized(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Download_InvalidID(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/bad/download", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Download_NotFound(t *testing.T) {
	download := &mockDownload{err: errors.New("render not found")}
	app := mount(nil, nil, nil, nil, download, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Download_NotReady(t *testing.T) {
	download := &mockDownload{err: errors.New("render is not ready")}
	app := mount(nil, nil, nil, nil, download, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Download_OutputMissing(t *testing.T) {
	download := &mockDownload{err: errors.New("render output missing")}
	app := mount(nil, nil, nil, nil, download, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Download_HandlerError(t *testing.T) {
	download := &mockDownload{err: errors.New("boom")}
	app := mount(nil, nil, nil, nil, download, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRenderHandler_Download_PartSuccess(t *testing.T) {
	download := &mockDownload{}
	app := mount(nil, nil, nil, nil, download, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download?part=2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if download.last.Part != 2 {
		t.Fatalf("part=%d", download.last.Part)
	}
}

func TestRenderHandler_Download_InvalidPart(t *testing.T) {
	app := mount(nil, nil, nil, nil, nil, true)
	for _, raw := range []string{"0", "-1", "abc"} {
		req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download?part="+raw, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("part=%s status=%d", raw, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
}

func TestRenderHandler_Download_RangeAndDefaultStatus(t *testing.T) {
	download := &mockDownload{out: &queryrender.DownloadResult{
		Filename: "part-1.mp4",
		Stream: &port.ObjectStream{
			Body:          io.NopCloser(strings.NewReader("ab")),
			ContentType:   "video/mp4",
			ContentLength: 2,
			ContentRange:  "bytes 0-1/8",
			StatusCode:    0,
		},
	}}
	app := mount(nil, nil, nil, nil, download, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/renders/"+testRenderID.String()+"/download", nil)
	req.Header.Set("Range", "bytes=0-1")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Range") != "bytes 0-1/8" {
		t.Fatalf("content-range=%q", resp.Header.Get("Content-Range"))
	}
	if download.last.RangeHeader != "bytes=0-1" {
		t.Fatalf("range=%q", download.last.RangeHeader)
	}
}
