//go:build integration

package integration

import (
	"context"
	"errors"
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

	owner := seedBootstrapAccount(t, ctx, store, "owner@example.com", "owner-token")
	collaborator := seedBootstrapAccount(t, ctx, store, "collab@example.com", "collab-token")
	otherOwner := seedBootstrapAccount(t, ctx, store, "other-owner@example.com", "other-owner-token")

	t.Run("EnsureBootstrapIsIdempotentAndResolveAccessKey", func(t *testing.T) {
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

		keys, err := store.AccessKeys().List(ctx, owner.ID)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(keys) != 1 {
			t.Fatalf("expected a single bootstrap key, got %d", len(keys))
		}

		accountID, err := store.Accounts().ResolveAccountIDByAccessKey(ctx, "owner-token")
		if err != nil {
			t.Fatalf("ResolveAccountIDByAccessKey() error = %v", err)
		}
		if accountID != owner.ID {
			t.Fatalf("expected resolved account %q, got %q", owner.ID, accountID)
		}

		_, err = store.Accounts().ResolveAccountIDByAccessKey(ctx, "missing-token")
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}

		expired, err := store.AccessKeys().Create(ctx, domain.AccessKey{
			AccountID:    owner.ID,
			FriendlyName: "expired",
			Description:  "expired",
			CreatedBy:    "test",
			CreatedTime:  time.Now().Add(-2 * time.Hour).UnixMilli(),
			Expires:      time.Now().Add(-time.Hour).UnixMilli(),
		}, "expired-token")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if expired.ID == "" {
			t.Fatalf("expected expired token row to be created")
		}
		_, err = store.Accounts().ResolveAccountIDByAccessKey(ctx, "expired-token")
		if !errors.Is(err, domain.ErrExpired) {
			t.Fatalf("expected ErrExpired, got %v", err)
		}
	})

	t.Run("AccessKeyRepoHandlesConflictAndMissingDelete", func(t *testing.T) {
		_, err := store.AccessKeys().Create(ctx, domain.AccessKey{
			AccountID:    owner.ID,
			FriendlyName: "duplicate",
			Description:  "duplicate",
			CreatedBy:    "test",
			CreatedTime:  time.Now().UnixMilli(),
			Expires:      time.Now().Add(time.Hour).UnixMilli(),
		}, "owner-token")
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}

		if err := store.AccessKeys().Delete(ctx, owner.ID, "missing-token"); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}

		rows, err := store.AccessKeys().DeleteSessionsByCreator(ctx, owner.ID, "nobody")
		if err != nil {
			t.Fatalf("DeleteSessionsByCreator() error = %v", err)
		}
		if rows != 0 {
			t.Fatalf("expected zero deleted sessions, got %d", rows)
		}
	})

	t.Run("AppNameUniquenessAndOwnerSelection", func(t *testing.T) {
		ownedApp, err := store.Apps().Create(ctx, owner.ID, domain.App{Name: "demo"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if _, err := store.Deployments().Create(ctx, owner.ID, ownedApp.ID, domain.Deployment{Name: "Production", Key: "dep-owned"}); err != nil {
			t.Fatalf("Create deployment error = %v", err)
		}

		if _, err := store.Apps().Create(ctx, owner.ID, domain.App{Name: "demo"}); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}

		otherApp, err := store.Apps().Create(ctx, otherOwner.ID, domain.App{Name: "demo"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if err := store.Apps().AddCollaborator(ctx, otherOwner.ID, otherApp.ID, owner.Email); err != nil {
			t.Fatalf("AddCollaborator() error = %v", err)
		}

		got, err := store.Apps().GetByName(ctx, owner.ID, "demo")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}
		if got.ID != ownedApp.ID {
			t.Fatalf("expected owned app to be preferred, got %q", got.ID)
		}

		got, err = store.Apps().GetByName(ctx, owner.ID, otherOwner.Email+":demo")
		if err != nil {
			t.Fatalf("GetByName() with owner prefix error = %v", err)
		}
		if got.ID != otherApp.ID {
			t.Fatalf("expected owner-qualified app, got %q", got.ID)
		}

		apps, err := store.Apps().List(ctx, owner.ID)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		byID := map[string]domain.App{}
		for _, app := range apps {
			byID[app.ID] = app
		}
		if byID[ownedApp.ID].Deployments[0] != "Production" {
			t.Fatalf("expected batched deployments for owned app, got %#v", byID[ownedApp.ID])
		}
		if byID[ownedApp.ID].Collaborators[owner.Email].Permission != domain.PermissionOwner {
			t.Fatalf("expected owner collaborator metadata, got %#v", byID[ownedApp.ID].Collaborators)
		}
		if byID[otherApp.ID].Collaborators[owner.Email].Permission != domain.PermissionCollaborator {
			t.Fatalf("expected collaborator metadata on shared app, got %#v", byID[otherApp.ID].Collaborators)
		}
	})

	t.Run("TransferAndCollaboratorRemovalRules", func(t *testing.T) {
		app, err := store.Apps().Create(ctx, owner.ID, domain.App{Name: "transfer-demo"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if err := store.Apps().AddCollaborator(ctx, owner.ID, app.ID, collaborator.Email); err != nil {
			t.Fatalf("AddCollaborator() error = %v", err)
		}
		if err := store.Apps().Transfer(ctx, owner.ID, app.ID, collaborator.Email); err != nil {
			t.Fatalf("Transfer() error = %v", err)
		}

		collabs, err := store.Apps().ListCollaborators(ctx, owner.ID, app.ID)
		if err != nil {
			t.Fatalf("ListCollaborators() error = %v", err)
		}
		if collabs[owner.Email].Permission != domain.PermissionCollaborator {
			t.Fatalf("expected original owner to become collaborator, got %#v", collabs[owner.Email])
		}
		if collabs[collaborator.Email].Permission != domain.PermissionOwner {
			t.Fatalf("expected transferred owner role, got %#v", collabs[collaborator.Email])
		}

		if err := store.Apps().RemoveCollaborator(ctx, collaborator.ID, app.ID, collaborator.Email); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected owner self-removal to be forbidden, got %v", err)
		}
		if err := store.Apps().RemoveCollaborator(ctx, owner.ID, app.ID, owner.Email); err != nil {
			t.Fatalf("expected collaborator self-removal to succeed, got %v", err)
		}
	})

	t.Run("DeploymentPermissionsAndPackageOrdinal", func(t *testing.T) {
		app, err := store.Apps().Create(ctx, owner.ID, domain.App{Name: "package-demo"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		deployment, err := store.Deployments().Create(ctx, owner.ID, app.ID, domain.Deployment{Name: "Production", Key: "pkg-dep"})
		if err != nil {
			t.Fatalf("Create deployment error = %v", err)
		}
		if err := store.Apps().AddCollaborator(ctx, owner.ID, app.ID, collaborator.Email); err != nil {
			t.Fatalf("AddCollaborator() error = %v", err)
		}

		if _, err := store.Deployments().List(ctx, collaborator.ID, app.ID); err != nil {
			t.Fatalf("expected collaborator to list deployments, got %v", err)
		}
		if _, err := store.Deployments().Create(ctx, collaborator.ID, app.ID, domain.Deployment{Name: "Staging", Key: "pkg-stage"}); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected collaborator create to be forbidden, got %v", err)
		}

		first, err := store.Packages().CommitRollback(ctx, owner.ID, app.ID, deployment.ID, domain.Package{
			AppVersion:         "1.0.0",
			Description:        "first",
			PackageHash:        "hash-1",
			BlobURL:            "http://example.com/v1.zip",
			UploadTime:         time.Now().UnixMilli(),
			ReleaseMethod:      domain.ReleaseRollback,
			OriginalLabel:      "base",
			OriginalDeployment: deployment.Name,
		})
		if err != nil {
			t.Fatalf("CommitRollback() error = %v", err)
		}
		second, err := store.Packages().CommitRollback(ctx, owner.ID, app.ID, deployment.ID, domain.Package{
			AppVersion:         "1.0.0",
			Description:        "second",
			PackageHash:        "hash-2",
			BlobURL:            "http://example.com/v2.zip",
			UploadTime:         time.Now().UnixMilli(),
			ReleaseMethod:      domain.ReleaseRollback,
			OriginalLabel:      first.Label,
			OriginalDeployment: deployment.Name,
		})
		if err != nil {
			t.Fatalf("CommitRollback() error = %v", err)
		}
		if first.Label != "v1" || second.Label != "v2" {
			t.Fatalf("expected monotonic labels, got %q and %q", first.Label, second.Label)
		}

		history, err := store.Packages().ListHistory(ctx, collaborator.ID, app.ID, deployment.ID)
		if err != nil {
			t.Fatalf("ListHistory() error = %v", err)
		}
		if len(history) != 2 || history[0].Label != "v1" || history[1].Label != "v2" {
			t.Fatalf("unexpected package history %#v", history)
		}

		if err := store.Packages().ClearHistory(ctx, collaborator.ID, app.ID, deployment.ID); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected collaborator clear to be forbidden, got %v", err)
		}
		if err := store.Packages().ClearHistory(ctx, owner.ID, app.ID, deployment.ID); err != nil {
			t.Fatalf("owner clear history error = %v", err)
		}
		history, err = store.Packages().ListHistory(ctx, owner.ID, app.ID, deployment.ID)
		if err != nil {
			t.Fatalf("ListHistory() error = %v", err)
		}
		if len(history) != 0 {
			t.Fatalf("expected cleared history, got %d items", len(history))
		}
	})
}

func seedBootstrapAccount(t *testing.T, ctx context.Context, store *postgres.Store, email, token string) domain.Account {
	t.Helper()
	if err := store.Accounts().EnsureBootstrap(ctx, domain.Account{
		Email: email,
		Name:  email,
	}, domain.AccessKey{
		Name:         token,
		FriendlyName: token,
		Description:  "seed",
		CreatedBy:    "test",
		CreatedTime:  time.Now().UnixMilli(),
		Expires:      time.Now().Add(time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatalf("EnsureBootstrap(%q) error = %v", email, err)
	}
	account, err := store.Accounts().GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetByEmail(%q) error = %v", email, err)
	}
	return account
}
