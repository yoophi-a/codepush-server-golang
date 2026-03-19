package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

type fakeAccounts struct {
	account             domain.Account
	getByIDAccount      domain.Account
	getByIDErr          error
	resolveAccountID    string
	resolveErr          error
	healthErr           error
	resolveAccessKeyArg string
}

func (f *fakeAccounts) CheckHealth(context.Context) error { return f.healthErr }
func (f *fakeAccounts) EnsureBootstrap(context.Context, domain.Account, domain.AccessKey) error {
	return nil
}
func (f *fakeAccounts) GetByID(context.Context, string) (domain.Account, error) {
	if f.getByIDErr != nil {
		return domain.Account{}, f.getByIDErr
	}
	if f.getByIDAccount.ID != "" {
		return f.getByIDAccount, nil
	}
	return f.account, nil
}
func (f *fakeAccounts) GetByEmail(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *fakeAccounts) ResolveAccountIDByAccessKey(_ context.Context, token string) (string, error) {
	f.resolveAccessKeyArg = token
	if f.resolveErr != nil {
		return "", f.resolveErr
	}
	if f.resolveAccountID != "" {
		return f.resolveAccountID, nil
	}
	return f.account.ID, nil
}

type fakeAccessKeys struct {
	listResult         []domain.AccessKey
	listErr            error
	createResult       domain.AccessKey
	createErr          error
	createInput        domain.AccessKey
	createToken        string
	getByNameResult    domain.AccessKey
	getByNameErr       error
	updateResult       domain.AccessKey
	updateErr          error
	updateInput        domain.AccessKey
	deleteErr          error
	deleteSessionsRows int64
	deleteSessionsErr  error
}

func (f *fakeAccessKeys) List(context.Context, string) ([]domain.AccessKey, error) {
	return f.listResult, f.listErr
}
func (f *fakeAccessKeys) Create(_ context.Context, key domain.AccessKey, token string) (domain.AccessKey, error) {
	f.createInput = key
	f.createToken = token
	if f.createErr != nil {
		return domain.AccessKey{}, f.createErr
	}
	if f.createResult.ID != "" || f.createResult.Name != "" || f.createResult.FriendlyName != "" {
		return f.createResult, nil
	}
	key.ID = "created-key"
	key.Name = token
	return key, nil
}
func (f *fakeAccessKeys) GetByName(context.Context, string, string) (domain.AccessKey, error) {
	if f.getByNameErr != nil {
		return domain.AccessKey{}, f.getByNameErr
	}
	return f.getByNameResult, nil
}
func (f *fakeAccessKeys) Update(_ context.Context, key domain.AccessKey) (domain.AccessKey, error) {
	f.updateInput = key
	if f.updateErr != nil {
		return domain.AccessKey{}, f.updateErr
	}
	if f.updateResult.ID != "" || f.updateResult.FriendlyName != "" {
		return f.updateResult, nil
	}
	return key, nil
}
func (f *fakeAccessKeys) Delete(context.Context, string, string) error { return f.deleteErr }
func (f *fakeAccessKeys) DeleteSessionsByCreator(context.Context, string, string) (int64, error) {
	return f.deleteSessionsRows, f.deleteSessionsErr
}

type fakeApps struct {
	listResult              []domain.App
	listErr                 error
	createResult            domain.App
	createErr               error
	createInput             domain.App
	getByNameResult         domain.App
	getByNameErr            error
	updateResult            domain.App
	updateErr               error
	updateInput             domain.App
	deleteErr               error
	transferErr             error
	addCollaboratorErr      error
	listCollaboratorsResult map[string]domain.CollaboratorProperties
	listCollaboratorsErr    error
	removeCollaboratorErr   error
}

func (f *fakeApps) List(context.Context, string) ([]domain.App, error) {
	return f.listResult, f.listErr
}
func (f *fakeApps) Create(_ context.Context, _ string, app domain.App) (domain.App, error) {
	f.createInput = app
	return f.createResult, f.createErr
}
func (f *fakeApps) GetByName(context.Context, string, string) (domain.App, error) {
	if f.getByNameErr != nil {
		return domain.App{}, f.getByNameErr
	}
	return f.getByNameResult, nil
}
func (f *fakeApps) Update(context.Context, string, domain.App) (domain.App, error) {
	return f.updateResult, f.updateErr
}
func (f *fakeApps) Delete(context.Context, string, string) error           { return f.deleteErr }
func (f *fakeApps) Transfer(context.Context, string, string, string) error { return f.transferErr }
func (f *fakeApps) AddCollaborator(context.Context, string, string, string) error {
	return f.addCollaboratorErr
}
func (f *fakeApps) ListCollaborators(context.Context, string, string) (map[string]domain.CollaboratorProperties, error) {
	return f.listCollaboratorsResult, f.listCollaboratorsErr
}
func (f *fakeApps) RemoveCollaborator(context.Context, string, string, string) error {
	return f.removeCollaboratorErr
}

