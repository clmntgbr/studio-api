package detectiontest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	domaindetection "go-api/internal/domain/detection"
	"go-api/internal/interfaces/http/testutil"
)

func TestDetectionHandler_ListByMedia_Success(t *testing.T) {
	list := &mockList{items: []domaindetection.Detection{sampleDetection()}}
	app := mount(list, true)

	req := httptest.NewRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/detections?kind=silence", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	var payload []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1 {
		t.Fatalf("len=%d", len(payload))
	}
	if payload[0]["kind"] != "silence" {
		t.Fatalf("kind=%v", payload[0]["kind"])
	}
	if list.calls != 1 {
		t.Fatalf("calls=%d", list.calls)
	}
}

func TestDetectionHandler_ListByMedia_Unauthorized(t *testing.T) {
	app := mount(&mockList{}, false)
	req := httptest.NewRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/detections", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestDetectionHandler_ListByMedia_InvalidMediaID(t *testing.T) {
	app := mount(&mockList{}, true)
	req := httptest.NewRequest(http.MethodGet, "/api/media/not-a-uuid/detections", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestDetectionHandler_ListByMedia_HandlerError(t *testing.T) {
	app := mount(&mockList{err: errors.New("boom")}, true)
	req := httptest.NewRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/detections", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestDetectionHandler_ListByMedia_FromToMs_Success(t *testing.T) {
	list := &mockList{items: []domaindetection.Detection{sampleDetection()}}
	app := mount(list, true)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/media/"+testutil.TestMediaID.String()+"/detections?fromMs=500&toMs=3000",
		nil,
	)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if list.last.FromMs == nil || *list.last.FromMs != 500 {
		t.Fatalf("fromMs=%v", list.last.FromMs)
	}
	if list.last.ToMs == nil || *list.last.ToMs != 3000 {
		t.Fatalf("toMs=%v", list.last.ToMs)
	}
}

func TestDetectionHandler_ListByMedia_InvalidFromMs(t *testing.T) {
	app := mount(&mockList{}, true)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/media/"+testutil.TestMediaID.String()+"/detections?fromMs=abc",
		nil,
	)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestDetectionHandler_ListByMedia_InvalidToMs(t *testing.T) {
	app := mount(&mockList{}, true)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/media/"+testutil.TestMediaID.String()+"/detections?toMs=nope",
		nil,
	)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestDetectionHandler_ListByMedia_FromMsGreaterThanToMs(t *testing.T) {
	app := mount(&mockList{}, true)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/media/"+testutil.TestMediaID.String()+"/detections?fromMs=5000&toMs=1000",
		nil,
	)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
