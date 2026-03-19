package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/yoophi/codepush-server-golang/internal/application"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

type authFakeAccounts struct {
	account          domain.Account
	resolveErr       error
	resolveTokenSeen string
}

func (f *authFakeAccounts) CheckHealth(context.Context) error { return nil }
func (f *authFakeAccounts) EnsureBootstrap(context.Context, domain.Account, domain.AccessKey) error {
	return nil
}
func (f *authFakeAccounts) GetByID(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *authFakeAccounts) GetByEmail(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *authFakeAccounts) ResolveAccountIDByAccessKey(_ context.Context, token string) (string, error) {
	f.resolveTokenSeen = token
	if f.resolveErr != nil {
		return "", f.resolveErr
	}
	return f.account.ID, nil
}

type authNoopAccessKeys struct{}

func (authNoopAccessKeys) List(context.Context, string) ([]domain.AccessKey, error) { return nil, nil }
func (authNoopAccessKeys) Create(context.Context, domain.AccessKey, string) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (authNoopAccessKeys) GetByName(context.Context, string, string) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (authNoopAccessKeys) Update(context.Context, domain.AccessKey) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (authNoopAccessKeys) Delete(context.Context, string, string) error { return nil }
func (authNoopAccessKeys) DeleteSessionsByCreator(context.Context, string, string) (int64, error) {
	return 0, nil
}

type authNoopApps struct{}

func (authNoopApps) List(context.Context, string) ([]domain.App, error) { return nil, nil }
func (authNoopApps) Create(context.Context, string, domain.App) (domain.App, error) {
	return domain.App{}, nil
}
func (authNoopApps) GetByName(context.Context, string, string) (domain.App, error) {
	return domain.App{}, nil
}
func (authNoopApps) Update(context.Context, string, domain.App) (domain.App, error) {
	return domain.App{}, nil
}
func (authNoopApps) Delete(context.Context, string, string) error { return nil }
func (authNoopApps) Transfer(context.Context, string, string, string) error {
	return nil
}
func (authNoopApps) AddCollaborator(context.Context, string, string, string) error {
	return nil
}
func (authNoopApps) ListCollaborators(context.Context, string, string) (map[string]domain.CollaboratorProperties, error) {
	return nil, nil
}
func (authNoopApps) RemoveCollaborator(context.Context, string, string, string) error { return nil }

type authNoopDeployments struct{}

func (authNoopDeployments) List(context.Context, string, string) ([]domain.Deployment, error) {
	return nil, nil
}
func (authNoopDeployments) Create(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (authNoopDeployments) GetByName(context.Context, string, string, string) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (authNoopDeployments) GetByKey(context.Context, string) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (authNoopDeployments) Update(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (authNoopDeployments) Delete(context.Context, string, string, string) error { return nil }

type authNoopPackages struct{}

func (authNoopPackages) ListHistory(context.Context, string, string, string) ([]domain.Package, error) {
	return nil, nil
}
func (authNoopPackages) ClearHistory(context.Context, string, string, string) error { return nil }
func (authNoopPackages) CommitRollback(context.Context, string, string, string, domain.Package) (domain.Package, error) {
	return domain.Package{}, nil
}

type authNoopMetrics struct{}

func (authNoopMetrics) CheckHealth(context.Context) error { return nil }
func (authNoopMetrics) IncrementDownload(context.Context, string, string) error {
	return nil
}
func (authNoopMetrics) ReportDeploy(context.Context, domain.DeploymentStatusReport) error { return nil }
func (authNoopMetrics) GetMetrics(context.Context, string) (map[string]domain.UpdateMetrics, error) {
	return nil, nil
}
func (authNoopMetrics) Clear(context.Context, string) error { return nil }

func TestRequireAuthRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := application.NewService(
		&authFakeAccounts{account: domain.Account{ID: "acc-1"}},
		authNoopAccessKeys{},
		authNoopApps{},
		authNoopDeployments{},
		authNoopPackages{},
		authNoopMetrics{},
	)
	router := gin.New()
	router.Use(RequireAuth(service))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuthAcceptsCaseInsensitiveBearerAndStoresAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	accounts := &authFakeAccounts{account: domain.Account{ID: "acc-1", Email: "user@example.com"}}
	service := application.NewService(accounts, authNoopAccessKeys{}, authNoopApps{}, authNoopDeployments{}, authNoopPackages{}, authNoopMetrics{})
	router := gin.New()
	router.Use(RequireAuth(service))
	router.GET("/", func(c *gin.Context) {
		account := CurrentAccount(c)
		c.JSON(http.StatusOK, gin.H{"email": account.Email})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "BeArEr   token-1   ")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if accounts.resolveTokenSeen != "token-1" {
		t.Fatalf("expected trimmed token, got %q", accounts.resolveTokenSeen)
	}
}

func TestRequireAuthMapsDomainErrorsToStatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	accounts := &authFakeAccounts{
		account:    domain.Account{ID: "acc-1"},
		resolveErr: domain.ErrExpired,
	}
	service := application.NewService(accounts, authNoopAccessKeys{}, authNoopApps{}, authNoopDeployments{}, authNoopPackages{}, authNoopMetrics{})
	router := gin.New()
	router.Use(RequireAuth(service))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", rec.Code)
	}
}
