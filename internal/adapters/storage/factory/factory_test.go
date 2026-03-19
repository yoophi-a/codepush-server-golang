package factory

import (
	"context"
	"testing"

	"github.com/yoophi/codepush-server-golang/internal/config"
)

func TestValidateBackend(t *testing.T) {
	for _, backend := range []string{"s3", "S3", " minio "} {
		if err := ValidateBackend(backend); err != nil {
			t.Fatalf("ValidateBackend(%q) error = %v", backend, err)
		}
	}
	if err := ValidateBackend("azure"); err == nil {
		t.Fatalf("expected unsupported backend error")
	}
}

func TestNewCreatesStorageForSupportedBackends(t *testing.T) {
	ctx := context.Background()

	minioStorage, err := New(ctx, config.Config{
		StorageBackend:       "minio",
		MinIOEndpoint:        "localhost:9000",
		MinIOAccessKeyID:     "minioadmin",
		MinIOSecretAccessKey: "minioadmin",
		MinIOBucket:          "codepush",
	})
	if err != nil || minioStorage == nil {
		t.Fatalf("expected minio storage, got %T err=%v", minioStorage, err)
	}

	s3Storage, err := New(ctx, config.Config{
		StorageBackend:    "s3",
		S3Region:          "us-east-1",
		S3Endpoint:        "http://localhost:9000",
		S3AccessKeyID:     "minioadmin",
		S3SecretAccessKey: "minioadmin",
		S3Bucket:          "codepush",
		S3UsePathStyle:    true,
	})
	if err != nil || s3Storage == nil {
		t.Fatalf("expected s3 storage, got %T err=%v", s3Storage, err)
	}
}

func TestNewRejectsUnsupportedBackend(t *testing.T) {
	storage, err := New(context.Background(), config.Config{StorageBackend: "azure"})
	if err == nil || storage != nil {
		t.Fatalf("expected unsupported backend error, got storage=%T err=%v", storage, err)
	}
}