type deploymentCreateCall struct {
	accountID string
	appID     string
	dep       domain.Deployment
}

type fakeDeployments struct {
	listResult      []domain.Deployment
	listErr         error
	createResult    domain.Deployment
	createErr       error
	createErrs      []error
	createCalls     []deploymentCreateCall
	getByNameResult domain.Deployment
	getByNameErr    error
	getByKeyResult  domain.Deployment
	getByKeyErr     error
	updateResult    domain.Deployment
	updateErr       error
	deleteErr       error
}

func (f *fakeDeployments) List(context.Context, string, string) ([]domain.Deployment, error) {
	return f.listResult, f.listErr
}
func (f *fakeDeployments) Create(_ context.Context, accountID, appID string, dep domain.Deployment) (domain.Deployment, error) {
	f.createCalls = append(f.createCalls, deploymentCreateCall{accountID: accountID, appID: appID, dep: dep})
	if len(f.createErrs) > 0 {
		err := f.createErrs[0]
		f.createErrs = f.createErrs[1:]
		if err != nil {
			return domain.Deployment{}, err
		}
	}
	if f.createErr != nil {
		return domain.Deployment{}, f.createErr
	}
	if f.createResult.ID != "" || f.createResult.Name != "" {
		return f.createResult, nil
	}
	dep.ID = dep.Name
	dep.AppID = appID
	return dep, nil
}
func (f *fakeDeployments) GetByName(context.Context, string, string, string) (domain.Deployment, error) {
	if f.getByNameErr != nil {
		return domain.Deployment{}, f.getByNameErr
	}
	return f.getByNameResult, nil
}
func (f *fakeDeployments) GetByKey(context.Context, string) (domain.Deployment, error) {
	if f.getByKeyErr != nil {
		return domain.Deployment{}, f.getByKeyErr
	}
	return f.getByKeyResult, nil
}
func (f *fakeDeployments) Update(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return f.updateResult, f.updateErr
}
func (f *fakeDeployments) Delete(context.Context, string, string, string) error { return f.deleteErr }

type fakePackages struct {
	history       []domain.Package
	listErr       error
	clearErr      error
	clearCalled   bool
	commitResult  domain.Package
	commitErr     error
	commitCalled  bool
	commitAccount string
	commitAppID   string
	commitDepID   string
	commitInput   domain.Package
}

func (f *fakePackages) ListHistory(context.Context, string, string, string) ([]domain.Package, error) {
	return f.history, f.listErr
}
func (f *fakePackages) ClearHistory(context.Context, string, string, string) error {
	f.clearCalled = true
	return f.clearErr
}
func (f *fakePackages) CommitRollback(_ context.Context, accountID, appID, depID string, pkg domain.Package) (domain.Package, error) {
	f.commitCalled = true
	f.commitAccount = accountID
	f.commitAppID = appID
	f.commitDepID = depID
	f.commitInput = pkg
	if f.commitErr != nil {
		return domain.Package{}, f.commitErr
	}
	if f.commitResult.ID != "" || f.commitResult.Label != "" {
		return f.commitResult, nil
	}
	return pkg, nil
}

type fakeMetrics struct {
	healthErr               error
	clearErr                error
	clearCalled             bool
	clearDeploymentKey      string
	incrementDownloadErr    error
	incrementDownloadKey    string
	incrementDownloadLabel  string
	reportDeployErr         error
	reportDeployCalled      bool
	reportDeployInput       domain.DeploymentStatusReport
	getMetricsErr           error
	getMetricsResult        map[string]domain.UpdateMetrics
	getMetricsDeploymentKey string
	checkHealthCalled       bool
}

func (f *fakeMetrics) CheckHealth(context.Context) error {
	f.checkHealthCalled = true
	return f.healthErr
}
func (f *fakeMetrics) IncrementDownload(_ context.Context, key, label string) error {
	f.incrementDownloadKey = key
	f.incrementDownloadLabel = label
	return f.incrementDownloadErr
}
func (f *fakeMetrics) ReportDeploy(_ context.Context, report domain.DeploymentStatusReport) error {
	f.reportDeployCalled = true
	f.reportDeployInput = report
	return f.reportDeployErr
}
func (f *fakeMetrics) GetMetrics(_ context.Context, key string) (map[string]domain.UpdateMetrics, error) {
	f.getMetricsDeploymentKey = key
	return f.getMetricsResult, f.getMetricsErr
}
func (f *fakeMetrics) Clear(_ context.Context, key string) error {
	f.clearCalled = true
	f.clearDeploymentKey = key
	return f.clearErr
}

