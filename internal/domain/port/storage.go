package port

import (
	"context"
	"io"
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

// ObjectStream is a ranged object body from storage (for authenticated API proxy).
type ObjectStream struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
	ContentRange  string // set when Range request returns 206
	AcceptRanges  bool
	StatusCode    int // 200 or 206
}

// ObjectStorage is the MinIO/S3 port used by media upload commands.
type ObjectStorage interface {
	CreateMultipartUpload(ctx context.Context, bucket, key, contentType string) (uploadID string, err error)
	PresignUploadParts(ctx context.Context, bucket, key, uploadID string, partNumbers []int32, expiry time.Duration, publicBase string) ([]UploadPartURL, error)
	CompleteMultipartUpload(ctx context.Context, bucket, key, uploadID string, parts []CompletedUploadPart) error
	AbortMultipartUpload(ctx context.Context, bucket, key, uploadID string) error
	HeadObjectSize(ctx context.Context, bucket, key string) (int64, error)
	PutObject(ctx context.Context, bucket, key string, body []byte, contentType string) error
	PresignGetObject(ctx context.Context, bucket, key string, expiry time.Duration, publicBase string) (string, error)
	GetObjectStream(ctx context.Context, bucket, key, rangeHeader string) (*ObjectStream, error)
}
