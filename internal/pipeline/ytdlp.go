package pipeline

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
	// MaxHeight caps progressive download resolution (0 → 1080). Avoids 4K+ when proxy/ship do not need it.
	MaxHeight   int
	CookiesFile string
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

// DownloadURL downloads a public URL into workDir and returns local path + origin metadata.
func DownloadURL(ctx context.Context, cfg YTDLPConfig, rawURL, workDir string) (string, *Origin, error) {
	if err := ValidatePublicHTTPURL(rawURL); err != nil {
		return "", nil, err
	}

	maxMB := cfg.MaxFilesizeMB
	if maxMB <= 0 {
		maxMB = 8000
	}
	maxH := cfg.MaxHeight
	if maxH <= 0 {
		maxH = 1080
	}
	// Cap the short side at MaxHeight so landscape (≤1080h) and portrait/Shorts (≤1080w) both stay ~1080p-class.
	format := fmt.Sprintf(
		"bv*[height<=%[1]d]+ba/bv*[width<=%[1]d]+ba/b[height<=%[1]d]/b[width<=%[1]d]",
		maxH,
	)

	outTemplate := filepath.Join(workDir, "original.%(ext)s")
	args := []string{
		"--no-playlist",
		"--no-mtime",
		"--newline",
		"--max-filesize", fmt.Sprintf("%dM", maxMB),
		"-f", format,
		"-S", "res,ext:mp4:m4a",
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
	origin := &Origin{
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
		if strings.HasSuffix(lower, ".part") || strings.HasSuffix(lower, ".ytdl") {
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