func TestListAccessKeysMasksAndSorts(t *testing.T) {
	keys := &fakeAccessKeys{
		listResult: []domain.AccessKey{
			{Name: "late", CreatedTime: 20},
			{Name: "early", CreatedTime: 10},
		},
	}
	service := NewService(&fakeAccounts{}, keys, &fakeApps{}, &fakeDeployments{}, &fakePackages{}, &fakeMetrics{})

	got, err := service.ListAccessKeys(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("ListAccessKeys() error = %v", err)
	}
	if got[0].CreatedTime != 10 || got[1].CreatedTime != 20 {
		t.Fatalf("expected ascending sort, got %#v", got)
	}
	for _, key := range got {
		if key.Name != domain.AccessKeyMask {
			t.Fatalf("expected masked access key name, got %q", key.Name)
		}
	}
}

func TestCreateAccessKeyAppliesDefaults(t *testing.T) {
	keys := &fakeAccessKeys{}
	service := NewService(&fakeAccounts{}, keys, &fakeApps{}, &fakeDeployments{}, &fakePackages{}, &fakeMetrics{})

	got, err := service.CreateAccessKey(context.Background(), "acc-1", "", domain.AccessKeyRequest{})
	if err != nil {
		t.Fatalf("CreateAccessKey() error = %v", err)
	}
	if keys.createInput.CreatedBy != "unknown" {
		t.Fatalf("expected default createdBy, got %q", keys.createInput.CreatedBy)
	}
	if keys.createInput.FriendlyName != keys.createInput.Name {
		t.Fatalf("expected friendly name to default to token, got %q vs %q", keys.createInput.FriendlyName, keys.createInput.Name)
	}
	if keys.createInput.Expires-keys.createInput.CreatedTime != defaultAccessKeyTTL {
		t.Fatalf("expected default TTL %d, got %d", defaultAccessKeyTTL, keys.createInput.Expires-keys.createInput.CreatedTime)
	}
	if len(keys.createToken) != 40 {
		t.Fatalf("expected generated sha1 token, got %q", keys.createToken)
	}
	if got.Name != keys.createToken {
		t.Fatalf("expected returned token to match generated token, got %q", got.Name)
	}
}

func TestCreateAccessKeyUsesConfiguredDefaultTTL(t *testing.T) {
	keys := &fakeAccessKeys{}
	service := NewService(&fakeAccounts{}, keys, &fakeApps{}, &fakeDeployments{}, &fakePackages{}, &fakeMetrics{}, WithDefaultAccessKeyTTL(12345))

	if _, err := service.CreateAccessKey(context.Background(), "acc-1", "", domain.AccessKeyRequest{}); err != nil {
		t.Fatalf("CreateAccessKey() error = %v", err)
	}
	if keys.createInput.Expires-keys.createInput.CreatedTime != 12345 {
		t.Fatalf("expected configured default TTL, got %d", keys.createInput.Expires-keys.createInput.CreatedTime)
	}
}

func TestGetAccessKeyHidesName(t *testing.T) {
	keys := &fakeAccessKeys{
		getByNameResult: domain.AccessKey{Name: "secret", FriendlyName: "visible"},
	}
	service := NewService(&fakeAccounts{}, keys, &fakeApps{}, &fakeDeployments{}, &fakePackages{}, &fakeMetrics{})

	got, err := service.GetAccessKey(context.Background(), "acc-1", "secret")
	if err != nil {
		t.Fatalf("GetAccessKey() error = %v", err)
	}
	if got.Name != "" {
		t.Fatalf("expected secret to be hidden, got %q", got.Name)
	}
}

func TestUpdateAccessKeyUpdatesFieldsAndHidesName(t *testing.T) {
	keys := &fakeAccessKeys{
		getByNameResult: domain.AccessKey{
			ID:           "k1",
			Name:         "secret",
			FriendlyName: "old",
			Description:  "old-desc",
			Expires:      100,
		},
	}
	service := NewService(&fakeAccounts{}, keys, &fakeApps{}, &fakeDeployments{}, &fakePackages{}, &fakeMetrics{})

	got, err := service.UpdateAccessKey(context.Background(), "acc-1", "secret", domain.AccessKeyRequest{
		FriendlyName: "new-name",
		TTL:          1000,
	})
	if err != nil {
		t.Fatalf("UpdateAccessKey() error = %v", err)
	}
	if keys.updateInput.FriendlyName != "new-name" {
		t.Fatalf("expected updated friendly name, got %q", keys.updateInput.FriendlyName)
	}
	if keys.updateInput.Description != "new-name" {
		t.Fatalf("expected description to follow current implementation, got %q", keys.updateInput.Description)
	}
	if keys.updateInput.Expires <= 100 {
		t.Fatalf("expected expiry to be extended, got %d", keys.updateInput.Expires)
	}
	if got.Name != "" {
		t.Fatalf("expected returned key name hidden, got %q", got.Name)
	}
}

