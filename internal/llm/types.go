package llm

import "context"

// Word is one timed transcript token (SourceTime).
type Word struct {
	Text          string `json:"text"`
	SourceStartMs int64  `json:"sourceStartMs"`
	SourceEndMs   int64  `json:"sourceEndMs"`
}

type SponsorHit struct {
	SourceStartMs int64   `json:"sourceStartMs"`
	SourceEndMs   int64   `json:"sourceEndMs"`
	FirstWord     *int    `json:"firstWord,omitempty"`
	LastWord      *int    `json:"lastWord,omitempty"`
	Confidence    float64 `json:"confidence"`
	Reason        string  `json:"reason"`
}

type ChapterHit struct {
	Title         string `json:"title"`
	SourceStartMs int64  `json:"sourceStartMs"`
	SourceEndMs   int64  `json:"sourceEndMs"`
}

type ClipHit struct {
	SourceStartMs int64   `json:"sourceStartMs"`
	SourceEndMs   int64   `json:"sourceEndMs"`
	Score         float64 `json:"score"`
	Hook          string  `json:"hook"`
	Reason        string  `json:"reason"`
}

type Result struct {
	Provider string       `json:"provider"`
	Sponsors []SponsorHit `json:"sponsors"`
	Chapters []ChapterHit `json:"chapters"`
	Clips    []ClipHit    `json:"clips"`
}

type Analyzer interface {
	Analyze(ctx context.Context, words []Word, tasks []string, language string) (*Result, error)
}
