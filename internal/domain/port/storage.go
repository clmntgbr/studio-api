package port

import (
	"context"
	"time"
)

type UploadPartURL struct {
	PartNumber int32
	URL        string
}

type CompletedUploadPart struct {
	PartNumber int32
	ETag       string
}

// ObjectStorage is the MinIO/S3 port used by media upload commands.
type ObjectStorage interface {
	CreateMultipartUpload(ctx context.Context, bucket, key, contentType string) (uploadID string, err error)
	PresignUploadParts(ctx context.Context, bucket, key, uploadID string, partNumbers []int32, expiry time.Duration, publicBase string) ([]UploadPartURL, error)
	CompleteMultipartUpload(ctx context.Context, bucket, key, uploadID string, parts []CompletedUploadPart) error
	AbortMultipartUpload(ctx context.Context, bucket, key, uploadID string) error
	HeadObjectSize(ctx context.Context, bucket, key string) (int64, error)
}
