package renderworker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaledFillSize(t *testing.T) {
	sw, sh := scaledFillSize(1920, 1080, 1080, 1920)
	if sw < 1080 || sh < 1920 {
		t.Fatalf("expected cover 1080x1920, got %dx%d", sw, sh)
	}
}

func TestWriteFaceTrackSendcmd(t *testing.T) {
	dir := t.TempDir()
	doc := &cropPathDoc{
		Samples: []cropSample{
			{TMs: 1000, CX: 0.4, CY: 0.3},
			{TMs: 2000, CX: 0.6, CY: 0.5},
			{TMs: 5000, CX: 0.5, CY: 0.5},
		},
	}
	path, err := writeFaceTrackSendcmd(dir, 0, doc, 1000, 3000, 1920, 1080, 1080, 1920)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, "crop x") || !strings.Contains(body, "crop y") {
		t.Fatalf("missing crop commands: %s", body)
	}
	if strings.Contains(body, "5.000") {
		t.Fatalf("sample outside window should be skipped: %s", body)
	}
	if filepath.Base(path) != "face_track_0.txt" {
		t.Fatalf("path=%s", path)
	}
}

func TestFaceTrackGeometry(t *testing.T) {
	g := faceTrackGeometry("/tmp/cmds.txt", 1080, 1920)
	if !strings.Contains(g, "sendcmd=f=") || !strings.Contains(g, "crop=1080:1920") {
		t.Fatalf("g=%s", g)
	}
}
