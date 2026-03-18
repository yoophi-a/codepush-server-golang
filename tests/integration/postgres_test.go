//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/yoophi/codepush-server-golang/internal/adapters/persistence/postgres"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/testutil"
)

func TestPostgresRepositories(t *testing.T) {
	ctx := context.Background()
	stack, endpoints, err := testutil.StartStack(ctx)
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	defer stack.Terminate(ctx)

	store, err := postgres.NewStore(ctx, endpoints.DatabaseURL)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if err := store.Accounts().EnsureBootstrap(ctx, domain.Account{
		Email: "owner@example.com",
		Name:  "Owner",
	}, domain.AccessKey{
		Name:         "owner-token",
		FriendlyName: "Owner",
		Description:  "seed",
		CreatedBy:    "test",
		CreatedTime:  time.Now().UnixMilli(),
		Expires:      time.Now().Add(time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatalf("EnsureBootstrap() error = %v", err)
	}

	owner, err := store.Accounts().GetByEmail(ctx, "owner@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	app, err := store.Apps().Create(ctx, owner.ID, domain.App{Name: "demo"})
	if err != nil {
		t.Fatalf("Create app error = %v", err)
	}
	if _, err := store.Deployments().Create(ctx, owner.ID, app.ID, domain.Deployment{Name: "Production", Key: "dep-key"}); err != nil {
		t.Fatalf("Create deployment error = %v", err)
	}
	apps, err := store.Apps().List(ctx, owner.ID)
	if err != nil {
		t.Fatalf("List apps error = %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
}
