package renderworker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-api/internal/renderplan"
)

func TestVideoGeometry_CenterUsesCrop(t *testing.T) {
	g := videoGeometry("center", 1080, 1920)
	if !strings.Contains(g, "force_original_aspect_ratio=increase") || !strings.Contains(g, "crop=1080:1920") {
		t.Fatalf("g=%s", g)
	}
}

func TestVideoGeometry_NoneUsesPad(t *testing.T) {
	g := videoGeometry("none", 1920, 1080)
	if !strings.Contains(g, "force_original_aspect_ratio=decrease") || !strings.Contains(g, "pad=1920:1080") {
		t.Fatalf("g=%s", g)
	}
}

func TestHeadlineDrawtext(t *testing.T) {
	dir := t.TempDir()
	fontPath := filepath.Join(dir, "DejaVuSans.ttf")
	if err := os.WriteFile(fontPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := renderplan.Plan{
		Output: renderplan.Output{Height: 1920},
		Overlays: []renderplan.Overlay{
			{Kind: "headline", Text: "Hello:world", StartMs: 0, EndMs: 3000},
		},
	}
	out := headlineDrawtext(plan, dir)
	if !strings.Contains(out, "drawtext=") || !strings.Contains(out, `Hello\:world`) || !strings.Contains(out, "fontfile=") {
		t.Fatalf("out=%s", out)
	}
	if !strings.Contains(out, `enable='between(t,0,3)'`) {
		t.Fatalf("out=%s", out)
	}
	if strings.Contains(out, "fontfile='") {
		t.Fatalf("fontfile must not wrap quotes, got %s", out)
	}
}

func TestHeadlineDrawtext_NoFontSkips(t *testing.T) {
	plan := renderplan.Plan{
		Output: renderplan.Output{Height: 1920},
		Overlays: []renderplan.Overlay{
			{Kind: "headline", Text: "Hello", StartMs: 0, EndMs: 3000},
		},
	}
	if out := headlineDrawtext(plan, t.TempDir()); out != "" {
		t.Fatalf("expected empty without font, got %s", out)
	}
}
