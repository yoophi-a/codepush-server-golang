package ginhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/yoophi/codepush-server-golang/internal/application"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

type routerFakeAccounts struct {
	account       domain.Account
	healthErr     error
	resolveErr    error
	resolveToken  string
	resolveID     string
	getByIDResult domain.Account
	getByIDErr    error
}

func (f *routerFakeAccounts) CheckHealth(context.Context) error { return f.healthErr }
func (f *routerFakeAccounts) EnsureBootstrap(context.Context, domain.Account, domain.AccessKey) error {
	return nil
}
func (f *routerFakeAccounts) GetByID(context.Context, string) (domain.Account, error) {
	if f.getByIDErr != nil {
		return domain.Account{}, f.getByIDErr
	}
	if f.getByIDResult.ID != "" {
		return f.getByIDResult, nil
	}
	return f.account, nil
}
func (f *routerFakeAccounts) GetByEmail(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *routerFakeAccounts) ResolveAccountIDByAccessKey(_ context.Context, token string) (string, error) {
	f.resolveToken = token
	if f.resolveErr != nil {
		return "", f.resolveErr
	}
	if f.resolveID != "" {
		return f.resolveID, nil
	}
	return f.account.ID, nil
}

type routerFakeAccessKeys struct {
	listResult      []domain.AccessKey
	createResult    domain.AccessKey
	createErr       error
	getByNameResult domain.AccessKey
	getByNameErr    error
}

func (f *routerFakeAccessKeys) List(context.Context, string) ([]domain.AccessKey, error) {
	return f.listResult, nil
}
func (f *routerFakeAccessKeys) Create(context.Context, domain.AccessKey, string) (domain.AccessKey, error) {
	return f.createResult, f.createErr
}
func (f *routerFakeAccessKeys) GetByName(context.Context, string, string) (domain.AccessKey, error) {
	return f.getByNameResult, f.getByNameErr
}
func (f *routerFakeAccessKeys) Update(context.Context, domain.AccessKey) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (f *routerFakeAccessKeys) Delete(context.Context, string, string) error { return nil }
func (f *routerFakeAccessKeys) DeleteSessionsByCreator(context.Context, string, string) (int64, error) {
	return 1, nil
}

type routerFakeApps struct {
	listResult []domain.App
	listErr    error
	getByName  domain.App
}

func (f *routerFakeApps) List(context.Context, string) ([]domain.App, error) {
	return f.listResult, f.listErr
}
func (f *routerFakeApps) Create(context.Context, string, domain.App) (domain.App, error) {
	return domain.App{}, nil
}
func (f *routerFakeApps) GetByName(context.Context, string, string) (domain.App, error) {
	return f.getByName, nil
}
func (f *routerFakeApps) Update(context.Context, string, domain.App) (domain.App, error) {
	return domain.App{}, nil
}
func (f *routerFakeApps) Delete(context.Context, string, string) error { return nil }
func (f *routerFakeApps) Transfer(context.Context, string, string, string) error {
	return nil
}
func (f *routerFakeApps) AddCollaborator(context.Context, string, string, string) error {
	return nil
}
func (f *routerFakeApps) ListCollaborators(context.Context, string, string) (map[string]domain.CollaboratorProperties, error) {
	return nil, nil
}
func (f *routerFakeApps) RemoveCollaborator(context.Context, string, string, string) error {
	return nil
}

type routerFakeDeployments struct {
	getByKeyResult  domain.Deployment
	getByKeyErr     error
	getByNameResult domain.Deployment
}

