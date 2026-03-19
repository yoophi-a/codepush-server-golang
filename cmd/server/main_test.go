package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/yoophi/codepush-server-golang/internal/adapters/persistence/postgres"
	"github.com/yoophi/codepush-server-golang/internal/application"
	"github.com/yoophi/codepush-server-golang/internal/config"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/core/ports"
)

type fakeAccounts struct {
	lastAccount domain.Account
	lastKey     domain.AccessKey
	err         error
}

func (f *fakeAccounts) CheckHealth(context.Context) error { return nil }
func (f *fakeAccounts) EnsureBootstrap(_ context.Context, account domain.Account, key domain.AccessKey) error {
	f.lastAccount = account
	f.lastKey = key
	return f.err
}
func (f *fakeAccounts) GetByID(context.Context, string) (domain.Account, error) {
	return domain.Account{}, nil
}
func (f *fakeAccounts) GetByEmail(context.Context, string) (domain.Account, error) {
	return domain.Account{}, nil
}
func (f *fakeAccounts) ResolveAccountIDByAccessKey(context.Context, string) (string, error) {
	return "", nil
}

type fakeBlobStorage struct {
	healthErr error
	called    bool
}

func (f *fakeBlobStorage) CheckHealth(context.Context) error {
	f.called = true
	return f.healthErr
}
func (f *fakeBlobStorage) PutObject(context.Context, string, []byte, string) (string, error) {
	return "", nil
}
func (f *fakeBlobStorage) DeleteObject(context.Context, string) error { return nil }

func TestBootstrapAccessKey(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cfg := config.Config{BootstrapAccessKey: "bootstrap-token"}

	got := bootstrapAccessKey(now, cfg)
	if got.Name != "bootstrap-token" || got.FriendlyName != "Bootstrap" || got.CreatedBy != "bootstrap" {
		t.Fatalf("unexpected bootstrap key %#v", got)
	}
	if got.Expires-got.CreatedTime != int64(365*24*time.Hour/time.Millisecond) {
		t.Fatalf("unexpected bootstrap key expiry delta %d", got.Expires-got.CreatedTime)
	}
}

func TestEnsureBootstrap(t *testing.T) {
	repo := &fakeAccounts{}
	cfg := config.Config{
		BootstrapEmail:     "admin@example.com",
		BootstrapName:      "Admin",
		BootstrapAccessKey: "bootstrap-token",
	}
	now := time.Unix(1700000000, 0)

	if err := ensureBootstrap(context.Background(), repo, cfg, now); err != nil {
		t.Fatalf("ensureBootstrap() error = %v", err)
	}
	if repo.lastAccount.Email != "admin@example.com" || repo.lastAccount.Name != "Admin" {
		t.Fatalf("unexpected bootstrap account %#v", repo.lastAccount)
	}
	if repo.lastKey.Name != "bootstrap-token" || repo.lastKey.CreatedTime != now.UnixMilli() {
		t.Fatalf("unexpected bootstrap key %#v", repo.lastKey)
	}
}

func TestLogBlobHealth(t *testing.T) {
	logBlobHealth(context.Background(), nil)

	blob := &fakeBlobStorage{}
	logBlobHealth(context.Background(), blob)
	if !blob.called {
		t.Fatalf("expected blob health check")
	}

	blob = &fakeBlobStorage{healthErr: assertErr{}}
	logBlobHealth(context.Background(), blob)
	if !blob.called {
		t.Fatalf("expected blob health check on error path")
	}
}

