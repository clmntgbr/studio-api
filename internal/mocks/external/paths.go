package external

import (
	"os"
	"path/filepath"
)

// DefaultDir is the in-container path (compose mounts the repo at /app).
const DefaultDir = "/app/testdata/mocks/external"

// Root returns the directory that holds recorded paid-external payloads.
func Root() string {
	if v := os.Getenv("MOCK_EXTERNAL_DIR"); v != "" {
		return v
	}
	return DefaultDir
}

func AssemblyAIWordsPath() string {
	return filepath.Join(Root(), "assemblyai", "words.json")
}

func AssemblyAIProviderJobPath() string {
	return filepath.Join(Root(), "assemblyai", "provider.job")
}

func LLMResultPath() string {
	return filepath.Join(Root(), "llm", "llm.json")
}
