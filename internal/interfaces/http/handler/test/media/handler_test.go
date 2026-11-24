package mediatest

import (
	"errors"
	"net/http"
	"testing"

	cmdmedia "go-api/internal/application/command/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

func TestMediaHandler_InitiateUpload_Success(t *testing.T) {
	init := &mockInitiate{out: &cmdmedia.InitiateUploadResult{
		MediaID:  testutil.TestMediaID,
		UploadID: "up-1",
		Bucket:   "media",
		Key:      "media/" + testutil.TestMediaID.String() + "/source/original.mp4",
		Parts:    []port.UploadPartURL{{PartNumber: 1, URL: "https://minio/part1"}},
	}}
	h := newHandler(init, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/uploads", map[string]any{
		"filename":  "clip.mp4",
		"partCount": 1,
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out presenter.InitiateUploadResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.UploadID != "up-1" || len(out.Parts) != 1 {
		t.Fatalf("%+v", out)
	}
}

func TestMediaHandler_InitiateUpload_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/uploads", map[string]any{
		"filename": "clip.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_FromURL_InvalidURL_Blocked(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/from-url", map[string]any{
		"url": "http://127.0.0.1/secret",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_Success(t *testing.T) {
	m := sampleMedia()
	get := &mockGet{view: &domainmedia.FileView{
		ID: m.ID, ProjectID: m.ProjectID, SourceKind: m.SourceKind, Status: m.Status,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}}
	h := newHandler(nil, nil, nil, nil, nil, get, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Delete_HandlerError_Internal(t *testing.T) {
	del := &mockDelete{err: errors.New("boom")}
	h := newHandler(nil, nil, nil, nil, del, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/media/"+testutil.TestMediaID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_CompleteUpload_Success(t *testing.T) {
	complete := &mockComplete{out: sampleMedia()}
	h := newHandler(nil, nil, complete, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/complete", map[string]any{
		"uploadId": "up-1",
		"parts":    []map[string]any{{"partNumber": 1, "etag": "\"abc\""}},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_PresignParts_Success(t *testing.T) {
	presign := &mockPresign{out: []port.UploadPartURL{{PartNumber: 2, URL: "https://minio/p2"}}}
	h := newHandler(nil, presign, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/parts", map[string]any{
		"uploadId":    "up-1",
		"partNumbers": []int{2},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_ListByProject_Success(t *testing.T) {
	list := &mockList{views: []domainmedia.FileView{{ID: testutil.TestMediaID, ProjectID: testutil.TestProjectID, Status: domainmedia.StatusPending}}, total: 1}
	h := newHandler(nil, nil, nil, nil, nil, nil, list)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/media", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_FromURL_Success(t *testing.T) {
	fromURL := &mockFromURL{out: sampleMedia()}
	h := newHandler(nil, nil, nil, fromURL, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/from-url", map[string]any{
		"url": "https://example.com/video.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_InitiateUpload_InvalidProjectID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/bad/media/uploads", map[string]any{"filename": "a.mp4"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_NotFound(t *testing.T) {
	get := &mockGet{err: errors.New("media not found")}
	h := newHandler(nil, nil, nil, nil, nil, get, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Delete_Success(t *testing.T) {
	del := &mockDelete{}
	h := newHandler(nil, nil, nil, nil, del, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/media/"+testutil.TestMediaID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_PresignParts_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/parts", map[string]any{
		"uploadId": "up", "partNumbers": []int{1},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
