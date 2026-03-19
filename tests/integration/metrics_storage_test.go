//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	redisadapter "github.com/yoophi/codepush-server-golang/internal/adapters/metrics/redis"
	miniostorage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/minio"
	s3storage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/s3"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/testutil"
)

func TestRedisMetricsAndStorageAdapters(t *testing.T) {
	ctx := context.Background()
	stack, endpoints, err := testutil.StartStack(ctx)
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	defer stack.Terminate(ctx)

	client, err := minio.New(endpoints.MinIOAddr, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("minio.New() error = %v", err)
	}
	if err := client.MakeBucket(ctx, "codepush", minio.MakeBucketOptions{}); err != nil {
		t.Fatalf("MakeBucket() error = %v", err)
	}

	t.Run("MinIOAndS3StorageContracts", func(t *testing.T) {
		minioStore, err := miniostorage.New(endpoints.MinIOAddr, "minioadmin", "minioadmin", "codepush", false)
		if err != nil {
			t.Fatalf("New minio storage error = %v", err)
		}
		if err := minioStore.CheckHealth(ctx); err != nil {
			t.Fatalf("CheckHealth() error = %v", err)
		}
		uri, err := minioStore.PutObject(ctx, "artifacts/demo.txt", []byte("hello"), "text/plain")
		if err != nil {
			t.Fatalf("PutObject() error = %v", err)
		}
		if uri != "minio://codepush/artifacts/demo.txt" {
			t.Fatalf("unexpected minio URI %q", uri)
		}
		if err := minioStore.DeleteObject(ctx, "artifacts/demo.txt"); err != nil {
			t.Fatalf("DeleteObject() error = %v", err)
		}

		s3Store, err := s3storage.New(ctx, "us-east-1", "http://"+endpoints.MinIOAddr, "minioadmin", "minioadmin", "codepush", true)
		if err != nil {
			t.Fatalf("New s3 storage error = %v", err)
		}
		if err := s3Store.CheckHealth(ctx); err != nil {
			t.Fatalf("S3 CheckHealth() error = %v", err)
		}
		uri, err = s3Store.PutObject(ctx, "artifacts/demo-s3.txt", []byte("hello"), "text/plain")
		if err != nil {
			t.Fatalf("S3 PutObject() error = %v", err)
		}
		if uri != "s3://codepush/artifacts/demo-s3.txt" {
			t.Fatalf("unexpected s3 URI %q", uri)
		}
		if err := s3Store.DeleteObject(ctx, "artifacts/demo-s3.txt"); err != nil {
			t.Fatalf("S3 DeleteObject() error = %v", err)
		}
	})

	t.Run("RedisMetricsTrackStateTransitions", func(t *testing.T) {
		metrics := redisadapter.New(endpoints.RedisAddr, "", 0)
		defer metrics.Close()

		if err := metrics.ReportDeploy(ctx, domain.DeploymentStatusReport{
			DeploymentKey:  "dep-key",
			ClientUniqueID: "client-1",
			Label:          "v1",
			Status:         "DeploymentSucceeded",
		}); err != nil {
			t.Fatalf("ReportDeploy() error = %v", err)
		}
		if err := metrics.ReportDeploy(ctx, domain.DeploymentStatusReport{
			DeploymentKey:  "dep-key",
			ClientUniqueID: "client-2",
			AppVersion:     "1.0.1",
			Status:         "DeploymentFailed",
		}); err != nil {
			t.Fatalf("ReportDeploy() error = %v", err)
		}
		if err := metrics.ReportDeploy(ctx, domain.DeploymentStatusReport{
			DeploymentKey:  "dep-key",
			ClientUniqueID: "client-1",
			Label:          "v2",
			Status:         "DeploymentSucceeded",
		}); err != nil {
			t.Fatalf("ReportDeploy() error = %v", err)
		}
		if err := metrics.IncrementDownload(ctx, "dep-key", "v1"); err != nil {
			t.Fatalf("IncrementDownload() error = %v", err)
		}
		if err := metrics.IncrementDownload(ctx, "dep-key", "v2"); err != nil {
			t.Fatalf("IncrementDownload() error = %v", err)
		}

		got, err := metrics.GetMetrics(ctx, "dep-key")
		if err != nil {
			t.Fatalf("GetMetrics() error = %v", err)
		}
		if got["v1"].Installed != 1 || got["v1"].Downloaded != 1 || got["v1"].Active != 0 {
			t.Fatalf("unexpected metrics for v1: %#v", got["v1"])
		}
		if got["v2"].Installed != 1 || got["v2"].Downloaded != 1 || got["v2"].Active != 1 {
			t.Fatalf("unexpected metrics for v2: %#v", got["v2"])
		}
		if got["1.0.1"].Failed != 1 {
			t.Fatalf("expected appVersion fallback label to track failures, got %#v", got["1.0.1"])
		}

		if err := metrics.Clear(ctx, "dep-key"); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}
		got, err = metrics.GetMetrics(ctx, "dep-key")
		if err != nil {
			t.Fatalf("GetMetrics() after clear error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected cleared metrics, got %#v", got)
		}
	})
}
