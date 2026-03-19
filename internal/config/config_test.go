package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected DATABASE_URL validation error")
	}
}

func TestLoadUsesFallbacksAndParsesValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://demo")
	t.Setenv("APP_ENV", "test")
	t.Setenv("PORT", "8080")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_POOL_SIZE", "40")
	t.Setenv("REDIS_MIN_IDLE_CONNS", "7")
	t.Setenv("REDIS_MAX_RETRIES", "9")
	t.Setenv("REDIS_DIAL_TIMEOUT_SEC", "11")
	t.Setenv("REDIS_READ_TIMEOUT_SEC", "12")
	t.Setenv("REDIS_WRITE_TIMEOUT_SEC", "13")
	t.Setenv("DEFAULT_ACCESS_KEY_TTL", "98765")
	t.Setenv("STORAGE_BACKEND", "MINIO")
	t.Setenv("S3_USE_PATH_STYLE", "yes")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("BOOTSTRAP_ACCOUNT_EMAIL", "owner@example.com")
	t.Setenv("BOOTSTRAP_ACCOUNT_NAME", "Owner")
	t.Setenv("BOOTSTRAP_ACCESS_KEY", "bootstrap-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AppEnv != "test" || cfg.Port != 8080 || cfg.RedisDB != 2 {
		t.Fatalf("unexpected parsed config: %#v", cfg)
	}
	if cfg.RedisPoolSize != 40 || cfg.RedisMinIdleConns != 7 || cfg.RedisMaxRetries != 9 {
		t.Fatalf("unexpected redis pool config: %#v", cfg)
	}
	if cfg.RedisDialTimeoutSec != 11 || cfg.RedisReadTimeoutSec != 12 || cfg.RedisWriteTimeoutSec != 13 {
		t.Fatalf("unexpected redis timeout config: %#v", cfg)
	}
	if cfg.DefaultAccessKeyTTL != 98765 {
		t.Fatalf("unexpected default access key ttl: %#v", cfg)
	}
	if cfg.StorageBackend != "minio" {
		t.Fatalf("expected lower-cased storage backend, got %q", cfg.StorageBackend)
	}
	if !cfg.S3UsePathStyle || !cfg.MinIOUseSSL {
		t.Fatalf("expected bool parsing to succeed, got %#v", cfg)
	}
	if cfg.BootstrapEmail != "owner@example.com" || cfg.BootstrapAccessKey != "bootstrap-token" {
		t.Fatalf("unexpected bootstrap config: %#v", cfg)
	}
}

func TestHelpersFallBackOnInvalidValues(t *testing.T) {
	t.Setenv("BAD_INT", "nope")
	t.Setenv("BAD_BOOL", "maybe")

	if got := getEnv("MISSING_ENV", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback env value, got %q", got)
	}
	if got := getEnvInt("BAD_INT", 42); got != 42 {
		t.Fatalf("expected fallback int, got %d", got)
	}
	if got := getEnvBool("BAD_BOOL", true); !got {
		t.Fatalf("expected fallback bool, got %v", got)
	}
	if got := getEnvInt64("BAD_INT", 64); got != 64 {
		t.Fatalf("expected fallback int64, got %d", got)
	}
}

func TestLoadUsesRedisDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://demo")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.RedisPoolSize != 20 || cfg.RedisMinIdleConns != 5 || cfg.RedisMaxRetries != 3 {
		t.Fatalf("unexpected redis defaults: %#v", cfg)
	}
	if cfg.RedisDialTimeoutSec != 5 || cfg.RedisReadTimeoutSec != 3 || cfg.RedisWriteTimeoutSec != 3 {
		t.Fatalf("unexpected redis timeout defaults: %#v", cfg)
	}
	if cfg.DefaultAccessKeyTTL <= 0 {
		t.Fatalf("expected default access key ttl, got %#v", cfg)
	}
}
