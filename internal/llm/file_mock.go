package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"go-api/internal/mocks/external"
)

// FileMockAnalyzer returns the recorded llm.json under MOCK_EXTERNAL_DIR.
// Falls back to heuristic MockAnalyzer if the file is missing.
type FileMockAnalyzer struct {
	Path string
	Fallback Analyzer

	once   sync.Once
	result *Result
	err    error
	useFB  bool
}

func NewFileMockAnalyzer(path string) *FileMockAnalyzer {
	if path == "" {
		path = external.LLMResultPath()
	}
	return &FileMockAnalyzer{Path: path, Fallback: NewMockAnalyzer()}
}

func (a *FileMockAnalyzer) Analyze(ctx context.Context, words []Word, tasks []string, language string) (*Result, error) {
	a.once.Do(func() {
		raw, err := os.ReadFile(a.Path)
		if err != nil {
			a.useFB = true
			a.err = err
			return
		}
		var out Result
		if err := json.Unmarshal(raw, &out); err != nil {
			a.err = fmt.Errorf("parse llm mock: %w", err)
			return
		}
		if out.Provider == "" {
			out.Provider = "mock"
		} else {
			out.Provider = "mock"
		}
		if out.Sponsors == nil {
			out.Sponsors = []SponsorHit{}
		}
		if out.Chapters == nil {
			out.Chapters = []ChapterHit{}
		}
		if out.Clips == nil {
			out.Clips = []ClipHit{}
		}
		a.result = &out
	})
	if a.useFB {
		return a.Fallback.Analyze(ctx, words, tasks, language)
	}
	if a.err != nil {
		return nil, a.err
	}
	want := taskSet(tasks)
	out := &Result{Provider: "mock", Sponsors: []SponsorHit{}, Chapters: []ChapterHit{}, Clips: []ClipHit{}}
	if want["sponsors"] {
		out.Sponsors = append(out.Sponsors, a.result.Sponsors...)
	}
	if want["chapters"] {
		out.Chapters = append(out.Chapters, a.result.Chapters...)
	}
	if want["viral"] {
		out.Clips = append(out.Clips, a.result.Clips...)
	}
	return out, nil
}
