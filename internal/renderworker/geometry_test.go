package renderworker

import (
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
	plan := renderplan.Plan{
		Output: renderplan.Output{Height: 1920},
		Overlays: []renderplan.Overlay{
			{Kind: "headline", Text: "Hello:world", StartMs: 0, EndMs: 3000},
		},
	}
	out := headlineDrawtext(plan)
	if !strings.Contains(out, "drawtext=") || !strings.Contains(out, `Hello\:world`) {
		t.Fatalf("out=%s", out)
	}
}
