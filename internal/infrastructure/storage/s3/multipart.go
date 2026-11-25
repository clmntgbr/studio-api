package s3

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go-api/internal/domain/port"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (c *Client) CreateMultipartUpload(ctx context.Context, bucket, key, contentType string) (uploadID string, err error) {
	in := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	out, err := c.inner.CreateMultipartUpload(ctx, in)
	if err != nil {
		return "", fmt.Errorf("create multipart: %w", err)
	}
	if out.UploadId == nil || *out.UploadId == "" {
		return "", fmt.Errorf("create multipart: empty upload id")
	}
	return *out.UploadId, nil
}

func (c *Client) PresignUploadParts(
	ctx context.Context,
	bucket, key, uploadID string,
	partNumbers []int32,
	expiry time.Duration,
	publicBase string,
) ([]port.UploadPartURL, error) {
	presigner := s3.NewPresignClient(c.inner)
	if expiry <= 0 {
		expiry = time.Hour
	}
	out := make([]port.UploadPartURL, 0, len(partNumbers))
	for _, n := range partNumbers {
		res, err := presigner.PresignUploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String(key),
			UploadId:   aws.String(uploadID),
			PartNumber: aws.Int32(n),
		}, s3.WithPresignExpires(expiry))
		if err != nil {
			return nil, fmt.Errorf("presign part %d: %w", n, err)
		}
		out = append(out, port.UploadPartURL{
			PartNumber: n,
			URL:        rewritePublicURL(res.URL, publicBase),
		})
	}
	return out, nil
}

func (c *Client) CompleteMultipartUpload(
	ctx context.Context,
	bucket, key, uploadID string,
	parts []port.CompletedUploadPart,
) error {
	completed := make([]types.CompletedPart, 0, len(parts))
	for _, p := range parts {
		etag := p.ETag
		completed = append(completed, types.CompletedPart{
			ETag:       aws.String(etag),
			PartNumber: aws.Int32(p.PartNumber),
		})
	}
	_, err := c.inner.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completed,
		},
	})
	if err != nil {
		return fmt.Errorf("complete multipart: %w", err)
	}
	return nil
}

func (c *Client) AbortMultipartUpload(ctx context.Context, bucket, key, uploadID string) error {
	_, err := c.inner.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		return fmt.Errorf("abort multipart: %w", err)
	}
	return nil
}

func (c *Client) HeadObjectSize(ctx context.Context, bucket, key string) (int64, error) {
	out, err := c.inner.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, fmt.Errorf("head object: %w", err)
	}
	if out.ContentLength == nil {
		return 0, nil
	}
	return *out.ContentLength, nil
}

func (c *Client) PutObject(ctx context.Context, bucket, key string, body []byte, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if _, err := c.inner.PutObject(ctx, input); err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", bucket, key, err)
	}
	return nil
}

func (c *Client) PresignGetObject(ctx context.Context, bucket, key string, expiry time.Duration, publicBase string) (string, error) {
	presigner := s3.NewPresignClient(c.inner)
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	res, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign get s3://%s/%s: %w", bucket, key, err)
	}
	return rewritePublicURL(res.URL, publicBase), nil
}

func (c *Client) GetObjectStream(ctx context.Context, bucket, key, rangeHeader string) (*port.ObjectStream, error) {
	in := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if rangeHeader != "" {
		in.Range = aws.String(rangeHeader)
	}
	out, err := c.inner.GetObject(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("get s3://%s/%s: %w", bucket, key, err)
	}

	stream := &port.ObjectStream{
		Body:         out.Body,
		AcceptRanges: true,
		StatusCode:   200,
	}
	if out.ContentType != nil {
		stream.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		stream.ContentLength = *out.ContentLength
	}
	if out.ContentRange != nil && *out.ContentRange != "" {
		stream.ContentRange = *out.ContentRange
		stream.StatusCode = 206
	}
	if stream.ContentType == "" {
		stream.ContentType = "application/octet-stream"
	}
	return stream, nil
}

func rewritePublicURL(signed, publicBase string) string {
	publicBase = strings.TrimRight(publicBase, "/")
	if publicBase == "" {
		return signed
	}
	u, err := url.Parse(signed)
	if err != nil {
		return signed
	}
	pub, err := url.Parse(publicBase)
	if err != nil {
		return signed
	}
	u.Scheme = pub.Scheme
	u.Host = pub.Host
	return u.String()
}
