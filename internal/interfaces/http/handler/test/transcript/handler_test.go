package transcripttest

import (
	"errors"
	"net/http"
	"testing"

	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

func TestTranscriptHandler_GetByMedia_Success(t *testing.T) {
	get := &mockGet{view: sampleView()}
	app := mount(get, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript?fromMs=0&toMs=1000", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out presenter.TranscriptResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.Provider != "assemblyai" || len(out.Words) != 2 {
		t.Fatalf("%+v", out)
	}
	if get.last.FromMs == nil || *get.last.FromMs != 0 || get.last.ToMs == nil || *get.last.ToMs != 1000 {
		t.Fatalf("query=%+v", get.last)
	}
}

func TestTranscriptHandler_GetByMedia_Unauthorized(t *testing.T) {
	app := mount(nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetByMedia_InvalidMediaID(t *testing.T) {
	app := mount(nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad/transcript", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetByMedia_InvalidFromMs(t *testing.T) {
	app := mount(nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript?fromMs=x", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetByMedia_InvalidToMs(t *testing.T) {
	app := mount(nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript?toMs=x", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetByMedia_FromMsGreaterThanToMs(t *testing.T) {
	app := mount(nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript?fromMs=9&toMs=1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetByMedia_NotFound(t *testing.T) {
	get := &mockGet{err: errors.New("transcript not found")}
	app := mount(get, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetByMedia_HandlerError_Internal(t *testing.T) {
	get := &mockGet{err: errors.New("boom")}
	app := mount(get, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetSRT_Success(t *testing.T) {
	srt := &mockGetSRT{out: "1\n00:00:00,000 --> 00:00:01,000\nbonjour\n"}
	app := mountFull(nil, srt, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript.srt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if srt.calls != 1 {
		t.Fatalf("calls=%d", srt.calls)
	}
}

func TestTranscriptHandler_GetSRT_Unauthorized(t *testing.T) {
	app := mountFull(nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript.srt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetSRT_InvalidMediaID(t *testing.T) {
	app := mountFull(nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad/transcript.srt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetSRT_NotFound(t *testing.T) {
	srt := &mockGetSRT{err: errors.New("transcript not found")}
	app := mountFull(nil, srt, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript.srt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTranscriptHandler_GetSRT_HandlerError_Internal(t *testing.T) {
	srt := &mockGetSRT{err: errors.New("boom")}
	app := mountFull(nil, srt, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/transcript.srt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