func TestDeleteSessionsByCreatorReturnsNotFoundOnZeroRows(t *testing.T) {
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, &fakeApps{}, &fakeDeployments{}, &fakePackages{}, &fakeMetrics{})

	err := service.DeleteSessionsByCreator(context.Background(), "acc-1", "creator")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateAppAutoCreatesDefaultDeployments(t *testing.T) {
	apps := &fakeApps{
		createResult:    domain.App{ID: "app-1", Name: "demo"},
		getByNameResult: domain.App{ID: "app-1", Name: "demo"},
	}
	deps := &fakeDeployments{}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, deps, &fakePackages{}, &fakeMetrics{})

	got, err := service.CreateApp(context.Background(), "acc-1", domain.AppCreationRequest{Name: "demo"})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	if got.Name != "demo" {
		t.Fatalf("expected created app, got %#v", got)
	}
	if len(deps.createCalls) != 2 {
		t.Fatalf("expected 2 default deployments, got %d", len(deps.createCalls))
	}
	if deps.createCalls[0].dep.Name != "Production" || deps.createCalls[1].dep.Name != "Staging" {
		t.Fatalf("unexpected deployment names: %#v", deps.createCalls)
	}
	for _, call := range deps.createCalls {
		if call.dep.Key == "" {
			t.Fatalf("expected generated deployment key")
		}
	}
}

func TestCreateAppRetriesGeneratedDeploymentKeyConflict(t *testing.T) {
	prevTokenGenerator := tokenGenerator
	defer func() { tokenGenerator = prevTokenGenerator }()

	generated := []string{"dup-key", "ok-key-1", "ok-key-2"}
	tokenGenerator = func(string, int64) string {
		token := generated[0]
		generated = generated[1:]
		return token
	}

	apps := &fakeApps{
		createResult:    domain.App{ID: "app-1", Name: "demo"},
		getByNameResult: domain.App{ID: "app-1", Name: "demo"},
	}
	deps := &fakeDeployments{createErrs: []error{domain.ErrConflict, nil, nil}}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, deps, &fakePackages{}, &fakeMetrics{})

	if _, err := service.CreateApp(context.Background(), "acc-1", domain.AppCreationRequest{Name: "demo"}); err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	if len(deps.createCalls) != 3 {
		t.Fatalf("expected retry plus second deployment create, got %d calls", len(deps.createCalls))
	}
	if deps.createCalls[0].dep.Key != "dup-key" || deps.createCalls[1].dep.Key != "ok-key-1" || deps.createCalls[2].dep.Key != "ok-key-2" {
		t.Fatalf("unexpected generated keys: %#v", deps.createCalls)
	}
}

func TestCreateAppManualProvisionSkipsDefaultDeployments(t *testing.T) {
	apps := &fakeApps{
		createResult:    domain.App{ID: "app-1", Name: "demo"},
		getByNameResult: domain.App{ID: "app-1", Name: "demo"},
	}
	deps := &fakeDeployments{}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, deps, &fakePackages{}, &fakeMetrics{})

	_, err := service.CreateApp(context.Background(), "acc-1", domain.AppCreationRequest{
		Name:                         "demo",
		ManuallyProvisionDeployments: true,
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	if len(deps.createCalls) != 0 {
		t.Fatalf("expected manual provisioning to skip default deployments")
	}
}

func TestListAppsSortsByName(t *testing.T) {
	apps := &fakeApps{
		listResult: []domain.App{{Name: "zeta"}, {Name: "alpha"}},
	}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, &fakeDeployments{}, &fakePackages{}, &fakeMetrics{})

	got, err := service.ListApps(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("ListApps() error = %v", err)
	}
	if got[0].Name != "alpha" || got[1].Name != "zeta" {
		t.Fatalf("expected sorted apps, got %#v", got)
	}
}

func TestListDeploymentsSortsByName(t *testing.T) {
	apps := &fakeApps{getByNameResult: domain.App{ID: "app-1", Name: "demo"}}
	deps := &fakeDeployments{
		listResult: []domain.Deployment{{Name: "Staging"}, {Name: "Production"}},
	}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, deps, &fakePackages{}, &fakeMetrics{})

	got, err := service.ListDeployments(context.Background(), "acc-1", "demo")
	if err != nil {
		t.Fatalf("ListDeployments() error = %v", err)
	}
	if got[0].Name != "Production" || got[1].Name != "Staging" {
		t.Fatalf("expected sorted deployments, got %#v", got)
	}
}

