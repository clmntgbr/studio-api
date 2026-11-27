package render

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

type DownloadQuery struct {
	ID          uuid.UUID
	Part        int // 0 = auto (single file or zip); 1..N = outputParts[part-1]
	RangeHeader string
}

type DownloadResult struct {
	Stream   *port.ObjectStream
	Filename string
}

type DownloadHandler struct {
	repo         domainrender.ReadRepository
	storage      port.ObjectStorage
	renderBucket string
}

func NewDownloadHandler(
	repo domainrender.ReadRepository,
	storage port.ObjectStorage,
	renderBucket string,
) *DownloadHandler {
	return &DownloadHandler{
		repo:         repo,
		storage:      storage,
		renderBucket: renderBucket,
	}
}

func (h *DownloadHandler) Handle(ctx context.Context, q DownloadQuery) (*DownloadResult, error) {
	r, err := h.repo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("render not found")
	}
	if r.Status != domainrender.StatusCompleted {
		return nil, errors.New("render is not ready")
	}

	// Explicit part request: always a single object.
	if q.Part > 0 {
		if q.Part > len(r.OutputParts) {
			return nil, errors.New("render part not found")
		}
		key := r.OutputParts[q.Part-1]
		filename := fmt.Sprintf("render-%s-part-%02d.mp4", r.ID.String(), q.Part)
		return h.openObject(ctx, key, filename, q.RangeHeader)
	}

	keys := collectDownloadKeys(r)
	if len(keys) == 0 {
		return nil, errors.New("render output missing")
	}
	if len(keys) == 1 {
		filename := fmt.Sprintf("render-%s.mp4", r.ID.String())
		return h.openObject(ctx, keys[0], filename, q.RangeHeader)
	}

	stream, err := h.zipObjects(ctx, keys)
	if err != nil {
		return nil, err
	}
	return &DownloadResult{
		Stream:   stream,
		Filename: fmt.Sprintf("render-%s.zip", r.ID.String()),
	}, nil
}

func collectDownloadKeys(r *domainrender.Render) []string {
	keys := make([]string, 0, 1+len(r.OutputParts))
	if r.StorageKey != nil {
		if k := strings.TrimSpace(*r.StorageKey); k != "" {
			keys = append(keys, k)
		}
	}
	for _, p := range r.OutputParts {
		if k := strings.TrimSpace(p); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

func (h *DownloadHandler) openObject(ctx context.Context, key, filename, rangeHeader string) (*DownloadResult, error) {
	stream, err := h.storage.GetObjectStream(ctx, h.renderBucket, key, rangeHeader)
	if err != nil {
		return nil, errors.New("failed to open render object")
	}
	if stream.ContentType == "" {
		stream.ContentType = "video/mp4"
	}
	return &DownloadResult{
		Stream:   stream,
		Filename: filename,
	}, nil
}

func (h *DownloadHandler) zipObjects(ctx context.Context, keys []string) (*port.ObjectStream, error) {
	pr, pw := io.Pipe()
	go func() {
		zw := zip.NewWriter(pw)
		var writeErr error
		defer func() {
			if writeErr != nil {
				_ = zw.Close()
				_ = pw.CloseWithError(writeErr)
				return
			}
			if err := zw.Close(); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()

		usedNames := map[string]int{}
		for _, key := range keys {
			if err := ctx.Err(); err != nil {
				writeErr = err
				return
			}
			obj, err := h.storage.GetObjectStream(ctx, h.renderBucket, key, "")
			if err != nil {
				writeErr = errors.New("failed to open render object")
				return
			}
			name := zipEntryName(key, usedNames)
			w, err := zw.Create(name)
			if err != nil {
				_ = obj.Body.Close()
				writeErr = err
				return
			}
			_, err = io.Copy(w, obj.Body)
			_ = obj.Body.Close()
			if err != nil {
				writeErr = err
				return
			}
		}
	}()

	return &port.ObjectStream{
		Body:          pr,
		ContentType:   "application/zip",
		ContentLength: -1, // unknown; omit Content-Length at HTTP layer when ≤0
		StatusCode:    200,
	}, nil
}

func zipEntryName(key string, used map[string]int) string {
	base := path.Base(key)
	if base == "" || base == "." || base == "/" {
		base = "part.mp4"
	}
	n := used[base]
	used[base] = n + 1
	if n == 0 {
		return base
	}
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return fmt.Sprintf("%s-%d%s", stem, n+1, ext)
}
