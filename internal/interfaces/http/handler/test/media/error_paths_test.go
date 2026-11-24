package mediatest

import (
	"errors"
	"net/http"
	"testing"

	cmdmedia "go-api/internal/application/command/media"
	"go-api/internal/domain/port"
	"go-api/internal/interfaces/http/testutil"
)

func TestMediaHandler_InitiateUpload_InvalidBody(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/uploads", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_InitiateUpload_NotFound(t *testing.T) {
	init := &mockInitiate{err: errors.New("project not found")}
	h := newHandler(init, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/uploads", map[string]any{
		"filename": "clip.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_InitiateUpload_HandlerError_Internal(t *testing.T) {
	init := &mockInitiate{err: errors.New("boom")}
	h := newHandler(init, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/uploads", map[string]any{
		"filename": "clip.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_InitiateUpload_DefaultPartCount(t *testing.T) {
	init := &mockInitiate{out: &cmdmedia.InitiateUploadResult{
		MediaID:  testutil.TestMediaID,
		UploadID: "up-default",
		Bucket:   "media",
		Key:      "media/" + testutil.TestMediaID.String() + "/source/original.mp4",
		Parts:    []port.UploadPartURL{{PartNumber: 1, URL: "https://minio/part1"}},
	}}
	h := newHandler(init, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/uploads", map[string]any{
		"filename": "clip.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_PresignParts_InvalidMediaID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/bad/uploads/parts", map[string]any{
		"uploadId": "up", "partNumbers": []int{1},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_PresignParts_InvalidBody(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/parts", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_PresignParts_NotFound(t *testing.T) {
	presign := &mockPresign{err: errors.New("media not found")}
	h := newHandler(nil, presign, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/parts", map[string]any{
		"uploadId": "up", "partNumbers": []int{1},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_PresignParts_HandlerError_Internal(t *testing.T) {
	presign := &mockPresign{err: errors.New("boom")}
	h := newHandler(nil, presign, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/parts", map[string]any{
		"uploadId": "up", "partNumbers": []int{1},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_CompleteUpload_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/complete", map[string]any{
		"uploadId": "up", "parts": []map[string]any{{"partNumber": 1, "etag": "\"a\""}},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_CompleteUpload_InvalidMediaID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/bad/uploads/complete", map[string]any{
		"uploadId": "up", "parts": []map[string]any{{"partNumber": 1, "etag": "\"a\""}},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_CompleteUpload_InvalidBody(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/complete", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_CompleteUpload_NotFound(t *testing.T) {
	complete := &mockComplete{err: errors.New("media not found")}
	h := newHandler(nil, nil, complete, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/complete", map[string]any{
		"uploadId": "up", "parts": []map[string]any{{"partNumber": 1, "etag": "\"a\""}},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_CompleteUpload_HandlerError_Internal(t *testing.T) {
	complete := &mockComplete{err: errors.New("boom")}
	h := newHandler(nil, nil, complete, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/media/"+testutil.TestMediaID.String()+"/uploads/complete", map[string]any{
		"uploadId": "up", "parts": []map[string]any{{"partNumber": 1, "etag": "\"a\""}},
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_FromURL_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/from-url", map[string]any{
		"url": "https://1.1.1.1/video.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_FromURL_InvalidProjectID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/bad/media/from-url", map[string]any{
		"url": "https://1.1.1.1/video.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_FromURL_InvalidBody(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/from-url", map[string]any{})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_FromURL_NotFound(t *testing.T) {
	fromURL := &mockFromURL{err: errors.New("project not found")}
	h := newHandler(nil, nil, nil, fromURL, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/from-url", map[string]any{
		"url": "https://1.1.1.1/video.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_FromURL_HandlerError_Internal(t *testing.T) {
	fromURL := &mockFromURL{err: errors.New("boom")}
	h := newHandler(nil, nil, nil, fromURL, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodPost, "/api/projects/"+testutil.TestProjectID.String()+"/media/from-url", map[string]any{
		"url": "https://1.1.1.1/video.mp4",
	})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_ListByProject_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/media", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_ListByProject_InvalidProjectID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/bad/media", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_ListByProject_HandlerError(t *testing.T) {
	list := &mockList{err: errors.New("boom")}
	h := newHandler(nil, nil, nil, nil, nil, nil, list)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/projects/"+testutil.TestProjectID.String()+"/media", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_InvalidMediaID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_HandlerError_Internal(t *testing.T) {
	get := &mockGet{err: errors.New("boom")}
	h := newHandler(nil, nil, nil, nil, nil, get, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Delete_Unauthorized(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, false)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/media/"+testutil.TestMediaID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Delete_InvalidMediaID(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/media/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMediaHandler_Delete_NotFound(t *testing.T) {
	del := &mockDelete{err: errors.New("media not found")}
	h := newHandler(nil, nil, nil, nil, del, nil, nil)
	app := mount(h, true)
	req, _ := testutil.JSONRequest(http.MethodDelete, "/api/media/"+testutil.TestMediaID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