func (f *routerFakeDeployments) List(context.Context, string, string) ([]domain.Deployment, error) {
	return nil, nil
}
func (f *routerFakeDeployments) Create(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (f *routerFakeDeployments) GetByName(context.Context, string, string, string) (domain.Deployment, error) {
	return f.getByNameResult, nil
}
func (f *routerFakeDeployments) GetByKey(context.Context, string) (domain.Deployment, error) {
	return f.getByKeyResult, f.getByKeyErr
}
func (f *routerFakeDeployments) Update(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (f *routerFakeDeployments) Delete(context.Context, string, string, string) error { return nil }

type routerFakePackages struct {
	history []domain.Package
}

func (f *routerFakePackages) ListHistory(context.Context, string, string, string) ([]domain.Package, error) {
	return f.history, nil
}
func (f *routerFakePackages) ClearHistory(context.Context, string, string, string) error { return nil }
func (f *routerFakePackages) CommitRollback(context.Context, string, string, string, domain.Package) (domain.Package, error) {
	return domain.Package{}, nil
}

type routerFakeMetrics struct {
	healthErr            error
	reportDeployErr      error
	reportDownloadErr    error
	reportDeployCalled   bool
	incrementDownloadKey string
	getMetricsResult     map[string]domain.UpdateMetrics
}

func (f *routerFakeMetrics) CheckHealth(context.Context) error { return f.healthErr }
func (f *routerFakeMetrics) IncrementDownload(context.Context, string, string) error {
	return f.reportDownloadErr
}
func (f *routerFakeMetrics) ReportDeploy(context.Context, domain.DeploymentStatusReport) error {
	f.reportDeployCalled = true
	return f.reportDeployErr
}
func (f *routerFakeMetrics) GetMetrics(context.Context, string) (map[string]domain.UpdateMetrics, error) {
	return f.getMetricsResult, nil
}
func (f *routerFakeMetrics) Clear(context.Context, string) error { return nil }

func newRouterForTest(
	accounts *routerFakeAccounts,
	accessKeys *routerFakeAccessKeys,
	apps *routerFakeApps,
	deps *routerFakeDeployments,
	pkgs *routerFakePackages,
	metrics *routerFakeMetrics,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	service := application.NewService(accounts, accessKeys, apps, deps, pkgs, metrics)
	return NewRouter(service)
}

func TestHealthEndpoint(t *testing.T) {
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		&routerFakeAccessKeys{},
		&routerFakeApps{},
		&routerFakeDeployments{},
		&routerFakePackages{},
		&routerFakeMetrics{},
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestStaticRoutes(t *testing.T) {
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		&routerFakeAccessKeys{},
		&routerFakeApps{},
		&routerFakeDeployments{},
		&routerFakePackages{},
		&routerFakeMetrics{},
	)

	for _, tc := range []struct {
		path string
		code int
		want string
	}{
		{path: "/", code: http.StatusOK, want: "Welcome to the CodePush REST API!"},
		{path: "/auth/login", code: http.StatusOK, want: "<h1>Login</h1>"},
		{path: "/auth/register", code: http.StatusOK, want: "<h1>Register</h1>"},
		{path: "/auth/login/github", code: http.StatusNotImplemented, want: "oauth flow not implemented"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tc.code {
			t.Fatalf("%s expected %d, got %d", tc.path, tc.code, rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(tc.want)) {
			t.Fatalf("%s expected body to contain %q, got %s", tc.path, tc.want, rec.Body.String())
		}
	}
}

func TestHealthEndpointReturnsErrorPayload(t *testing.T) {
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}, healthErr: domain.ErrUnauthorized},
		&routerFakeAccessKeys{},
		&routerFakeApps{},
		&routerFakeDeployments{},
		&routerFakePackages{},
		&routerFakeMetrics{},
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestUpdateCheckEndpointsAcceptAliasQueryParameters(t *testing.T) {
	deps := &routerFakeDeployments{getByKeyResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Key: "dep-key"}}
	pkgs := &routerFakePackages{history: []domain.Package{
		{Label: "v1", AppVersion: "1.0.0", PackageHash: "hash-1", BlobURL: "http://example.com/v1.zip", Size: 1},
	}}
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		&routerFakeAccessKeys{},
		&routerFakeApps{},
		deps,
		pkgs,
		&routerFakeMetrics{},
	)

	req := httptest.NewRequest(http.MethodGet, "/v0.1/public/codepush/update_check?deploymentKey=dep-key&app_version=1.0.0&client_unique_id=client-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"packageHash":"hash-1"`)) {
		t.Fatalf("expected update payload, got %s", rec.Body.String())
	}
}

func TestReportEndpointsRejectMalformedJSON(t *testing.T) {
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		&routerFakeAccessKeys{},
		&routerFakeApps{},
		&routerFakeDeployments{},
		&routerFakePackages{},
		&routerFakeMetrics{},
	)

	paths := []string{"/reportStatus/deploy", "/reportStatus/download"}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s expected 400, got %d", path, rec.Code)
		}
	}
}

func TestAuthedEndpointsRequireBearerToken(t *testing.T) {
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		&routerFakeAccessKeys{},
		&routerFakeApps{},
		&routerFakeDeployments{},
		&routerFakePackages{},
		&routerFakeMetrics{},
	)

	req := httptest.NewRequest(http.MethodGet, "/authenticated", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthedAppsEndpointMapsServiceError(t *testing.T) {
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		&routerFakeAccessKeys{},
		&routerFakeApps{listErr: domain.ErrNotFound},
		&routerFakeDeployments{},
		&routerFakePackages{},
		&routerFakeMetrics{},
	)

	req := httptest.NewRequest(http.MethodGet, "/apps", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAuthedAccessKeysCreateAndRollbackStatusCodes(t *testing.T) {
	accessKeys := &routerFakeAccessKeys{
		createResult: domain.AccessKey{FriendlyName: "demo"},
	}
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		accessKeys,
		&routerFakeApps{getByName: domain.App{ID: "app-1", Name: "demo"}},
		&routerFakeDeployments{getByNameResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production"}},
		&routerFakePackages{history: []domain.Package{
			{Label: "v1", AppVersion: "1.0.0", PackageHash: "hash-1"},
			{Label: "v2", AppVersion: "1.0.0", PackageHash: "hash-2"},
		}},
		&routerFakeMetrics{},
	)

	body, _ := json.Marshal(domain.AccessKeyRequest{FriendlyName: "demo"})
	req := httptest.NewRequest(http.MethodPost, "/accessKeys", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer token-1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/apps/demo/deployments/Production/rollback", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 rollback response, got %d", rec.Code)
	}
}

func TestAuthedReadEndpoints(t *testing.T) {
	router := newRouterForTest(
		&routerFakeAccounts{
			account:       domain.Account{ID: "acc-1"},
			getByIDResult: domain.Account{ID: "acc-1", Email: "user@example.com"},
		},
		&routerFakeAccessKeys{
			listResult:      []domain.AccessKey{{FriendlyName: "demo"}},
			getByNameResult: domain.AccessKey{FriendlyName: "demo"},
		},
		&routerFakeApps{
			listResult: []domain.App{{Name: "demo"}},
			getByName:  domain.App{ID: "app-1", Name: "demo"},
		},
		&routerFakeDeployments{
			getByNameResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production", Key: "dep-key"},
		},
		&routerFakePackages{history: []domain.Package{{Label: "v1"}}},
		&routerFakeMetrics{getMetricsResult: map[string]domain.UpdateMetrics{"v1": {Installed: 1}}},
	)

	for _, path := range []string{
		"/authenticated",
		"/account",
		"/accessKeys",
		"/accessKeys/demo",
		"/apps",
		"/apps/demo",
		"/apps/demo/collaborators",
		"/apps/demo/deployments",
		"/apps/demo/deployments/Production",
		"/apps/demo/deployments/Production/history",
		"/apps/demo/deployments/Production/metrics",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer token-1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d", path, rec.Code)
		}
	}
}

