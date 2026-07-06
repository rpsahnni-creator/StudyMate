package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// EnsureBucket creates the configured bucket when missing (MinIO local dev / first deploy).
func (c *Client) EnsureBucket(ctx context.Context) error {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err == nil {
		return nil
	}

	_, createErr := c.s3.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if createErr == nil {
		return nil
	}

	var exists *types.BucketAlreadyExists
	var owned *types.BucketAlreadyOwnedByYou
	if errors.As(createErr, &exists) || errors.As(createErr, &owned) {
		return nil
	}
	if strings.Contains(strings.ToLower(createErr.Error()), "already owned") ||
		strings.Contains(strings.ToLower(createErr.Error()), "already exists") {
		return nil
	}
	return fmt.Errorf("create bucket %q: %w", c.bucket, createErr)
}
