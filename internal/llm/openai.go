package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatibleAnalyzer calls a Chat Completions API (OpenAI, DeepSeek, …).
type OpenAICompatibleAnalyzer struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
	Provider   string
}

func NewOpenAICompatibleAnalyzer(baseURL, apiKey, model, provider string) *OpenAICompatibleAnalyzer {
	if provider == "" {
		provider = "openai"
	}
	if baseURL == "" {
		switch provider {
		case "grok", "xai":
			baseURL = "https://api.x.ai/v1"
		case "deepseek":
			baseURL = "https://api.deepseek.com/v1"
		default:
			baseURL = "https://api.openai.com/v1"
		}
	}
	if model == "" {
		switch provider {
		case "grok", "xai":
			model = "grok-3-mini"
		case "deepseek":
			model = "deepseek-chat"
		default:
			model = "gpt-4o-mini"
		}
	}
	return &OpenAICompatibleAnalyzer{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     apiKey,
		Model:      model,
		Provider:   provider,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat *chatFormat   `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type llmJSON struct {
	Sponsors []struct {
		StartMs    int64   `json:"startMs"`
		EndMs      int64   `json:"endMs"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	} `json:"sponsors"`
	Chapters []struct {
		Title   string `json:"title"`
		StartMs int64  `json:"startMs"`
		EndMs   int64  `json:"endMs"`
	} `json:"chapters"`
	Clips []struct {
		StartMs int64   `json:"startMs"`
		EndMs   int64   `json:"endMs"`
		Score   float64 `json:"score"`
		Hook    string  `json:"hook"`
		Reason  string  `json:"reason"`
	} `json:"clips"`
}

func (a *OpenAICompatibleAnalyzer) Analyze(ctx context.Context, words []Word, tasks []string, language string) (*Result, error) {
	if a.APIKey == "" {
		return nil, fmt.Errorf("LLM API key is required")
	}
	want := taskSet(tasks)
	prompt := buildPrompt(words, want, language)
	body, _ := json.Marshal(chatRequest{
		Model: a.Model,
		Messages: []chatMessage{
			{Role: "system", Content: "You analyze timed transcripts. Reply with JSON only matching the schema."},
			{Role: "user", Content: prompt},
		},
		Temperature:    0.2,
		ResponseFormat: &chatFormat{Type: "json_object"},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("llm http %d: %s", res.StatusCode, truncate(string(raw), 400))
	}
	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("llm empty choices")
	}
	var payload llmJSON
	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &payload); err != nil {
		return nil, fmt.Errorf("llm invalid json: %w", err)
	}
	out := &Result{Provider: a.Provider, Sponsors: []SponsorHit{}, Chapters: []ChapterHit{}, Clips: []ClipHit{}}
	if want["sponsors"] {
		for _, s := range payload.Sponsors {
			first, last, ok := ResolveBounds(words, s.StartMs, s.EndMs)
			if !ok {
				continue
			}
			fi, li := first, last
			conf := s.Confidence
			if conf <= 0 {
				conf = 0.7
			}
			out.Sponsors = append(out.Sponsors, SponsorHit{
				SourceStartMs: words[first].SourceStartMs,
				SourceEndMs:   words[last].SourceEndMs,
				FirstWord:     &fi,
				LastWord:      &li,
				Confidence:    conf,
				Reason:        s.Reason,
			})
		}
	}
	if want["chapters"] {
		for _, c := range payload.Chapters {
			first, last, ok := ResolveBounds(words, c.StartMs, c.EndMs)
			if !ok {
				continue
			}
			title := strings.TrimSpace(c.Title)
			if title == "" {
				title = "Chapter"
			}
			out.Chapters = append(out.Chapters, ChapterHit{
				Title:         title,
				SourceStartMs: words[first].SourceStartMs,
				SourceEndMs:   words[last].SourceEndMs,
			})
		}
	}
	if want["viral"] {
		for _, c := range payload.Clips {
			first, last, ok := ResolveBounds(words, c.StartMs, c.EndMs)
			if !ok {
				continue
			}
			score := c.Score
			if score < 0 {
				score = 0
			}
			if score > 1 {
				score = 1
			}
			out.Clips = append(out.Clips, ClipHit{
				SourceStartMs: words[first].SourceStartMs,
				SourceEndMs:   words[last].SourceEndMs,
				Score:         score,
				Hook:          c.Hook,
				Reason:        c.Reason,
			})
		}
	}
	return out, nil
}

func buildPrompt(words []Word, want map[string]bool, language string) string {
	var b strings.Builder
	b.WriteString("Transcript lines as [startMs-endMs] text. Language hint: ")
	if language == "" {
		language = "auto"
	}
	b.WriteString(language)
	b.WriteString("\nTasks: ")
	tasks := make([]string, 0, 3)
	if want["sponsors"] {
		tasks = append(tasks, "sponsors")
	}
	if want["chapters"] {
		tasks = append(tasks, "chapters")
	}
	if want["viral"] {
		tasks = append(tasks, "viral clips")
	}
	b.WriteString(strings.Join(tasks, ", "))
	b.WriteString("\nReturn JSON: {\"sponsors\":[{\"startMs\":0,\"endMs\":0,\"confidence\":0.8,\"reason\":\"\"}],\"chapters\":[{\"title\":\"\",\"startMs\":0,\"endMs\":0}],\"clips\":[{\"startMs\":0,\"endMs\":0,\"score\":0.7,\"hook\":\"\",\"reason\":\"\"}]}\nOmit unused arrays as [].\n\n")
	limit := len(words)
	if limit > 2500 {
		limit = 2500
	}
	for i := 0; i < limit; i++ {
		w := words[i]
		fmt.Fprintf(&b, "[%d-%d] %s\n", w.SourceStartMs, w.SourceEndMs, w.Text)
	}
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
