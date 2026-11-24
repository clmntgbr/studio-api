package llm

import (
	"strings"
	"testing"
)

func TestResolveCompatibleDefaults_Grok(t *testing.T) {
	base, model, provider := resolveCompatibleDefaults(Config{Provider: "grok"})
	if base != "https://api.x.ai/v1" {
		t.Fatalf("base=%s", base)
	}
	if model != "grok-3-mini" {
		t.Fatalf("model=%s", model)
	}
	if provider != "grok" {
		t.Fatalf("provider=%s", provider)
	}
}

func TestResolveCompatibleDefaults_GrokKeepsExplicitModel(t *testing.T) {
	_, model, _ := resolveCompatibleDefaults(Config{
		Provider: "xai",
		Model:    "grok-3",
		BaseURL:  "https://api.x.ai/v1",
	})
	if model != "grok-3" {
		t.Fatalf("model=%s", model)
	}
}

func TestNewAnalyzer_GrokRequiresKeyAtRuntime(t *testing.T) {
	a := NewAnalyzer(Config{Provider: "grok", APIKey: "xai-test"})
	if _, ok := a.(*OpenAICompatibleAnalyzer); !ok {
		t.Fatalf("got %T", a)
	}
}

func TestWritePackedTranscript_GroupsAndCaps(t *testing.T) {
	words := make([]Word, 0, 40)
	for i := 0; i < 40; i++ {
		words = append(words, Word{
			Text:          "w",
			SourceStartMs: int64(i * 100),
			SourceEndMs:   int64(i*100 + 90),
		})
	}
	var b strings.Builder
	writePackedTranscript(&b, words, 10, 80)
	out := b.String()
	if !strings.Contains(out, "[0-") {
		t.Fatalf("missing start bound: %q", out)
	}
	if !strings.Contains(out, "…(truncated)") {
		t.Fatalf("expected truncation: %q", out)
	}
	lines := strings.Count(out, "\n")
	if lines < 1 || lines > 4 {
		t.Fatalf("unexpected line count=%d out=%q", lines, out)
	}
}
