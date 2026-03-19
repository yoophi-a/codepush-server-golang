package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

func TestMetricsWithMiniRedis(t *testing.T) {
	ctx := context.Background()
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	metrics := New(server.Addr(), "", 0)
	defer metrics.Close()

	if err := metrics.CheckHealth(ctx); err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if err := metrics.ReportDeploy(ctx, domain.DeploymentStatusReport{
		DeploymentKey:  "dep-key",
		ClientUniqueID: "client-1",
		Label:          "v1",
		Status:         deploySucceeded,
	}); err != nil {
		t.Fatalf("ReportDeploy() error = %v", err)
	}
	if err := metrics.ReportDeploy(ctx, domain.DeploymentStatusReport{
		DeploymentKey:  "dep-key",
		ClientUniqueID: "client-2",
		AppVersion:     "1.0.1",
		Status:         deployFailed,
	}); err != nil {
		t.Fatalf("ReportDeploy() error = %v", err)
	}
	if err := metrics.ReportDeploy(ctx, domain.DeploymentStatusReport{
		DeploymentKey:  "dep-key",
		ClientUniqueID: "client-1",
		Label:          "v2",
		Status:         "SomethingElse",
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
		t.Fatalf("unexpected v1 metrics %#v", got["v1"])
	}
	if got["v2"].Installed != 1 || got["v2"].Downloaded != 1 || got["v2"].Active != 1 {
		t.Fatalf("unexpected v2 metrics %#v", got["v2"])
	}
	if got["1.0.1"].Failed != 1 {
		t.Fatalf("expected fallback label failure metrics, got %#v", got["1.0.1"])
	}

	if err := metrics.Clear(ctx, "dep-key"); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	got, err = metrics.GetMetrics(ctx, "dep-key")
	if err != nil {
		t.Fatalf("GetMetrics() after clear error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty metrics after clear, got %#v", got)
	}
}

func TestHelpers(t *testing.T) {
	if labelsKey("dep") != "metrics:dep:labels" {
		t.Fatalf("unexpected labels key")
	}
	if countersKey("dep", "v1") != "metrics:dep:label:v1:counters" {
		t.Fatalf("unexpected counters key")
	}
	if activeSetKey("dep", "v1") != "metrics:dep:label:v1:active" {
		t.Fatalf("unexpected active set key")
	}
	if activeClientKey("dep", "client") != "metrics:dep:client:client" {
		t.Fatalf("unexpected active client key")
	}
	if parseCounter("") != 0 || parseCounter("bad") != 0 || parseCounter("12") != 12 {
		t.Fatalf("unexpected parseCounter behavior")
	}
}

func TestNewAppliesClientOptions(t *testing.T) {
	metrics := New("redis:6379", "secret", 2, ClientOptions{
		PoolSize:     41,
		MinIdleConns: 9,
		MaxRetries:   7,
		DialTimeout:  8 * time.Second,
		ReadTimeout:  4 * time.Second,
		WriteTimeout: 6 * time.Second,
	})
	defer metrics.Close()

	options := metrics.client.Options()
	if options.Addr != "redis:6379" || options.Password != "secret" || options.DB != 2 {
		t.Fatalf("unexpected base redis options: %#v", options)
	}
	if options.PoolSize != 41 || options.MinIdleConns != 9 || options.MaxRetries != 7 {
		t.Fatalf("unexpected pool options: %#v", options)
	}
	if options.DialTimeout != 8*time.Second || options.ReadTimeout != 4*time.Second || options.WriteTimeout != 6*time.Second {
		t.Fatalf("unexpected timeout options: %#v", options)
	}
}

func TestNewUsesDefaultClientOptionsForZeroValues(t *testing.T) {
	metrics := New("redis:6379", "", 0, ClientOptions{})
	defer metrics.Close()

	options := metrics.client.Options()
	if options.PoolSize != 20 || options.MinIdleConns != 5 || options.MaxRetries != 3 {
		t.Fatalf("unexpected default pool options: %#v", options)
	}
	if options.DialTimeout != 5*time.Second || options.ReadTimeout != 3*time.Second || options.WriteTimeout != 3*time.Second {
		t.Fatalf("unexpected default timeout options: %#v", options)
	}
}
