package application

import (
	"context"
	"testing"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

type fakeAccounts struct {
	account domain.Account
}

func (f *fakeAccounts) CheckHealth(context.Context) error { return nil }
func (f *fakeAccounts) EnsureBootstrap(context.Context, domain.Account, domain.AccessKey) error {
	return nil
}
func (f *fakeAccounts) GetByID(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *fakeAccounts) GetByEmail(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *fakeAccounts) ResolveAccountIDByAccessKey(context.Context, string) (string, error) {
	return f.account.ID, nil
}

type fakeAccessKeys struct{}

func (f *fakeAccessKeys) List(context.Context, string) ([]domain.AccessKey, error) { return nil, nil }
func (f *fakeAccessKeys) Create(context.Context, domain.AccessKey, string) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (f *fakeAccessKeys) GetByName(context.Context, string, string) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (f *fakeAccessKeys) Update(context.Context, domain.AccessKey) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (f *fakeAccessKeys) Delete(context.Context, string, string) error { return nil }
func (f *fakeAccessKeys) DeleteSessionsByCreator(context.Context, string, string) (int64, error) {
	return 0, nil
}

type fakeApps struct {
	app domain.App
}

func (f *fakeApps) List(context.Context, string) ([]domain.App, error) {
	return []domain.App{f.app}, nil
}
func (f *fakeApps) Create(context.Context, string, domain.App) (domain.App, error) { return f.app, nil }
func (f *fakeApps) GetByName(context.Context, string, string) (domain.App, error)  { return f.app, nil }
func (f *fakeApps) Update(context.Context, string, domain.App) (domain.App, error) { return f.app, nil }
func (f *fakeApps) Delete(context.Context, string, string) error                   { return nil }
func (f *fakeApps) Transfer(context.Context, string, string, string) error         { return nil }
func (f *fakeApps) AddCollaborator(context.Context, string, string, string) error  { return nil }
func (f *fakeApps) ListCollaborators(context.Context, string, string) (map[string]domain.CollaboratorProperties, error) {
	return nil, nil
}
func (f *fakeApps) RemoveCollaborator(context.Context, string, string, string) error { return nil }

type fakeDeployments struct {
	deployment domain.Deployment
}

func (f *fakeDeployments) List(context.Context, string, string) ([]domain.Deployment, error) {
	return []domain.Deployment{f.deployment}, nil
}
func (f *fakeDeployments) Create(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return f.deployment, nil
}
func (f *fakeDeployments) GetByName(context.Context, string, string, string) (domain.Deployment, error) {
	return f.deployment, nil
}
func (f *fakeDeployments) GetByKey(context.Context, string) (domain.Deployment, error) {
	return f.deployment, nil
}
func (f *fakeDeployments) Update(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return f.deployment, nil
}
func (f *fakeDeployments) Delete(context.Context, string, string, string) error { return nil }

type fakePackages struct {
	history []domain.Package
}

func (f *fakePackages) ListHistory(context.Context, string, string, string) ([]domain.Package, error) {
	return f.history, nil
}
func (f *fakePackages) ClearHistory(context.Context, string, string, string) error { return nil }
func (f *fakePackages) CommitRollback(context.Context, string, string, string, domain.Package) (domain.Package, error) {
	return domain.Package{}, nil
}

type fakeMetrics struct{}

func (f *fakeMetrics) CheckHealth(context.Context) error                                 { return nil }
func (f *fakeMetrics) IncrementDownload(context.Context, string, string) error           { return nil }
func (f *fakeMetrics) ReportDeploy(context.Context, domain.DeploymentStatusReport) error { return nil }
func (f *fakeMetrics) GetMetrics(context.Context, string) (map[string]domain.UpdateMetrics, error) {
	return nil, nil
}
func (f *fakeMetrics) Clear(context.Context, string) error { return nil }

func TestUpdateCheckSelectsLatestMatchingPackage(t *testing.T) {
	service := NewService(
		&fakeAccounts{account: domain.Account{ID: "a1"}},
		&fakeAccessKeys{},
		&fakeApps{app: domain.App{ID: "app1", Name: "demo"}},
		&fakeDeployments{deployment: domain.Deployment{ID: "dep1", AppID: "app1", Name: "Production", Key: "key"}},
		&fakePackages{history: []domain.Package{
			{Label: "v1", AppVersion: "1.0.0", PackageHash: "old", BlobURL: "http://example/old.zip", Size: 1},
			{Label: "v2", AppVersion: "1.0.0", PackageHash: "new", BlobURL: "http://example/new.zip", Size: 2},
		}},
		&fakeMetrics{},
	)

	resp, err := service.UpdateCheck(context.Background(), domain.UpdateCheckRequest{
		DeploymentKey: "key",
		AppVersion:    "1.0.0",
	})
	if err != nil {
		t.Fatalf("UpdateCheck() error = %v", err)
	}
	if !resp.IsAvailable {
		t.Fatalf("expected update to be available")
	}
	if resp.PackageHash != "new" {
		t.Fatalf("expected latest package hash, got %q", resp.PackageHash)
	}
}

func TestSelectedForRolloutIsDeterministic(t *testing.T) {
	first := selectedForRollout("client-1", 20, "v5")
	second := selectedForRollout("client-1", 20, "v5")
	if first != second {
		t.Fatalf("expected deterministic rollout selection")
	}
}
