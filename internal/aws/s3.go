package aws

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 is a minimal object-store helper bound to a single bucket.
type S3 struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// NewS3 builds an S3 helper for bucket from a loaded AWS config.
func NewS3(cfg aws.Config, bucket string) *S3 {
	client := s3.NewFromConfig(cfg)
	return &S3{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  bucket,
	}
}

// Put uploads body under key with the given content type.
func (s *S3) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	in := &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   body,
	}
	if contentType != "" {
		in.ContentType = &contentType
	}
	if _, err := s.client.PutObject(ctx, in); err != nil {
		return fmt.Errorf("s3: put %q: %w", key, err)
	}
	return nil
}

// PresignGet returns a time-limited URL to GET key, valid for ttl.
func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	out, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("s3: presign get %q: %w", key, err)
	}
	return out.URL, nil
}