func TestAuthedWriteEndpoints(t *testing.T) {
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		&routerFakeAccessKeys{
			createResult:    domain.AccessKey{FriendlyName: "demo"},
			getByNameResult: domain.AccessKey{FriendlyName: "demo"},
		},
		&routerFakeApps{getByName: domain.App{ID: "app-1", Name: "demo"}},
		&routerFakeDeployments{getByNameResult: domain.Deployment{ID: "dep-1", AppID: "app-1", Name: "Production", Key: "dep-key"}},
		&routerFakePackages{history: []domain.Package{
			{Label: "v1", AppVersion: "1.0.0", PackageHash: "hash-1"},
			{Label: "v2", AppVersion: "1.0.0", PackageHash: "hash-2"},
		}},
		&routerFakeMetrics{},
	)

	for _, tc := range []struct {
		method string
		path   string
		body   string
		code   int
	}{
		{method: http.MethodPatch, path: "/accessKeys/demo", body: `{"friendlyName":"renamed"}`, code: http.StatusOK},
		{method: http.MethodDelete, path: "/accessKeys/demo", code: http.StatusNoContent},
		{method: http.MethodDelete, path: "/sessions/127.0.0.1", code: http.StatusNoContent},
		{method: http.MethodPost, path: "/apps", body: `{"name":"demo"}`, code: http.StatusCreated},
		{method: http.MethodPatch, path: "/apps/demo", body: `{"name":"demo-2"}`, code: http.StatusOK},
		{method: http.MethodDelete, path: "/apps/demo", code: http.StatusNoContent},
		{method: http.MethodPost, path: "/apps/demo/transfer/user@example.com", code: http.StatusCreated},
		{method: http.MethodPost, path: "/apps/demo/collaborators/user@example.com", code: http.StatusCreated},
		{method: http.MethodDelete, path: "/apps/demo/collaborators/user@example.com", code: http.StatusNoContent},
		{method: http.MethodPost, path: "/apps/demo/deployments", body: `{"name":"Production"}`, code: http.StatusCreated},
		{method: http.MethodPatch, path: "/apps/demo/deployments/Production", body: `{"name":"Production-2"}`, code: http.StatusOK},
		{method: http.MethodDelete, path: "/apps/demo/deployments/Production", code: http.StatusNoContent},
		{method: http.MethodDelete, path: "/apps/demo/deployments/Production/history", code: http.StatusNoContent},
		{method: http.MethodPost, path: "/apps/demo/deployments/Production/rollback/v1", code: http.StatusCreated},
	} {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		req.Header.Set("Authorization", "Bearer token-1")
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tc.code {
			t.Fatalf("%s %s expected %d, got %d body=%s", tc.method, tc.path, tc.code, rec.Code, rec.Body.String())
		}
	}
}

func TestPublicReportAliasEndpoints(t *testing.T) {
	metrics := &routerFakeMetrics{}
	router := newRouterForTest(
		&routerFakeAccounts{account: domain.Account{ID: "acc-1"}},
		&routerFakeAccessKeys{},
		&routerFakeApps{},
		&routerFakeDeployments{},
		&routerFakePackages{},
		metrics,
	)

	for _, tc := range []struct {
		path string
		body string
	}{
		{path: "/v0.1/public/codepush/report_status/deploy", body: `{"deploymentKey":"dep-key","label":"v1"}`},
		{path: "/v0.1/public/codepush/report_status/download", body: `{"deploymentKey":"dep-key","label":"v1"}`},
	} {
		req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d", tc.path, rec.Code)
		}
	}
}

func TestHelpers(t *testing.T) {
	if got := firstNonEmpty("", "fallback", "later"); got != "fallback" {
		t.Fatalf("firstNonEmpty() = %q, want fallback", got)
	}

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	writeError(c, domain.ErrConflict)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}
