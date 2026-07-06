package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	tempPrefix        = "temp/"
	quizAssetPrefix   = "quiz-assets/"
	expiryIntentMeta  = "x-expiry-intent"
	expiryIntentValue = "24h"
	multipartThreshold = 5 << 20 // 5MB
	defaultPresignTTL = time.Hour
)

// Client wraps an S3-compatible object store (AWS S3 or MinIO).
type Client struct {
	bucket   string
	s3       *s3.Client
	presign  *s3.PresignClient
	uploader *manager.Uploader
}

// NewClient creates an S3/MinIO client from the given configuration.
func NewClient(cfg StorageConfig) (*Client, error) {
	if !cfg.IsConfigured() {
		return nil, fmt.Errorf("storage config incomplete: bucket, access key, and secret key are required")
	}

	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	s3Opts := func(o *s3.Options) {
		if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
		o.UseAccelerate = false
	}

	s3Client := s3.NewFromConfig(awsCfg, s3Opts)
	return &Client{
		bucket:   cfg.Bucket,
		s3:       s3Client,
		presign:  s3.NewPresignClient(s3Client),
		uploader: manager.NewUploader(s3Client, func(u *manager.Uploader) {
			u.PartSize = multipartThreshold
			u.Concurrency = 4
		}),
	}, nil
}

// UploadTemp uploads a temporary object under the temp/ prefix with 24h expiry intent metadata.
// Returns a presigned GET URL valid for 1 hour (for OCR worker access).
func (c *Client) UploadTemp(ctx context.Context, key, contentType string, reader io.Reader, sizeBytes int64) (string, error) {
	fullKey := ensureTempPrefix(key)
	if err := c.putObject(ctx, fullKey, contentType, reader, sizeBytes, map[string]string{
		expiryIntentMeta: expiryIntentValue,
	}); err != nil {
		return "", err
	}
	return c.GetPresignedURL(ctx, fullKey, defaultPresignTTL)
}

// UploadQuizAsset uploads a permanent quiz image under quiz-assets/.
func (c *Client) UploadQuizAsset(ctx context.Context, key string, reader io.Reader) (string, error) {
	fullKey := key
	if !strings.HasPrefix(fullKey, quizAssetPrefix) {
		fullKey = quizAssetPrefix + strings.TrimPrefix(fullKey, "/")
	}
	if err := c.putObject(ctx, fullKey, "image/jpeg", reader, -1, nil); err != nil {
		return "", err
	}
	return fullKey, nil
}

// PutObject uploads raw bytes (used for chunk parts).
func (c *Client) PutObject(ctx context.Context, key string, data []byte) error {
	return c.putObject(ctx, key, "application/octet-stream", bytes.NewReader(data), int64(len(data)), nil)
}

// PutObjectStream uploads from a reader with optional content type.
func (c *Client) PutObjectStream(ctx context.Context, key, contentType string, reader io.Reader, sizeBytes int64) error {
	return c.putObject(ctx, key, contentType, reader, sizeBytes, map[string]string{
		expiryIntentMeta: expiryIntentValue,
	})
}

func (c *Client) putObject(ctx context.Context, key, contentType string, reader io.Reader, sizeBytes int64, metadata map[string]string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	}
	if len(metadata) > 0 {
		input.Metadata = metadata
	}
	if sizeBytes >= 0 {
		input.ContentLength = aws.Int64(sizeBytes)
	}

	if sizeBytes > multipartThreshold || sizeBytes < 0 {
		_, err := c.uploader.Upload(ctx, input)
		return err
	}
	_, err := c.s3.PutObject(ctx, input)
	return err
}

// GetObject downloads an object by key.
func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

// DeleteObject removes a single object.
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

// DeletePrefix removes all objects under a prefix.
func (c *Client) DeletePrefix(ctx context.Context, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		if len(page.Contents) == 0 {
			continue
		}
		objects := make([]types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			objects[i] = types.ObjectIdentifier{Key: obj.Key}
		}
		_, err = c.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(c.bucket),
			Delete: &types.Delete{Objects: objects},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPresignedURL returns a presigned GET URL for the given object key.
func (c *Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = defaultPresignTTL
	}
	out, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func ensureTempPrefix(key string) string {
	if strings.HasPrefix(key, tempPrefix) {
		return key
	}
	return tempPrefix + strings.TrimPrefix(key, "/")
}
