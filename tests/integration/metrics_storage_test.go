//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	redisadapter "github.com/yoophi/codepush-server-golang/internal/adapters/metrics/redis"
	miniostorage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/minio"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/testutil"
)

func TestRedisMetricsAndMinIO(t *testing.T) {
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

	storage, err := miniostorage.New(endpoints.MinIOAddr, "minioadmin", "minioadmin", "codepush", false)
	if err != nil {
		t.Fatalf("New minio storage error = %v", err)
	}
	if err := client.MakeBucket(ctx, "codepush", minio.MakeBucketOptions{}); err != nil {
		t.Fatalf("MakeBucket() error = %v", err)
	}
	if err := storage.CheckHealth(ctx); err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if _, err := storage.PutObject(ctx, "artifacts/demo.txt", []byte("hello"), "text/plain"); err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}

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
	if err := metrics.IncrementDownload(ctx, "dep-key", "v1"); err != nil {
		t.Fatalf("IncrementDownload() error = %v", err)
	}
	got, err := metrics.GetMetrics(ctx, "dep-key")
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}
	if got["v1"].Installed != 1 || got["v1"].Downloaded != 1 {
		t.Fatalf("unexpected metrics: %#v", got["v1"])
	}
}