func TestNewBlobStorage(t *testing.T) {
	ctx := context.Background()

	minioBlob, err := newBlobStorage(ctx, config.Config{
		StorageBackend:       "minio",
		MinIOEndpoint:        "localhost:9000",
		MinIOAccessKeyID:     "minioadmin",
		MinIOSecretAccessKey: "minioadmin",
		MinIOBucket:          "codepush",
	})
	if err != nil || minioBlob == nil {
		t.Fatalf("expected minio blob storage, got blob=%T err=%v", minioBlob, err)
	}

	s3Blob, err := newBlobStorage(ctx, config.Config{
		StorageBackend:    "s3",
		S3Region:          "us-east-1",
		S3Endpoint:        "http://localhost:9000",
		S3AccessKeyID:     "minioadmin",
		S3SecretAccessKey: "minioadmin",
		S3Bucket:          "codepush",
		S3UsePathStyle:    true,
	})
	if err != nil || s3Blob == nil {
		t.Fatalf("expected s3 blob storage, got blob=%T err=%v", s3Blob, err)
	}

	if blob, err := newBlobStorage(ctx, config.Config{StorageBackend: "azure"}); err == nil || blob != nil {
		t.Fatalf("expected invalid backend error, got blob=%T err=%v", blob, err)
	}
}

func TestNewHTTPServer(t *testing.T) {
	service := application.NewService(&fakeAccounts{}, noopAccessKeys{}, noopApps{}, noopDeployments{}, noopPackages{}, noopMetrics{})
	server := newHTTPServer(3000, service)

	if server.Addr != ":3000" {
		t.Fatalf("unexpected server addr %q", server.Addr)
	}
	if server.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("unexpected header timeout %s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 30*time.Second || server.WriteTimeout != 30*time.Second || server.IdleTimeout != 120*time.Second {
		t.Fatalf("unexpected server timeouts: read=%s write=%s idle=%s", server.ReadTimeout, server.WriteTimeout, server.IdleTimeout)
	}
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rec := &responseRecorder{}
	server.Handler.ServeHTTP(rec, req)
	if rec.code != http.StatusOK {
		t.Fatalf("expected router to handle root, got %d", rec.code)
	}
}

func TestNewService(t *testing.T) {
	service := newService(appDeps{
		close:       func() {},
		accounts:    &fakeAccounts{},
		accessKeys:  noopAccessKeys{},
		apps:        noopApps{},
		deployments: noopDeployments{},
		packages:    noopPackages{},
	}, noopMetrics{}, config.Config{DefaultAccessKeyTTL: 1234})
	if service == nil {
		t.Fatalf("expected service instance")
	}
}

func TestDepsFromStore(t *testing.T) {
	deps := depsFromStore(&postgres.Store{})
	if deps.close == nil || deps.accounts == nil || deps.accessKeys == nil || deps.apps == nil || deps.deployments == nil || deps.packages == nil {
		t.Fatalf("expected all deps to be wired from store: %#v", deps)
	}
}

func TestRunReturnsConfigError(t *testing.T) {
	restore := stubMainFns()
	defer restore()

	t.Setenv("DATABASE_URL", "")
	err := run(context.Background())
	if err == nil || err.Error() != "DATABASE_URL is required" {
		t.Fatalf("expected config validation error, got %v", err)
	}
}

