package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	cstorage "studyapp/backend/internal/common/storage"
)

const maxChunkObjectSize = 10 << 20 // 10MB per page

// Client resolves object keys to fetchable URLs and manages temporary uploads.
type Client interface {
	PresignedURL(ctx context.Context, objectRef string) (string, error)
	PutObject(ctx context.Context, key string, data []byte) error
	PutObjectStream(ctx context.Context, key, contentType string, reader io.Reader, sizeBytes int64) error
	GetObject(ctx context.Context, key string) ([]byte, error)
	DeleteObject(ctx context.Context, objectRef string) error
	DeletePrefix(ctx context.Context, prefix string) error
}

// NewClient returns an S3/MinIO client when configured, otherwise a local disk client for dev/tests.
func NewClient() (Client, error) {
	cfg := cstorage.LoadStorageConfig()
	if cfg.IsConfigured() {
		s3, err := cstorage.NewClient(cfg)
		if err != nil {
			return nil, err
		}
		if err := s3.EnsureBucket(context.Background()); err != nil {
			return nil, fmt.Errorf("storage bucket %q: %w", cfg.Bucket, err)
		}
		return &s3Adapter{client: s3}, nil
	}
	return NewLocalClient(), nil
}

// s3Adapter bridges the scan module to the common S3 client.
type s3Adapter struct {
	client *cstorage.Client
}

func (a *s3Adapter) PresignedURL(ctx context.Context, objectRef string) (string, error) {
	ref := strings.TrimSpace(objectRef)
	if ref == "" {
		return "", fmt.Errorf("empty object reference")
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, nil
	}
	if strings.HasPrefix(ref, "temp://") || strings.HasPrefix(ref, "stub://") {
		return ref, nil
	}
	return a.client.GetPresignedURL(ctx, ref, time.Hour)
}

func (a *s3Adapter) PutObject(ctx context.Context, key string, data []byte) error {
	if len(data) > maxChunkObjectSize {
		return fmt.Errorf("object exceeds max size of %d bytes", maxChunkObjectSize)
	}
	return a.client.PutObject(ctx, key, data)
}

func (a *s3Adapter) PutObjectStream(ctx context.Context, key, contentType string, reader io.Reader, sizeBytes int64) error {
	if sizeBytes > maxChunkObjectSize {
		return fmt.Errorf("object exceeds max size of %d bytes", maxChunkObjectSize)
	}
	return a.client.PutObjectStream(ctx, key, contentType, reader, sizeBytes)
}

func (a *s3Adapter) GetObject(ctx context.Context, key string) ([]byte, error) {
	return a.client.GetObject(ctx, key)
}

func (a *s3Adapter) DeleteObject(ctx context.Context, objectRef string) error {
	return a.client.DeleteObject(ctx, strings.TrimSpace(objectRef))
}

func (a *s3Adapter) DeletePrefix(ctx context.Context, prefix string) error {
	return a.client.DeletePrefix(ctx, prefix)
}

// LocalClient stores objects on disk for development and single-node deployments.
type LocalClient struct {
	baseDir   string
	publicURL string
}

func NewLocalClient() *LocalClient {
	base := strings.TrimSpace(os.Getenv("STORAGE_LOCAL_DIR"))
	if base == "" {
		base = filepath.Join(os.TempDir(), "studyapp-storage")
	}
	_ = os.MkdirAll(base, 0o755)

	publicURL := strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
	if publicURL == "" {
		publicURL = "http://localhost:8080"
	}
	publicURL = strings.TrimRight(publicURL, "/")

	return &LocalClient{baseDir: base, publicURL: publicURL}
}

func (c *LocalClient) objectPath(key string) string {
	clean := filepath.Clean(strings.ReplaceAll(key, "\\", "/"))
	clean = strings.TrimPrefix(clean, "/")
	return filepath.Join(c.baseDir, filepath.FromSlash(clean))
}

func (c *LocalClient) PresignedURL(ctx context.Context, objectRef string) (string, error) {
	_ = ctx
	ref := strings.TrimSpace(objectRef)
	if ref == "" {
		return "", fmt.Errorf("empty object reference")
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, nil
	}
	if strings.HasPrefix(ref, "temp://") || strings.HasPrefix(ref, "stub://") {
		return ref, nil
	}
	if strings.HasPrefix(ref, "temp/") {
		return fmt.Sprintf("%s/dev/storage/%s", c.publicURL, url.PathEscape(ref)), nil
	}
	return ref, nil
}

func (c *LocalClient) PutObject(ctx context.Context, key string, data []byte) error {
	_ = ctx
	if len(data) > maxChunkObjectSize {
		return fmt.Errorf("object exceeds max size of %d bytes", maxChunkObjectSize)
	}
	path := c.objectPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (c *LocalClient) PutObjectStream(ctx context.Context, key, contentType string, reader io.Reader, sizeBytes int64) error {
	_ = ctx
	_ = contentType
	data, err := io.ReadAll(io.LimitReader(reader, maxChunkObjectSize+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > maxChunkObjectSize {
		return fmt.Errorf("object exceeds max size of %d bytes", maxChunkObjectSize)
	}
	return c.PutObject(ctx, key, data)
}

func (c *LocalClient) GetObject(ctx context.Context, key string) ([]byte, error) {
	_ = ctx
	return os.ReadFile(c.objectPath(key))
}

func (c *LocalClient) DeleteObject(ctx context.Context, objectRef string) error {
	_ = ctx
	ref := strings.TrimSpace(objectRef)
	if ref == "" {
		return nil
	}
	path := c.objectPath(ref)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *LocalClient) DeletePrefix(ctx context.Context, prefix string) error {
	root := c.objectPath(prefix)
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		return os.Remove(path)
	})
}

// ServeObject writes a stored object to the HTTP response writer (local dev only).
func (c *LocalClient) ServeObject(w io.Writer, key string) error {
	data, err := c.GetObject(context.Background(), key)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// MaxPageUploadSize is the per-page upload limit enforced by chunk handlers.
func MaxPageUploadSize() int { return maxChunkObjectSize }
