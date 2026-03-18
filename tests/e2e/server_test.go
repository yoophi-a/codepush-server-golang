//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpadapter "github.com/yoophi/codepush-server-golang/internal/adapters/http/ginhandler"
	redisadapter "github.com/yoophi/codepush-server-golang/internal/adapters/metrics/redis"
	"github.com/yoophi/codepush-server-golang/internal/adapters/persistence/postgres"
	"github.com/yoophi/codepush-server-golang/internal/application"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/testutil"
)

func TestManagementAndAcquisitionFlow(t *testing.T) {
	ctx := context.Background()
	stack, endpoints, err := testutil.StartStack(ctx)
	if err != nil {
		t.Skipf("skipping e2e test: %v", err)
	}
	defer stack.Terminate(ctx)

	store, err := postgres.NewStore(ctx, endpoints.DatabaseURL)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if err := store.Accounts().EnsureBootstrap(ctx, domain.Account{
		Email: "admin@example.com",
		Name:  "Admin",
	}, domain.AccessKey{
		Name:         "bootstrap-token",
		FriendlyName: "Bootstrap",
		Description:  "seed",
		CreatedBy:    "test",
		CreatedTime:  time.Now().UnixMilli(),
		Expires:      time.Now().Add(time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatalf("EnsureBootstrap() error = %v", err)
	}
	account, err := store.Accounts().GetByEmail(ctx, "admin@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	metrics := redisadapter.New(endpoints.RedisAddr, "", 0)
	defer metrics.Close()

	service := application.NewService(store.Accounts(), store.AccessKeys(), store.Apps(), store.Deployments(), store.Packages(), metrics)
	server := httptest.NewServer(httpadapter.NewRouter(service))
	defer server.Close()

	body, _ := json.Marshal(map[string]any{"name": "demo"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/apps", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer bootstrap-token")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create app request error = %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	app, err := store.Apps().GetByName(ctx, account.ID, "demo")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}
	deployment, err := store.Deployments().GetByName(ctx, account.ID, app.ID, "Production")
	if err != nil {
		t.Fatalf("GetByName deployment error = %v", err)
	}
	deployment.Key = "dep-key"
	if _, err := store.Pool().Exec(ctx, `UPDATE deployments SET deployment_key = $1 WHERE id = $2`, deployment.Key, deployment.ID); err != nil {
		t.Fatalf("update deployment key error = %v", err)
	}
	if _, err := store.Packages().CommitRollback(ctx, account.ID, app.ID, deployment.ID, domain.Package{
		AppVersion:    "1.0.0",
		Description:   "baseline",
		PackageHash:   "hash-1",
		BlobURL:       "http://example.com/v1.zip",
		Size:          10,
		UploadTime:    time.Now().UnixMilli(),
		ReleaseMethod: domain.ReleaseRollback,
	}); err != nil {
		t.Fatalf("seed package error = %v", err)
	}

	updateResp, err := http.Get(server.URL + "/updateCheck?deploymentKey=dep-key&appVersion=1.0.0")
	if err != nil {
		t.Fatalf("updateCheck request error = %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", updateResp.StatusCode)
	}
}