func TestRunLifecycle(t *testing.T) {
	restore := stubMainFns()
	defer restore()

	cfg := config.Config{
		DatabaseURL:          "postgres://demo",
		Port:                 3000,
		StorageBackend:       "minio",
		MinIOEndpoint:        "localhost:9000",
		MinIOAccessKeyID:     "minioadmin",
		MinIOSecretAccessKey: "minioadmin",
		MinIOBucket:          "codepush",
		BootstrapEmail:       "admin@example.com",
		BootstrapName:        "Admin",
		BootstrapAccessKey:   "bootstrap-token",
	}
	repo := &fakeAccounts{}
	metrics := &fakeMetricsCloser{}
	blob := &fakeBlobStorage{}
	listened := false
	storeClosed := false
	shutdownCalled := false
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})

	loadConfigFn = func() (config.Config, error) { return cfg, nil }
	newDepsFn = func(context.Context, config.Config) (appDeps, error) {
		return appDeps{
			close:       func() { storeClosed = true },
			accounts:    repo,
			accessKeys:  noopAccessKeys{},
			apps:        noopApps{},
			deployments: noopDeployments{},
			packages:    noopPackages{},
		}, nil
	}
	newMetricsFn = func(config.Config) metricsCloser { return metrics }
	newBlobStorageFn = func(context.Context, config.Config) (ports.BlobStorage, error) { return blob, nil }
	listenFn = func(server *http.Server) error {
		listened = true
		close(started)
		if server.Addr != ":3000" {
			t.Fatalf("unexpected server addr %q", server.Addr)
		}
		<-ctx.Done()
		return http.ErrServerClosed
	}
	shutdownFn = func(server *http.Server, shutdownCtx context.Context) error {
		shutdownCalled = true
		cancel()
		return nil
	}

	go func() {
		<-started
		cancel()
	}()

	err := run(ctx)
	if err != nil {
		t.Fatalf("expected graceful shutdown, got %v", err)
	}
	if !storeClosed || !metrics.closed || !blob.called || !listened || !shutdownCalled {
		t.Fatalf("expected lifecycle hooks to run store=%v metrics=%v blob=%v listened=%v shutdown=%v", storeClosed, metrics.closed, blob.called, listened, shutdownCalled)
	}
	if repo.lastAccount.Email != "admin@example.com" || repo.lastKey.Name != "bootstrap-token" {
		t.Fatalf("unexpected bootstrap payload account=%#v key=%#v", repo.lastAccount, repo.lastKey)
	}
}

func TestRunDependencyErrors(t *testing.T) {
	restore := stubMainFns()
	defer restore()

	cfg := config.Config{DatabaseURL: "postgres://demo"}
	loadConfigFn = func() (config.Config, error) { return cfg, nil }
	newDepsFn = func(context.Context, config.Config) (appDeps, error) { return appDeps{}, errors.New("store boom") }
	if err := run(context.Background()); err == nil || err.Error() != "store boom" {
		t.Fatalf("expected store error, got %v", err)
	}

	loadConfigFn = func() (config.Config, error) { return cfg, nil }
	newDepsFn = func(context.Context, config.Config) (appDeps, error) {
		return appDeps{
			close:       func() {},
			accounts:    &fakeAccounts{},
			accessKeys:  noopAccessKeys{},
			apps:        noopApps{},
			deployments: noopDeployments{},
			packages:    noopPackages{},
		}, nil
	}
	newBlobStorageFn = func(context.Context, config.Config) (ports.BlobStorage, error) { return nil, errors.New("blob boom") }
	if err := run(context.Background()); err == nil || err.Error() != "blob boom" {
		t.Fatalf("expected blob error, got %v", err)
	}
}

