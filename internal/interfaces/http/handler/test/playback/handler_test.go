package playbacktest

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	querymedia "go-api/internal/application/query/media"
	"go-api/internal/domain/port"
	"go-api/internal/interfaces/http/testutil"
)

func TestPlaybackHandler_Get_Success(t *testing.T) {
	playback := &mockPlayback{}
	app := mount(playback, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/playback", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	out := testutil.DecodeJSONMap(t, resp)
	if out["url"] != "https://cdn.example/proxy.mp4" || out["kind"] != "proxy" {
		t.Fatalf("%+v", out)
	}
	if playback.calls != 1 || playback.last.Kind != "auto" {
		t.Fatalf("query=%+v", playback.last)
	}
}

func TestPlaybackHandler_Get_KindQuery(t *testing.T) {
	playback := &mockPlayback{}
	app := mount(playback, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/playback?kind=source", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if playback.last.Kind != "source" {
		t.Fatalf("kind=%q", playback.last.Kind)
	}
}

func TestPlaybackHandler_Get_Unauthorized(t *testing.T) {
	app := mount(nil, nil, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/playback", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Get_InvalidMediaID(t *testing.T) {
	app := mount(nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad/playback", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Get_NotFound(t *testing.T) {
	playback := &mockPlayback{err: errors.New("media not found")}
	app := mount(playback, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/playback", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Get_NotReady(t *testing.T) {
	playback := &mockPlayback{err: errors.New("proxy not ready")}
	app := mount(playback, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/playback", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Get_KindMust(t *testing.T) {
	playback := &mockPlayback{err: errors.New("kind must be proxy or source")}
	app := mount(playback, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/playback", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Get_HandlerError(t *testing.T) {
	playback := &mockPlayback{err: errors.New("boom")}
	app := mount(playback, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/playback", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Stream_Success(t *testing.T) {
	stream := &mockStream{}
	app := mount(nil, stream, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/stream", nil)
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
	if stream.calls != 1 || stream.last.Kind != "auto" {
		t.Fatalf("query=%+v", stream.last)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(body) != "fake-video" {
		t.Fatalf("body=%q", body)
	}
}

func TestPlaybackHandler_Stream_RangePartial(t *testing.T) {
	stream := &mockStream{out: &querymedia.StreamResult{
		Kind: "proxy",
		Stream: &port.ObjectStream{
			Body:          io.NopCloser(strings.NewReader("ab")),
			ContentType:   "video/mp4",
			ContentLength: 2,
			ContentRange:  "bytes 0-1/10",
			StatusCode:    206,
		},
	}}
	app := mount(nil, stream, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/stream?kind=proxy", nil)
	req.Header.Set("Range", "bytes=0-1")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Range") != "bytes 0-1/10" {
		t.Fatalf("content-range=%q", resp.Header.Get("Content-Range"))
	}
	if stream.last.RangeHeader != "bytes=0-1" || stream.last.Kind != "proxy" {
		t.Fatalf("query=%+v", stream.last)
	}
}

func TestPlaybackHandler_Stream_DefaultStatusWhenZero(t *testing.T) {
	stream := &mockStream{out: &querymedia.StreamResult{
		Kind: "source",
		Stream: &port.ObjectStream{
			Body:          io.NopCloser(strings.NewReader("x")),
			ContentType:   "video/mp4",
			ContentLength: 0,
			StatusCode:    0,
		},
	}}
	app := mount(nil, stream, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/stream", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Stream_Unauthorized(t *testing.T) {
	app := mount(nil, &mockStream{}, false)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/stream", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Stream_NotConfigured(t *testing.T) {
	app := mount(nil, nil, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/stream", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Stream_InvalidMediaID(t *testing.T) {
	app := mount(nil, &mockStream{}, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/bad/stream", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Stream_NotFound(t *testing.T) {
	stream := &mockStream{err: errors.New("media not found")}
	app := mount(nil, stream, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/stream", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Stream_NotReady(t *testing.T) {
	stream := &mockStream{err: errors.New("source not ready")}
	app := mount(nil, stream, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/stream", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPlaybackHandler_Stream_HandlerError(t *testing.T) {
	stream := &mockStream{err: errors.New("boom")}
	app := mount(nil, stream, true)
	req, _ := testutil.JSONRequest(http.MethodGet, "/api/media/"+testutil.TestMediaID.String()+"/stream", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