func TestCreateDeploymentGeneratesKeyWhenMissing(t *testing.T) {
	apps := &fakeApps{getByNameResult: domain.App{ID: "app-1", Name: "demo"}}
	deps := &fakeDeployments{}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, deps, &fakePackages{}, &fakeMetrics{})

	_, err := service.CreateDeployment(context.Background(), "acc-1", "demo", domain.DeploymentRequest{Name: "Production"})
	if err != nil {
		t.Fatalf("CreateDeployment() error = %v", err)
	}
	if len(deps.createCalls) != 1 {
		t.Fatalf("expected one deployment create call")
	}
	if deps.createCalls[0].dep.Key == "" {
		t.Fatalf("expected generated key")
	}
}

func TestCreateDeploymentRetriesGeneratedKeyConflict(t *testing.T) {
	prevTokenGenerator := tokenGenerator
	defer func() { tokenGenerator = prevTokenGenerator }()

	generated := []string{"dup-key", "ok-key"}
	tokenGenerator = func(string, int64) string {
		token := generated[0]
		generated = generated[1:]
		return token
	}

	apps := &fakeApps{getByNameResult: domain.App{ID: "app-1", Name: "demo"}}
	deps := &fakeDeployments{createErrs: []error{domain.ErrConflict, nil}}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, deps, &fakePackages{}, &fakeMetrics{})

	deployment, err := service.CreateDeployment(context.Background(), "acc-1", "demo", domain.DeploymentRequest{Name: "Production"})
	if err != nil {
		t.Fatalf("CreateDeployment() error = %v", err)
	}
	if deployment.Key != "ok-key" {
		t.Fatalf("expected retry to succeed with regenerated key, got %#v", deployment)
	}
	if len(deps.createCalls) != 2 {
		t.Fatalf("expected retry on conflict, got %d calls", len(deps.createCalls))
	}
}

func TestClearHistoryClearsMetricsAfterPackageHistory(t *testing.T) {
	apps := &fakeApps{getByNameResult: domain.App{ID: "app-1", Name: "demo"}}
	deps := &fakeDeployments{getByNameResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production", Key: "dep-key"}}
	pkgs := &fakePackages{}
	metrics := &fakeMetrics{}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, deps, pkgs, metrics)

	if err := service.ClearHistory(context.Background(), "acc-1", "demo", "Production"); err != nil {
		t.Fatalf("ClearHistory() error = %v", err)
	}
	if !pkgs.clearCalled {
		t.Fatalf("expected package history to be cleared")
	}
	if !metrics.clearCalled || metrics.clearDeploymentKey != "dep-key" {
		t.Fatalf("expected metrics clear after package clear, got called=%v key=%q", metrics.clearCalled, metrics.clearDeploymentKey)
	}
}

func TestClearHistoryDoesNotClearMetricsOnPackageError(t *testing.T) {
	apps := &fakeApps{getByNameResult: domain.App{ID: "app-1", Name: "demo"}}
	deps := &fakeDeployments{getByNameResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production", Key: "dep-key"}}
	pkgs := &fakePackages{clearErr: errors.New("boom")}
	metrics := &fakeMetrics{}
	service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, apps, deps, pkgs, metrics)

	err := service.ClearHistory(context.Background(), "acc-1", "demo", "Production")
	if err == nil {
		t.Fatalf("expected error")
	}
	if metrics.clearCalled {
		t.Fatalf("expected metrics clear to be skipped after package error")
	}
}

