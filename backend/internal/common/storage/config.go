package storage

import (
	"os"
	"strconv"
	"strings"
)

// StorageConfig holds S3/MinIO connection settings loaded from environment.
type StorageConfig struct {
	Endpoint  string // empty = AWS S3 default endpoint
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// LoadStorageConfig reads storage settings from environment variables.
func LoadStorageConfig() StorageConfig {
	useSSL := true
	if v := strings.TrimSpace(os.Getenv("STORAGE_USE_SSL")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			useSSL = parsed
		}
	}

	region := strings.TrimSpace(os.Getenv("STORAGE_REGION"))
	if region == "" {
		region = "us-east-1"
	}

	return StorageConfig{
		Endpoint:  strings.TrimSpace(os.Getenv("STORAGE_ENDPOINT")),
		Bucket:    strings.TrimSpace(os.Getenv("STORAGE_BUCKET")),
		Region:    region,
		AccessKey: strings.TrimSpace(os.Getenv("STORAGE_ACCESS_KEY")),
		SecretKey: strings.TrimSpace(os.Getenv("STORAGE_SECRET_KEY")),
		UseSSL:    useSSL,
	}
}

// IsConfigured reports whether S3/MinIO credentials and bucket are present.
func (c StorageConfig) IsConfigured() bool {
	return c.Bucket != "" && c.AccessKey != "" && c.SecretKey != ""
}
