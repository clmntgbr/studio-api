package transcribe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const assemblyAIBase = "https://api.assemblyai.com/v2"

type assemblyClient struct {
	apiKey string
	http   *http.Client
}

func newAssemblyClient(apiKey string) *assemblyClient {
	return &assemblyClient{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 120 * time.Second},
	}
}

type assemblyWord struct {
	Text       string  `json:"text"`
	Start      int64   `json:"start"`
	End        int64   `json:"end"`
	Confidence float64 `json:"confidence"`
	Speaker    *string `json:"speaker"`
}

type assemblyTranscript struct {
	ID       string         `json:"id"`
	Status   string         `json:"status"`
	Text     string         `json:"text"`
	Language string         `json:"language_code"`
	Words    []assemblyWord `json:"words"`
	Error    string         `json:"error"`
}

func (c *assemblyClient) uploadFile(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, assemblyAIBase+"/upload", f)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("assemblyai upload %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		UploadURL string `json:"upload_url"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.UploadURL == "" {
		return "", fmt.Errorf("assemblyai upload missing url")
	}
	return out.UploadURL, nil
}

func (c *assemblyClient) submit(ctx context.Context, audioURL, language string) (string, error) {
	payload := map[string]any{
		"audio_url":           audioURL,
		"punctuate":           true,
		"format_text":         true,
		"disfluencies":        true, // keep fillers like « euh »
		"speaker_labels":      false,
		"language_detection":  language == "",
	}
	if language != "" {
		payload["language_code"] = language
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, assemblyAIBase+"/transcript", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("assemblyai submit %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("assemblyai submit missing id")
	}
	return out.ID, nil
}

func (c *assemblyClient) get(ctx context.Context, id string) (*assemblyTranscript, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assemblyAIBase+"/transcript/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("assemblyai get %d: %s", resp.StatusCode, string(body))
	}
	var out assemblyTranscript
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *assemblyClient) wait(ctx context.Context, id string) (*assemblyTranscript, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		tr, err := c.get(ctx, id)
		if err != nil {
			return nil, err
		}
		switch tr.Status {
		case "completed":
			return tr, nil
		case "error":
			msg := tr.Error
			if msg == "" {
				msg = "transcription failed"
			}
			return nil, fmt.Errorf("%s", msg)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
