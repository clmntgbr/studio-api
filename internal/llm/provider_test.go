package llm

import "testing"

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
