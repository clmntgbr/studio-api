package media

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	domainproject "go-api/internal/domain/project"

	"github.com/google/uuid"
)

type InitiateUploadCommand struct {
	ProjectID uuid.UUID
	Filename  string
	MimeType  string
	SizeBytes int64
	PartCount int32
}

type InitiateUploadResult struct {
	MediaID  uuid.UUID
	UploadID string
	Bucket   string
	Key      string
	Parts    []port.UploadPartURL
}

type InitiateUploadHandler struct {
	projects domainproject.WriteRepository
	media    domainmedia.WriteRepository
	outbox   port.OutboxRepository
	storage  port.ObjectStorage
	bucket   string
	public   string
}

func NewInitiateUploadHandler(
	projects domainproject.WriteRepository,
	media domainmedia.WriteRepository,
	outbox port.OutboxRepository,
	storage port.ObjectStorage,
	bucket, publicEndpoint string,
) *InitiateUploadHandler {
	return &InitiateUploadHandler{
		projects: projects,
		media:    media,
		outbox:   outbox,
		storage:  storage,
		bucket:   bucket,
		public:   publicEndpoint,
	}
}

func (h *InitiateUploadHandler) Handle(ctx context.Context, cmd InitiateUploadCommand) (*InitiateUploadResult, error) {
	if cmd.Filename == "" {
		return nil, errors.New("filename is required")
	}
	if cmd.PartCount <= 0 {
		cmd.PartCount = 1
	}
	if cmd.PartCount > 10000 {
		return nil, errors.New("partCount too large")
	}

	project, err := h.projects.GetByID(ctx, cmd.ProjectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(cmd.Filename)), ".")
	if ext == "" {
		ext = "bin"
	}
	m := domainmedia.NewUpload(cmd.ProjectID, cmd.Filename, 0)
	if cmd.MimeType != "" {
		mt := cmd.MimeType
		m.MimeType = &mt
	}
	if cmd.SizeBytes > 0 {
		sz := cmd.SizeBytes
		m.SizeBytes = &sz
	}
	key := fmt.Sprintf("media/%s/source/original.%s", m.ID.String(), ext)
	m.StorageKey = &key

	cfg := domainmedia.DefaultConfiguration(m.ID)

	var uploadID string
	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.media.Save(txCtx, m); err != nil {
			return err
		}
		if err := h.media.SaveConfiguration(txCtx, cfg); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return nil, errors.New("failed to create media")
	}

	uploadID, err = h.storage.CreateMultipartUpload(ctx, h.bucket, key, cmd.MimeType)
	if err != nil {
		return nil, errors.New("failed to initiate multipart upload")
	}

	parts := make([]int32, 0, cmd.PartCount)
	for i := int32(1); i <= cmd.PartCount; i++ {
		parts = append(parts, i)
	}
	urls, err := h.storage.PresignUploadParts(ctx, h.bucket, key, uploadID, parts, 0, h.public)
	if err != nil {
		_ = h.storage.AbortMultipartUpload(ctx, h.bucket, key, uploadID)
		return nil, errors.New("failed to presign upload parts")
	}

	return &InitiateUploadResult{
		MediaID:  m.ID,
		UploadID: uploadID,
		Bucket:   h.bucket,
		Key:      key,
		Parts:    urls,
	}, nil
}
