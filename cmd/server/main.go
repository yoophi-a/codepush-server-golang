package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	httpadapter "github.com/yoophi/codepush-server-golang/internal/adapters/http/ginhandler"
	redisadapter "github.com/yoophi/codepush-server-golang/internal/adapters/metrics/redis"
	"github.com/yoophi/codepush-server-golang/internal/adapters/persistence/postgres"
	miniostorage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/minio"
	s3storage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/s3"
	"github.com/yoophi/codepush-server-golang/internal/application"
	"github.com/yoophi/codepush-server-golang/internal/config"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/core/ports"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	store, err := postgres.NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	metrics := redisadapter.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer func() {
		_ = metrics.Close()
	}()

	var blob ports.BlobStorage
	switch cfg.StorageBackend {
	case "minio":
		blob, err = miniostorage.New(cfg.MinIOEndpoint, cfg.MinIOAccessKeyID, cfg.MinIOSecretAccessKey, cfg.MinIOBucket, cfg.MinIOUseSSL)
	default:
		blob, err = s3storage.New(ctx, cfg.S3Region, cfg.S3Endpoint, cfg.S3AccessKeyID, cfg.S3SecretAccessKey, cfg.S3Bucket, cfg.S3UsePathStyle)
	}
	if err != nil {
		log.Fatal(err)
	}
	if blob != nil {
		if err := blob.CheckHealth(ctx); err != nil {
			log.Printf("blob storage health check failed: %v", err)
		}
	}

	accounts := store.Accounts()
	if err := accounts.EnsureBootstrap(ctx, domain.Account{
		Email: cfg.BootstrapEmail,
		Name:  cfg.BootstrapName,
	}, domain.AccessKey{
		Name:         cfg.BootstrapAccessKey,
		FriendlyName: "Bootstrap",
		Description:  "Bootstrap access key",
		CreatedBy:    "bootstrap",
		CreatedTime:  time.Now().UnixMilli(),
		Expires:      time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
	}); err != nil {
		log.Fatal(err)
	}

	service := application.NewService(
		accounts,
		store.AccessKeys(),
		store.Apps(),
		store.Deployments(),
		store.Packages(),
		metrics,
	)

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           httpadapter.NewRouter(service),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
