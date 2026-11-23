package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type YTDLPConfig struct {
	Bin           string
	MaxFilesizeMB int
	MaxDurationS  int
	CookiesFile   string
}

type downloadMeta struct {
	Title     string `json:"title"`
	Uploader  string `json:"uploader"`
	Extractor string `json:"extractor"`
	Ext       string `json:"ext"`
	URL       string `json:"webpage_url"`
}

func (c YTDLPConfig) bin() string {
	if c.Bin != "" {
		return c.Bin
	}
	return "yt-dlp"
}

// DownloadURL downloads a public URL into workDir and returns the local file path + origin metadata.
func DownloadURL(ctx context.Context, cfg YTDLPConfig, rawURL, workDir string) (localPath string, origin *OriginOut, err error) {
	if err := ValidatePublicHTTPURL(rawURL); err != nil {
		return "", nil, err
	}

	maxMB := cfg.MaxFilesizeMB
	if maxMB <= 0 {
		maxMB = 8000
	}
	_ = cfg.MaxDurationS // enforced after download via ffprobe in the pipeline

	outTemplate := filepath.Join(workDir, "original.%(ext)s")
	args := []string{
		"--no-playlist",
		"--no-mtime",
		"--newline",
		"--max-filesize", fmt.Sprintf("%dM", maxMB),
		"-f", "bv*+ba/b",
		"--merge-output-format", "mp4",
		"-o", outTemplate,
		"--print-json",
		rawURL,
	}
	if cfg.CookiesFile != "" {
		args = append([]string{"--cookies", cfg.CookiesFile}, args...)
	}

	cmd := exec.CommandContext(ctx, cfg.bin(), args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("yt-dlp failed: %w: %s", err, truncate(string(out), 2000))
	}

	meta, path, err := findDownload(workDir, out)
	if err != nil {
		return "", nil, err
	}
	origin = &OriginOut{
		URL:       rawURL,
		Title:     meta.Title,
		Uploader:  meta.Uploader,
		Extractor: meta.Extractor,
	}
	if meta.URL != "" {
		origin.URL = meta.URL
	}
	return path, origin, nil
}

func findDownload(workDir string, ytOut []byte) (*downloadMeta, string, error) {
	var meta downloadMeta
	lines := strings.Split(string(ytOut), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || line[0] != '{' {
			continue
		}
		if err := json.Unmarshal([]byte(line), &meta); err == nil {
			break
		}
	}

	entries, err := os.ReadDir(workDir)
	if err != nil {
		return nil, "", err
	}
	var preferred string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".part") || strings.HasSuffix(lower, ".ytdl") || strings.Contains(lower, ".f") {
			continue
		}
		if !strings.HasPrefix(name, "original.") {
			continue
		}
		path := filepath.Join(workDir, name)
		if strings.EqualFold(filepath.Ext(name), ".mp4") {
			return &meta, path, nil
		}
		if preferred == "" {
			preferred = path
		}
	}
	if preferred != "" {
		return &meta, preferred, nil
	}
	return nil, "", fmt.Errorf("yt-dlp produced no original.* file in %s", workDir)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