func TestRollbackScenarios(t *testing.T) {
	baseApp := &fakeApps{getByNameResult: domain.App{ID: "app-1", Name: "demo"}}
	baseDep := &fakeDeployments{getByNameResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production", Key: "dep-key"}}

	t.Run("not found when history empty", func(t *testing.T) {
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, baseApp, baseDep, &fakePackages{}, &fakeMetrics{})
		_, err := service.Rollback(context.Background(), "acc-1", "demo", "Production", "")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("not found when only latest exists and no target label", func(t *testing.T) {
		pkgs := &fakePackages{history: []domain.Package{{Label: "v1", AppVersion: "1.0.0"}}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, baseApp, baseDep, pkgs, &fakeMetrics{})
		_, err := service.Rollback(context.Background(), "acc-1", "demo", "Production", "")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("not found when target label missing", func(t *testing.T) {
		pkgs := &fakePackages{history: []domain.Package{
			{Label: "v1", AppVersion: "1.0.0"},
			{Label: "v2", AppVersion: "1.0.0"},
		}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, baseApp, baseDep, pkgs, &fakeMetrics{})
		_, err := service.Rollback(context.Background(), "acc-1", "demo", "Production", "v9")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("conflict when app version differs", func(t *testing.T) {
		pkgs := &fakePackages{history: []domain.Package{
			{Label: "v1", AppVersion: "1.0.0"},
			{Label: "v2", AppVersion: "2.0.0"},
		}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, baseApp, baseDep, pkgs, &fakeMetrics{})
		_, err := service.Rollback(context.Background(), "acc-1", "demo", "Production", "v1")
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("success clones target package as rollback", func(t *testing.T) {
		rollout := 50
		pkgs := &fakePackages{history: []domain.Package{
			{Label: "v1", AppVersion: "1.0.0", Description: "stable", PackageHash: "hash-1", Rollout: &rollout},
			{Label: "v2", AppVersion: "1.0.0", Description: "bad", PackageHash: "hash-2"},
		}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, baseApp, baseDep, pkgs, &fakeMetrics{})

		got, err := service.Rollback(context.Background(), "acc-1", "demo", "Production", "v1")
		if err != nil {
			t.Fatalf("Rollback() error = %v", err)
		}
		if !pkgs.commitCalled {
			t.Fatalf("expected rollback commit")
		}
		if pkgs.commitInput.ReleaseMethod != domain.ReleaseRollback {
			t.Fatalf("expected rollback release method, got %q", pkgs.commitInput.ReleaseMethod)
		}
		if pkgs.commitInput.OriginalDeployment != "Production" || pkgs.commitInput.OriginalLabel != "v1" {
			t.Fatalf("expected original package metadata, got %#v", pkgs.commitInput)
		}
		if pkgs.commitInput.ID != "" {
			t.Fatalf("expected cloned package ID to be cleared")
		}
		if pkgs.commitInput.UploadTime <= 0 {
			t.Fatalf("expected upload time to be refreshed")
		}
		if got.PackageHash != "hash-1" {
			t.Fatalf("expected committed rollback package to mirror target, got %#v", got)
		}
	})
}

func TestUpdateCheckBehaviors(t *testing.T) {
	base := NewService(
		&fakeAccounts{},
		&fakeAccessKeys{},
		&fakeApps{},
		&fakeDeployments{getByKeyResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Key: "dep-key"}},
		&fakePackages{},
		&fakeMetrics{},
	)

	t.Run("selects latest matching package", func(t *testing.T) {
		pkgs := &fakePackages{history: []domain.Package{
			{Label: "v1", AppVersion: "1.0.0", PackageHash: "old", BlobURL: "http://example/old.zip", Size: 1},
			{Label: "v2", AppVersion: "1.0.0", PackageHash: "new", BlobURL: "http://example/new.zip", Size: 2},
		}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, &fakeApps{}, &fakeDeployments{getByKeyResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Key: "dep-key"}}, pkgs, &fakeMetrics{})
		resp, err := service.UpdateCheck(context.Background(), domain.UpdateCheckRequest{DeploymentKey: "dep-key", AppVersion: "1.0.0"})
		if err != nil {
			t.Fatalf("UpdateCheck() error = %v", err)
		}
		if !resp.IsAvailable || resp.PackageHash != "new" {
			t.Fatalf("expected latest matching package, got %#v", resp)
		}
	})

	t.Run("skips disabled package and respects semver", func(t *testing.T) {
		pkgs := &fakePackages{history: []domain.Package{
			{Label: "v1", AppVersion: ">=1.0.0 <2.0.0", PackageHash: "hash-1", IsDisabled: true},
			{Label: "v2", AppVersion: "~1.2.0", PackageHash: "hash-2"},
		}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, &fakeApps{}, &fakeDeployments{getByKeyResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Key: "dep-key"}}, pkgs, &fakeMetrics{})
		resp, err := service.UpdateCheck(context.Background(), domain.UpdateCheckRequest{DeploymentKey: "dep-key", AppVersion: "1.2.3"})
		if err != nil {
			t.Fatalf("UpdateCheck() error = %v", err)
		}
		if resp.PackageHash != "hash-2" {
			t.Fatalf("expected semver match after skipping disabled package, got %#v", resp)
		}
	})

	t.Run("returns unavailable when package hash already installed", func(t *testing.T) {
		pkgs := &fakePackages{history: []domain.Package{{Label: "v1", AppVersion: "1.0.0", PackageHash: "same"}}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, &fakeApps{}, &fakeDeployments{getByKeyResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Key: "dep-key"}}, pkgs, &fakeMetrics{})
		resp, err := service.UpdateCheck(context.Background(), domain.UpdateCheckRequest{
			DeploymentKey: "dep-key",
			AppVersion:    "1.0.0",
			PackageHash:   "same",
		})
		if err != nil {
			t.Fatalf("UpdateCheck() error = %v", err)
		}
		if resp.IsAvailable {
			t.Fatalf("expected no update when package hash matches, got %#v", resp)
		}
	})

	t.Run("rollout requires client id", func(t *testing.T) {
		rollout := 20
		pkgs := &fakePackages{history: []domain.Package{{Label: "v1", AppVersion: "1.0.0", PackageHash: "hash-1", Rollout: &rollout}}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, &fakeApps{}, &fakeDeployments{getByKeyResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Key: "dep-key"}}, pkgs, &fakeMetrics{})
		resp, err := service.UpdateCheck(context.Background(), domain.UpdateCheckRequest{DeploymentKey: "dep-key", AppVersion: "1.0.0"})
		if err != nil {
			t.Fatalf("UpdateCheck() error = %v", err)
		}
		if resp.IsAvailable {
			t.Fatalf("expected rollout package to be skipped without client ID, got %#v", resp)
		}
	})

	t.Run("rollout can select deterministic client", func(t *testing.T) {
		rollout := 100
		pkgs := &fakePackages{history: []domain.Package{{Label: "v1", AppVersion: "1.0.0", PackageHash: "hash-1", Rollout: &rollout}}}
		service := NewService(&fakeAccounts{}, &fakeAccessKeys{}, &fakeApps{}, &fakeDeployments{getByKeyResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Key: "dep-key"}}, pkgs, &fakeMetrics{})
		resp, err := service.UpdateCheck(context.Background(), domain.UpdateCheckRequest{
			DeploymentKey:  "dep-key",
			AppVersion:     "1.0.0",
			ClientUniqueID: "client-1",
		})
		if err != nil {
			t.Fatalf("UpdateCheck() error = %v", err)
		}
		if !resp.IsAvailable {
			t.Fatalf("expected rollout package to be available, got %#v", resp)
		}
	})

	_ = base
}

func TestDelegatingMethods(t *testing.T) {
	accounts := &fakeAccounts{
		account:          domain.Account{ID: "acc-1"},
		getByIDAccount:   domain.Account{ID: "acc-1", Email: "user@example.com"},
		resolveAccountID: "acc-1",
	}
	metrics := &fakeMetrics{getMetricsResult: map[string]domain.UpdateMetrics{"v1": {Installed: 1}}}
	deps := &fakeDeployments{getByNameResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production", Key: "dep-key"}}
	apps := &fakeApps{getByNameResult: domain.App{ID: "app-1", Name: "demo"}}
	service := NewService(accounts, &fakeAccessKeys{}, apps, deps, &fakePackages{}, metrics)

	if err := service.Health(context.Background()); err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if !metrics.checkHealthCalled {
		t.Fatalf("expected metrics health check")
	}

	account, err := service.Authenticate(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if account.Email != "user@example.com" {
		t.Fatalf("unexpected account: %#v", account)
	}
	if accounts.resolveAccessKeyArg != "token-1" {
		t.Fatalf("expected access key token to be forwarded, got %q", accounts.resolveAccessKeyArg)
	}

	report := domain.DeploymentStatusReport{DeploymentKey: "dep-key", Label: "v1"}
	if err := service.ReportDeploy(context.Background(), report); err != nil {
		t.Fatalf("ReportDeploy() error = %v", err)
	}
	if !metrics.reportDeployCalled || metrics.reportDeployInput.Label != "v1" {
		t.Fatalf("expected deploy report to reach metrics backend")
	}

	if err := service.ReportDownload(context.Background(), domain.DownloadReport{DeploymentKey: "dep-key", Label: "v1"}); err != nil {
		t.Fatalf("ReportDownload() error = %v", err)
	}
	if metrics.incrementDownloadKey != "dep-key" || metrics.incrementDownloadLabel != "v1" {
		t.Fatalf("expected download report to reach metrics backend")
	}

	gotMetrics, err := service.GetMetrics(context.Background(), "acc-1", "demo", "Production")
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}
	if gotMetrics["v1"].Installed != 1 {
		t.Fatalf("unexpected metrics result: %#v", gotMetrics)
	}
	if metrics.getMetricsDeploymentKey != "dep-key" {
		t.Fatalf("expected deployment key to be used for metrics lookup, got %q", metrics.getMetricsDeploymentKey)
	}
}

func TestSimpleDelegatingManagementMethods(t *testing.T) {
	apps := &fakeApps{
		getByNameResult:         domain.App{ID: "app-1", Name: "demo"},
		updateResult:            domain.App{ID: "app-1", Name: "demo-renamed"},
		listCollaboratorsResult: map[string]domain.CollaboratorProperties{"user@example.com": {Permission: domain.PermissionOwner}},
	}
	deps := &fakeDeployments{
		getByNameResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production", Key: "dep-key"},
		updateResult:    domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production-2", Key: "dep-key"},
	}
	pkgs := &fakePackages{history: []domain.Package{{Label: "v1"}}}
	service := NewService(&fakeAccounts{getByIDAccount: domain.Account{ID: "acc-1", Email: "user@example.com"}}, &fakeAccessKeys{}, apps, deps, pkgs, &fakeMetrics{})

	account, err := service.GetAccount(context.Background(), "acc-1")
	if err != nil || account.Email != "user@example.com" {
		t.Fatalf("unexpected GetAccount() result %#v err=%v", account, err)
	}
	if err := service.DeleteAccessKey(context.Background(), "acc-1", "token-1"); err != nil {
		t.Fatalf("DeleteAccessKey() error = %v", err)
	}
	app, err := service.GetApp(context.Background(), "acc-1", "demo")
	if err != nil || app.ID != "app-1" {
		t.Fatalf("unexpected GetApp() result %#v err=%v", app, err)
	}
	app, err = service.UpdateApp(context.Background(), "acc-1", "demo", domain.AppPatchRequest{Name: "demo-renamed"})
	if err != nil || app.Name != "demo-renamed" {
		t.Fatalf("unexpected UpdateApp() result %#v err=%v", app, err)
	}
	if err := service.DeleteApp(context.Background(), "acc-1", "demo"); err != nil {
		t.Fatalf("DeleteApp() error = %v", err)
	}
	if err := service.TransferApp(context.Background(), "acc-1", "demo", "new@example.com"); err != nil {
		t.Fatalf("TransferApp() error = %v", err)
	}
	if err := service.AddCollaborator(context.Background(), "acc-1", "demo", "new@example.com"); err != nil {
		t.Fatalf("AddCollaborator() error = %v", err)
	}
	collabs, err := service.ListCollaborators(context.Background(), "acc-1", "demo")
	if err != nil || collabs["user@example.com"].Permission != domain.PermissionOwner {
		t.Fatalf("unexpected ListCollaborators() result %#v err=%v", collabs, err)
	}
	if err := service.RemoveCollaborator(context.Background(), "acc-1", "demo", "user@example.com"); err != nil {
		t.Fatalf("RemoveCollaborator() error = %v", err)
	}
	deployment, err := service.GetDeployment(context.Background(), "acc-1", "demo", "Production")
	if err != nil || deployment.ID != "dep-1" {
		t.Fatalf("unexpected GetDeployment() result %#v err=%v", deployment, err)
	}
	deployment, err = service.UpdateDeployment(context.Background(), "acc-1", "demo", "Production", domain.DeploymentPatchRequest{Name: "Production-2"})
	if err != nil || deployment.Name != "Production-2" {
		t.Fatalf("unexpected UpdateDeployment() result %#v err=%v", deployment, err)
	}
	if err := service.DeleteDeployment(context.Background(), "acc-1", "demo", "Production"); err != nil {
		t.Fatalf("DeleteDeployment() error = %v", err)
	}
	history, err := service.GetHistory(context.Background(), "acc-1", "demo", "Production")
	if err != nil || len(history) != 1 || history[0].Label != "v1" {
		t.Fatalf("unexpected GetHistory() result %#v err=%v", history, err)
	}
}

func TestHealthReturnsAccountErrorBeforeMetrics(t *testing.T) {
	service := NewService(&fakeAccounts{healthErr: errors.New("db down")}, &fakeAccessKeys{}, &fakeApps{}, &fakeDeployments{}, &fakePackages{}, &fakeMetrics{})

	err := service.Health(context.Background())
	if err == nil || err.Error() != "db down" {
		t.Fatalf("expected account health error, got %v", err)
	}
}

func TestHelpers(t *testing.T) {
	if first := selectedForRollout("client-1", 20, "v5"); first != selectedForRollout("client-1", 20, "v5") {
		t.Fatalf("expected deterministic rollout selection")
	}

	versionCases := []struct {
		input string
		want  string
	}{
		{input: "1", want: "1.0.0"},
		{input: "1.2", want: "1.2.0"},
		{input: "1.2.3", want: "1.2.3"},
	}
	for _, tc := range versionCases {
		if got := normalizeVersion(tc.input); got != tc.want {
			t.Fatalf("normalizeVersion(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}

	constraintCases := []struct {
		input string
		want  string
	}{
		{input: "1.2", want: "1.2.0"},
		{input: ">=1.0.0 <2.0.0", want: ">=1.0.0 <2.0.0"},
	}
	for _, tc := range constraintCases {
		if got := normalizeConstraint(tc.input); got != tc.want {
			t.Fatalf("normalizeConstraint(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}

	matchCases := []struct {
		target  string
		current string
		want    bool
	}{
		{target: "1.2", current: "1.2.0", want: true},
		{target: "~1.2.0", current: "1.2.5", want: true},
		{target: ">=2.0.0", current: "1.9.0", want: false},
		{target: "native", current: "native", want: true},
	}
	for _, tc := range matchCases {
		if got := matchesVersion(tc.target, tc.current); got != tc.want {
			t.Fatalf("matchesVersion(%q, %q) = %v, want %v", tc.target, tc.current, got, tc.want)
		}
	}

	token := generateToken("acc-1", time.Now().UnixNano())
	if len(token) != 40 || strings.Contains(token, "-") {
		t.Fatalf("unexpected generated token %q", token)
	}
}
