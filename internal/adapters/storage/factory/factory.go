package factory

import (
	"context"
	"fmt"
	"strings"

	miniostorage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/minio"
	s3storage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/s3"
	"github.com/yoophi/codepush-server-golang/internal/config"
	"github.com/yoophi/codepush-server-golang/internal/core/ports"
)

type Backend string

const (
	BackendS3    Backend = "s3"
	BackendMinIO Backend = "minio"
)

func normalizeBackend(name string) Backend {
	return Backend(strings.ToLower(strings.TrimSpace(name)))
}

func ValidateBackend(name string) error {
	switch normalizeBackend(name) {
	case BackendS3, BackendMinIO:
		return nil
	default:
		return fmt.Errorf("unsupported storage backend: %s", name)
	}
}

func New(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
	backend := normalizeBackend(cfg.StorageBackend)
	switch backend {
	case BackendMinIO:
		return miniostorage.New(cfg.MinIOEndpoint, cfg.MinIOAccessKeyID, cfg.MinIOSecretAccessKey, cfg.MinIOBucket, cfg.MinIOUseSSL)
	case BackendS3:
		return s3storage.New(ctx, cfg.S3Region, cfg.S3Endpoint, cfg.S3AccessKeyID, cfg.S3SecretAccessKey, cfg.S3Bucket, cfg.S3UsePathStyle)
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", cfg.StorageBackend)
	}
}
