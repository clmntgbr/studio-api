package miniowebhooktest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

func TestMinioWebhookHandler_ObjectCreated_Success(t *testing.T) {
	mark := &mockMarkReady{}
	app := mount(mark, true)

	mediaID := testutil.TestMediaID
	req := httptest.NewRequest(http.MethodPost, "/webhooks/minio/object-created", bytes.NewReader(sampleObjectCreatedBody(mediaID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testWebhookSecret)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	if len(mark.calls) != 1 {
		t.Fatalf("expected 1 markReady call, got %d", len(mark.calls))
	}
	if mark.calls[0].MediaID != mediaID {
		t.Fatalf("mediaId=%s", mark.calls[0].MediaID)
	}
	wantKey := "media/" + mediaID.String() + "/source/original.mp4"
	if mark.calls[0].SourceKey != wantKey {
		t.Fatalf("sourceKey=%s", mark.calls[0].SourceKey)
	}
	if mark.calls[0].SizeBytes == nil || *mark.calls[0].SizeBytes != 1024 {
		t.Fatalf("sizeBytes=%v", mark.calls[0].SizeBytes)
	}
}

func TestMinioWebhookHandler_ObjectCreated_Unauthorized(t *testing.T) {
	app := mount(&mockMarkReady{}, true)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/minio/object-created", bytes.NewReader(sampleObjectCreatedBody(testutil.TestMediaID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMinioWebhookHandler_ObjectCreated_InvalidBody(t *testing.T) {
	app := mount(&mockMarkReady{}, false)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/minio/object-created", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMinioWebhookHandler_ObjectCreated_IgnoresNonSourceKeys(t *testing.T) {
	mark := &mockMarkReady{}
	app := mount(mark, false)
	body := []byte(`{"Records":[{"eventName":"s3:ObjectCreated:Put","s3":{"bucket":{"name":"media"},"object":{"key":"media/` + uuid.New().String() + `/proxy/preview.mp4","size":1}}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/minio/object-created", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if len(mark.calls) != 0 {
		t.Fatalf("expected no markReady calls, got %d", len(mark.calls))
	}
}

func TestMinioWebhookHandler_ObjectCreated_HandlerError(t *testing.T) {
	mark := &mockMarkReady{err: errors.New("boom")}
	app := mount(mark, false)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/minio/object-created", bytes.NewReader(sampleObjectCreatedBody(testutil.TestMediaID)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["message"] == nil {
		t.Fatalf("expected message envelope, got %#v", payload)
	}
}

func TestMinioWebhookHandler_ObjectCreated_URLEncodedKey(t *testing.T) {
	mark := &mockMarkReady{}
	app := mount(mark, false)
	mediaID := testutil.TestMediaID
	encoded := "media%2F" + mediaID.String() + "%2Fsource%2Foriginal.mp4"
	body := []byte(`{"Records":[{"eventName":"s3:ObjectCreated:Put","s3":{"bucket":{"name":"media"},"object":{"key":"` + encoded + `","size":9}}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/minio/object-created", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if len(mark.calls) != 1 {
		t.Fatalf("calls=%d", len(mark.calls))
	}
	want := "media/" + mediaID.String() + "/source/original.mp4"
	if mark.calls[0].SourceKey != want {
		t.Fatalf("sourceKey=%s", mark.calls[0].SourceKey)
	}
}

func TestMinioWebhookHandler_ObjectCreated_EmptyRecords(t *testing.T) {
	mark := &mockMarkReady{}
	app := mount(mark, false)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/minio/object-created", bytes.NewReader([]byte(`{"Records":[]}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
