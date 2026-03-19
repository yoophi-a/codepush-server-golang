package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv               string
	Port                 int
	DatabaseURL          string
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	RedisPoolSize        int
	RedisMinIdleConns    int
	RedisMaxRetries      int
	RedisDialTimeoutSec  int
	RedisReadTimeoutSec  int
	RedisWriteTimeoutSec int
	DefaultAccessKeyTTL  int64
	StorageBackend       string
	S3Bucket             string
	S3Region             string
	S3Endpoint           string
	S3AccessKeyID        string
	S3SecretAccessKey    string
	S3UsePathStyle       bool
	MinIOEndpoint        string
	MinIOAccessKeyID     string
	MinIOSecretAccessKey string
	MinIOBucket          string
	MinIOUseSSL          bool
	BootstrapEmail       string
	BootstrapName        string
	BootstrapAccessKey   string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		Port:                 getEnvInt("PORT", 3000),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        os.Getenv("REDIS_PASSWORD"),
		RedisDB:              getEnvInt("REDIS_DB", 0),
		RedisPoolSize:        getEnvInt("REDIS_POOL_SIZE", 20),
		RedisMinIdleConns:    getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
		RedisMaxRetries:      getEnvInt("REDIS_MAX_RETRIES", 3),
		RedisDialTimeoutSec:  getEnvInt("REDIS_DIAL_TIMEOUT_SEC", 5),
		RedisReadTimeoutSec:  getEnvInt("REDIS_READ_TIMEOUT_SEC", 3),
		RedisWriteTimeoutSec: getEnvInt("REDIS_WRITE_TIMEOUT_SEC", 3),
		DefaultAccessKeyTTL:  getEnvInt64("DEFAULT_ACCESS_KEY_TTL", int64(60*24*time.Hour/time.Millisecond)),
		StorageBackend:       strings.ToLower(getEnv("STORAGE_BACKEND", "s3")),
		S3Bucket:             os.Getenv("S3_BUCKET"),
		S3Region:             getEnv("S3_REGION", "us-east-1"),
		S3Endpoint:           os.Getenv("S3_ENDPOINT"),
		S3AccessKeyID:        os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:    os.Getenv("S3_SECRET_ACCESS_KEY"),
		S3UsePathStyle:       getEnvBool("S3_USE_PATH_STYLE", false),
		MinIOEndpoint:        os.Getenv("MINIO_ENDPOINT"),
		MinIOAccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
		MinIOBucket:          os.Getenv("MINIO_BUCKET"),
		MinIOUseSSL:          getEnvBool("MINIO_USE_SSL", false),
		BootstrapEmail:       getEnv("BOOTSTRAP_ACCOUNT_EMAIL", "admin@example.com"),
		BootstrapName:        getEnv("BOOTSTRAP_ACCOUNT_NAME", "Admin"),
		BootstrapAccessKey:   getEnv("BOOTSTRAP_ACCESS_KEY", "dev-access-key"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "y":
			return true
		case "0", "false", "no", "n":
			return false
		}
	}
	return fallback
}
