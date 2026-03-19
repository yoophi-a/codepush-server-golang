package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "github.com/yoophi/codepush-server-golang/internal/adapters/http/ginhandler"
	redisadapter "github.com/yoophi/codepush-server-golang/internal/adapters/metrics/redis"
	"github.com/yoophi/codepush-server-golang/internal/adapters/persistence/postgres"
	storagefactory "github.com/yoophi/codepush-server-golang/internal/adapters/storage/factory"
	"github.com/yoophi/codepush-server-golang/internal/application"
	"github.com/yoophi/codepush-server-golang/internal/config"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/core/ports"
)

type metricsCloser interface {
	ports.MetricsRepository
	Close() error
}

type appDeps struct {
	close       func()
	accounts    ports.AccountRepository
	accessKeys  ports.AccessKeyRepository
	apps        ports.AppRepository
	deployments ports.DeploymentRepository
	packages    ports.PackageRepository
}

var (
	loadConfigFn = config.Load
	newDepsFn    = func(ctx context.Context, cfg config.Config) (appDeps, error) {
		store, err := postgres.NewStore(ctx, cfg.DatabaseURL)
		if err != nil {
			return appDeps{}, err
		}
		return depsFromStore(store), nil
	}
	newMetricsFn = func(cfg config.Config) metricsCloser {
		return redisadapter.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, redisadapter.ClientOptions{
			PoolSize:     cfg.RedisPoolSize,
			MinIdleConns: cfg.RedisMinIdleConns,
			MaxRetries:   cfg.RedisMaxRetries,
			DialTimeout:  time.Duration(cfg.RedisDialTimeoutSec) * time.Second,
			ReadTimeout:  time.Duration(cfg.RedisReadTimeoutSec) * time.Second,
			WriteTimeout: time.Duration(cfg.RedisWriteTimeoutSec) * time.Second,
		})
	}
	listenFn = func(server *http.Server) error {
		return server.ListenAndServe()
	}
	shutdownFn = func(server *http.Server, ctx context.Context) error {
		return server.Shutdown(ctx)
	}
	newBlobStorageFn = newBlobStorage
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg, err := loadConfigFn()
	if err != nil {
		return err
	}

	deps, err := newDepsFn(ctx, cfg)
	if err != nil {
		return err
	}
	defer deps.close()

	metrics := newMetricsFn(cfg)
	defer func() {
		_ = metrics.Close()
	}()

	blob, err := newBlobStorageFn(ctx, cfg)
	if err != nil {
		return err
	}
	logBlobHealth(ctx, blob)

	if err := ensureBootstrap(ctx, deps.accounts, cfg, time.Now()); err != nil {
		return err
	}

	service := newService(deps, metrics, cfg)
	server := newHTTPServer(cfg.Port, service)

	serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", server.Addr)
		errCh <- listenFn(server)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-serveCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := shutdownFn(server, shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func newBlobStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
	if err := storagefactory.ValidateBackend(cfg.StorageBackend); err != nil {
		return nil, err
	}
	return storagefactory.New(ctx, cfg)
}

func logBlobHealth(ctx context.Context, blob ports.BlobStorage) {
	if blob == nil {
		return
	}
	if err := blob.CheckHealth(ctx); err != nil {
		log.Printf("blob storage health check failed: %v", err)
	}
}

func bootstrapAccessKey(now time.Time, cfg config.Config) domain.AccessKey {
	return domain.AccessKey{
		Name:         cfg.BootstrapAccessKey,
		FriendlyName: "Bootstrap",
		Description:  "Bootstrap access key",
		CreatedBy:    "bootstrap",
		CreatedTime:  now.UnixMilli(),
		Expires:      now.Add(365 * 24 * time.Hour).UnixMilli(),
	}
}

func ensureBootstrap(ctx context.Context, accounts ports.AccountRepository, cfg config.Config, now time.Time) error {
	return accounts.EnsureBootstrap(ctx, domain.Account{
		Email: cfg.BootstrapEmail,
		Name:  cfg.BootstrapName,
	}, bootstrapAccessKey(now, cfg))
}

func newService(deps appDeps, metrics ports.MetricsRepository, cfg config.Config) *application.Service {
	return application.NewService(
		deps.accounts,
		deps.accessKeys,
		deps.apps,
		deps.deployments,
		deps.packages,
		metrics,
		application.WithDefaultAccessKeyTTL(cfg.DefaultAccessKeyTTL),
	)
}

func depsFromStore(store *postgres.Store) appDeps {
	return appDeps{
		close:       store.Close,
		accounts:    store.Accounts(),
		accessKeys:  store.AccessKeys(),
		apps:        store.Apps(),
		deployments: store.Deployments(),
		packages:    store.Packages(),
	}
}

func newHTTPServer(port int, service ports.HTTPAPI) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           httpadapter.NewRouter(service),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}