func TestRunReturnsListenError(t *testing.T) {
	restore := stubMainFns()
	defer restore()

	cfg := config.Config{DatabaseURL: "postgres://demo"}
	loadConfigFn = func() (config.Config, error) { return cfg, nil }
	newDepsFn = func(context.Context, config.Config) (appDeps, error) {
		return appDeps{
			close:       func() {},
			accounts:    &fakeAccounts{},
			accessKeys:  noopAccessKeys{},
			apps:        noopApps{},
			deployments: noopDeployments{},
			packages:    noopPackages{},
		}, nil
	}
	newMetricsFn = func(config.Config) metricsCloser { return &fakeMetricsCloser{} }
	newBlobStorageFn = func(context.Context, config.Config) (ports.BlobStorage, error) { return &fakeBlobStorage{}, nil }
	listenFn = func(*http.Server) error { return assertErr{} }

	if err := run(context.Background()); err == nil || err.Error() != "boom" {
		t.Fatalf("expected listener error, got %v", err)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }

type fakeMetricsCloser struct {
	closed bool
}

func (f *fakeMetricsCloser) CheckHealth(context.Context) error { return nil }
func (f *fakeMetricsCloser) IncrementDownload(context.Context, string, string) error {
	return nil
}
func (f *fakeMetricsCloser) ReportDeploy(context.Context, domain.DeploymentStatusReport) error {
	return nil
}
func (f *fakeMetricsCloser) GetMetrics(context.Context, string) (map[string]domain.UpdateMetrics, error) {
	return nil, nil
}
func (f *fakeMetricsCloser) Clear(context.Context, string) error { return nil }
func (f *fakeMetricsCloser) Close() error {
	f.closed = true
	return nil
}

func stubMainFns() func() {
	prevLoadConfig := loadConfigFn
	prevNewDeps := newDepsFn
	prevNewMetrics := newMetricsFn
	prevListen := listenFn
	prevShutdown := shutdownFn
	prevNewBlobStorage := newBlobStorageFn
	return func() {
		loadConfigFn = prevLoadConfig
		newDepsFn = prevNewDeps
		newMetricsFn = prevNewMetrics
		listenFn = prevListen
		shutdownFn = prevShutdown
		newBlobStorageFn = prevNewBlobStorage
	}
}

type noopAccessKeys struct{}

func (noopAccessKeys) List(context.Context, string) ([]domain.AccessKey, error) { return nil, nil }
func (noopAccessKeys) Create(context.Context, domain.AccessKey, string) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (noopAccessKeys) GetByName(context.Context, string, string) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (noopAccessKeys) Update(context.Context, domain.AccessKey) (domain.AccessKey, error) {
	return domain.AccessKey{}, nil
}
func (noopAccessKeys) Delete(context.Context, string, string) error { return nil }
func (noopAccessKeys) DeleteSessionsByCreator(context.Context, string, string) (int64, error) {
	return 0, nil
}

type noopApps struct{}

func (noopApps) List(context.Context, string) ([]domain.App, error) { return nil, nil }
func (noopApps) Create(context.Context, string, domain.App) (domain.App, error) {
	return domain.App{}, nil
}
func (noopApps) GetByName(context.Context, string, string) (domain.App, error) {
	return domain.App{}, nil
}
func (noopApps) Update(context.Context, string, domain.App) (domain.App, error) {
	return domain.App{}, nil
}
func (noopApps) Delete(context.Context, string, string) error           { return nil }
func (noopApps) Transfer(context.Context, string, string, string) error { return nil }
func (noopApps) AddCollaborator(context.Context, string, string, string) error {
	return nil
}
func (noopApps) ListCollaborators(context.Context, string, string) (map[string]domain.CollaboratorProperties, error) {
	return nil, nil
}
func (noopApps) RemoveCollaborator(context.Context, string, string, string) error { return nil }

type noopDeployments struct{}

func (noopDeployments) List(context.Context, string, string) ([]domain.Deployment, error) {
	return nil, nil
}
func (noopDeployments) Create(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (noopDeployments) GetByName(context.Context, string, string, string) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (noopDeployments) GetByKey(context.Context, string) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (noopDeployments) Update(context.Context, string, string, domain.Deployment) (domain.Deployment, error) {
	return domain.Deployment{}, nil
}
func (noopDeployments) Delete(context.Context, string, string, string) error { return nil }

type noopPackages struct{}

func (noopPackages) ListHistory(context.Context, string, string, string) ([]domain.Package, error) {
	return nil, nil
}
func (noopPackages) ClearHistory(context.Context, string, string, string) error { return nil }
func (noopPackages) CommitRollback(context.Context, string, string, string, domain.Package) (domain.Package, error) {
	return domain.Package{}, nil
}

type noopMetrics struct{}

func (noopMetrics) CheckHealth(context.Context) error                       { return nil }
func (noopMetrics) IncrementDownload(context.Context, string, string) error { return nil }
func (noopMetrics) ReportDeploy(context.Context, domain.DeploymentStatusReport) error {
	return nil
}
func (noopMetrics) GetMetrics(context.Context, string) (map[string]domain.UpdateMetrics, error) {
	return nil, nil
}
func (noopMetrics) Clear(context.Context, string) error { return nil }

type responseRecorder struct {
	header http.Header
	code   int
	body   []byte
}

func (r *responseRecorder) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}
func (r *responseRecorder) WriteHeader(statusCode int) { r.code = statusCode }
