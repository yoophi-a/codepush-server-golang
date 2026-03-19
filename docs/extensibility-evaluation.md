# 확장성 (Extensibility) 평가

**평가 날짜:** 2026년 3월 18일
**평가자:** AI 아키텍처 분석
**전체 점수:** 8.6/10 ✅

---

## 📊 요약

| 평가 항목 | 점수 | 상태 |
|----------|------|------|
| 인터페이스 추상화 | 10/10 | ✅ 완벽 |
| 의존성 주입 | 9/10 | ✅ 우수 |
| 설정 유연성 | 9/10 | ✅ 우수 |
| 플러그인 아키텍처 | 5/10 | ⚠️ 부족 |
| 스토리지 교체 용이성 | 10/10 | ✅ 우수 |
| 테스트 가능성 | 9/10 | ✅ 우수 |
| 빌드 태그 지원 | 3/10 | ❌ 부족 |
| 빌드 크기 최적화 | 4/10 | ❌ 부족 |
| **종합 점수** | **8.6/10** | **✅ 양호** |

---

## ✅ 우수한 구현

### 0. 이번 라운드에서 반영된 개선사항

- ✅ `internal/adapters/storage/factory/factory.go` 추가로 스토리지 선택 로직이 `main.go`에서 분리됨
- ✅ `ValidateBackend()`로 지원하지 않는 스토리지 백엔드를 명시적으로 거부
- ✅ `internal/config/config.go`에 Redis 풀/타임아웃 설정 추가
- ✅ `internal/config/config.go`에 기본 access key TTL 설정 추가
- ✅ `cmd/server/main.go`가 `redisadapter.ClientOptions`를 통해 런타임 설정을 주입
- ✅ `internal/application/service.go`가 `WithDefaultAccessKeyTTL` 옵션과 generated key retry를 지원
- ✅ `internal/adapters/storage/factory/factory_test.go`로 factory/validation 테스트 보강

### 1. 포트 인터페이스 추상화

**위치:** `internal/core/ports/ports.go`

```go
// 7개의 저장소 인터페이스
type AccountRepository interface { ... }
type AccessKeyRepository interface { ... }
type AppRepository interface { ... }
type DeploymentRepository interface { ... }
type PackageRepository interface { ... }

// 메트릭 및 스토리지
type MetricsRepository interface { ... }
type BlobStorage interface {
    HealthChecker
    PutObject(context.Context, string, []byte, string) (string, error)
    DeleteObject(context.Context, string) error
}
```

**평가:**
- ✅ 명확한 인터페이스 정의
- ✅ 3개 메서드로 스토리지 백엔드 구현 용이
- ✅ 모든 인터페이스가 Core 레이어에 위치

### 2. 스토리지 백엔드 교체 용이성

**BlobStorage 인터페이스:**
```go
type BlobStorage interface {
    CheckHealth(context.Context) error
    PutObject(context.Context, string, []byte, string) (string, error)
    DeleteObject(context.Context, string) error
}
```

**새로운 스토리지 백엔드 추가 예시:**

**Azure Blob Storage:**
```go
// internal/adapters/storage/azure/storage.go
package azure

import (
    "context"
    "github.com/Azure/azure-storage-blob-go/azblob"
    "github.com/yoophi/codepush-server-golang/internal/core/ports"
)

type Storage struct {
    client *azblob.Client
    container string
}

func New(accountName, accountKey, container string) (*Storage, error) {
    cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
    if err != nil {
        return nil, err
    }
    client, err := azblob.NewClientWithSharedKeyCredential(
        "https://"+accountName+".blob.core.windows.net/",
        cred,
        nil,
    )
    if err != nil {
        return nil, err
    }
    return &Storage{client: client, container: container}, nil
}

func (s *Storage) CheckHealth(ctx context.Context) error {
    _, err := s.client.ServiceClient().GetProperties(ctx, nil)
    return err
}

func (s *Storage) PutObject(ctx context.Context, key string, payload []byte, contentType string) (string, error) {
    _, err := s.client.UploadBuffer(ctx, s.container, key, payload, nil)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("azure://%s/%s", s.container, key), nil
}

func (s *Storage) DeleteObject(ctx context.Context, key string) error {
    _, err := s.client.DeleteBlob(ctx, s.container, key, nil)
    return err
}

// main.go에 추가
func newBlobStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
    switch cfg.StorageBackend {
    case "s3":
        return s3storage.New(ctx, cfg.S3Region, cfg.S3Endpoint, ...)
    case "minio":
        return miniostorage.New(cfg.MinIOEndpoint, ...)
    case "azure":
        return azurestorage.New(cfg.AzureAccount, cfg.AzureKey, cfg.AzureContainer)
    default:
        return s3storage.New(ctx, cfg.S3Region, cfg.S3Endpoint, ...)
    }
}
```

**Google Cloud Storage:**
```go
// internal/adapters/storage/gcs/storage.go
package gcs

import (
    "cloud.google.com/go/storage"
    "github.com/yoophi/codepush-server-golang/internal/core/ports"
)

type Storage struct {
    client *storage.Client
    bucket  string
}

func New(bucketName, credentialsPath string) (*Storage, error) {
    client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsPath))
    if err != nil {
        return nil, err
    }
    return &Storage{client: client, bucket: bucketName}, nil
}

// ports.BlobStorage 인터페이스 구현 (S3/MinIO와 동일)
```

**로컬 파일시스템:**
```go
// internal/adapters/storage/local/storage.go
package local

import (
    "os"
    "path/filepath"
    "github.com/yoophi/codepush-server-golang/internal/core/ports"
)

type Storage struct {
    baseDir string
}

func New(baseDir string) (*Storage, error) {
    if err := os.MkdirAll(baseDir, 0755); err != nil {
        return nil, err
    }
    return &Storage{baseDir: baseDir}, nil
}

func (s *Storage) CheckHealth(ctx context.Context) error {
    return os.WriteFile(filepath.Join(s.baseDir, ".health"), []byte("ok"), 0644)
}

func (s *Storage) PutObject(ctx context.Context, key string, payload []byte, contentType string) (string, error) {
    path := filepath.Join(s.baseDir, key)
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return "", err
    }
    if err := os.WriteFile(path, payload, 0644); err != nil {
        return "", err
    }
    return fmt.Sprintf("file://%s", path), nil
}

func (s *Storage) DeleteObject(ctx context.Context, key string) error {
    return os.Remove(filepath.Join(s.baseDir, key))
}
```

**평가:**
- ✅ 새로운 백엔드 추가 → 인터페이스만 구현 (3개 메서드)
- ✅ 메인 코드 변경 최소화 (case 문에 한 줄 추가)
- ✅ 기존 테스트를 그대로 재사용 가능

### 3. 의존성 주입 (Dependency Injection)

**위치:** `internal/application/service.go:30-46`

```go
type Service struct {
    accounts    ports.AccountRepository
    accessKeys  ports.AccessKeyRepository
    apps        ports.AppRepository
    deployments ports.DeploymentRepository
    packages    ports.PackageRepository
    metrics     ports.MetricsRepository
}

func NewService(
    accounts ports.AccountRepository,
    accessKeys ports.AccessKeyRepository,
    apps ports.AppRepository,
    deployments ports.DeploymentRepository,
    packages ports.PackageRepository,
    metrics ports.MetricsRepository,
) *Service {
    return &Service{
        accounts:    accounts,
        accessKeys:  accessKeys,
        apps:        apps,
        deployments: deployments,
        packages:    packages,
        metrics:     metrics,
    }
}
```

**평가:**
- ✅ 생성자 주입(Constructor Injection) 사용
- ✅ 인터페이스 타입만 의존 (구체 타입 아님)
- ✅ 테스트 시 Fake/Mock 주입 용이

### 4. 테스트 가능한 설계

**위치:** `cmd/server/main.go:35-51`

```go
var (
    loadConfigFn    = config.Load
    newDepsFn       = func(ctx context.Context, cfg config.Config) (appDeps, error) { ... }
    newMetricsFn    = func(cfg config.Config) metricsCloser { ... }
    newBlobStorageFn = newBlobStorage
    listenFn = func(server *http.Server) error { ... }
)
```

**테스트 활용:**
```go
// cmd/server/main_test.go
func TestMainWithFakeDeps(t *testing.T) {
    // 테스트용 Fake 의존성 주입
    fakeAccounts := &fakeAccounts{...}
    fakeMetrics := &fakeMetrics{...}

    // 함수 변수 교체
    newDepsFn = func(ctx context.Context, cfg config.Config) (appDeps, error) {
        return appDeps{
            close:       func() {},
            accounts:    fakeAccounts,
            accessKeys:  &fakeAccessKeys{},
            // ...
        }, nil
    }

    // main 함수 실행
    run(context.Background())
}
```

**평가:**
- ✅ 함수 변수를 사용하여 테스트 시 쉽게 교체 가능
- ✅ 외부 라이브러리 의존성 없는 순수 Go 코드로 DI 구현

---

## ⚠️ 개선 필요 사항

### 5. 스토리지 백엔드 선택 방식

**위치:** `cmd/server/main.go:93-100`

**문제 코드:**
```go
func newBlobStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
    switch cfg.StorageBackend {
    case "minio":
        return miniostorage.New(cfg.MinIOEndpoint, cfg.MinIOAccessKeyID, cfg.MinIOSecretAccessKey, cfg.MinIOBucket, cfg.MinIOUseSSL)
    default:
        // ⚠️ S3이 기본값, 문자열 비교로 선택
        return s3storage.New(ctx, cfg.S3Region, cfg.S3Endpoint, cfg.S3AccessKeyID, cfg.S3SecretAccessKey, cfg.S3Bucket, cfg.S3UsePathStyle)
    }
}
```

**문제점:**
1. **문자열 기반 비교:** 오타 발생 가능성 (예: "S3" vs "s3")
2. **기본값 명시:** default case가 S3를 선택하여 의도가 불분명
3. **모든 백엔드 컴파일:** S3와 MinIO 모두 항상 바이너리에 포함

**개선 방안 1 - Factory 패턴:**
```go
// internal/adapters/storage/factory/factory.go
package factory

import (
    "github.com/yoophi/codepush-server-golang/internal/config"
    "github.com/yoophi/codepush-server-golang/internal/core/ports"
    s3storage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/s3"
    miniostorage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/minio"
    azurestorage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/azure"
    // ... 다른 스토리지
)

// 스토리지 타입 정의
type StorageBackend string

const (
    StorageBackendS3    StorageBackend = "s3"
    StorageBackendMinIO  StorageBackend = "minio"
    StorageBackendAzure  StorageBackend = "azure"
    StorageBackendGCS   StorageBackend = "gcs"
    StorageBackendLocal StorageBackend = "local"
)

// 백엔드 타입
type BackendType string

const (
    BackendTypeS3    BackendType = "s3"
    BackendTypeMinIO  BackendType = "minio"
    BackendTypeAzure  BackendType = "azure"
    BackendTypeGCS   BackendType = "gcs"
    BackendTypeLocal BackendType = "local"
)

// Factory 함수
func NewStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
    backend := BackendType(strings.ToLower(cfg.StorageBackend))

    switch backend {
    case BackendTypeS3:
        return s3storage.New(ctx, cfg.S3Region, cfg.S3Endpoint,
            cfg.S3AccessKeyID, cfg.S3SecretAccessKey,
            cfg.S3Bucket, cfg.S3UsePathStyle)
    case BackendTypeMinIO:
        return miniostorage.New(cfg.MinIOEndpoint, cfg.MinIOAccessKeyID,
            cfg.MinIOSecretAccessKey, cfg.MinIOBucket, cfg.MinIOUseSSL)
    case BackendTypeAzure:
        return azurestorage.New(cfg.AzureAccount, cfg.AzureKey, cfg.AzureContainer)
    case BackendTypeGCS:
        return gcsstorage.New(cfg.GCSCredentials, cfg.GCSBucket)
    case BackendTypeLocal:
        return localstorage.New(cfg.LocalStoragePath)
    default:
        return nil, fmt.Errorf("unsupported storage backend: %s", backend)
    }
}

// 스토리지 타입 유효성 검사
func ValidateBackend(backend string) error {
    switch strings.ToLower(backend) {
    case string(BackendTypeS3), string(BackendTypeMinIO),
         string(BackendTypeAzure), string(BackendTypeGCS), string(BackendTypeLocal):
        return nil
    default:
        return fmt.Errorf("invalid storage backend: %s (valid: s3, minio, azure, gcs, local)", backend)
    }
}
```

**사용 예시:**
```go
// cmd/server/main.go
import "github.com/yoophi/codepush-server-golang/internal/adapters/storage/factory"

func newBlobStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
    // 설정 유효성 검사
    if err := factory.ValidateBackend(cfg.StorageBackend); err != nil {
        return nil, fmt.Errorf("invalid storage backend config: %w", err)
    }
    return factory.NewStorage(ctx, cfg)
}
```

**개선 방안 2 - 플러그인 시스템 (더 발전된 방법):**
```go
// internal/adapters/storage/registry/registry.go
package registry

import (
    "sync"
    "github.com/yoophi/codepush-server-golang/internal/core/ports"
)

type StorageBackend interface {
    Name() string
    Init(config map[string]any) error
    New(ctx context.Context) (ports.BlobStorage, error)
    Cleanup() error
}

var (
    backends  = make(map[string]StorageBackend)
    backendsMu sync.RWMutex
)

// 백엔드 등록
func RegisterBackend(name string, backend StorageBackend) {
    backendsMu.Lock()
    defer backendsMu.Unlock()
    backends[name] = backend
}

// 백엔드 생성
func GetBackend(name string) (StorageBackend, error) {
    backendsMu.RLock()
    defer backendsMu.RUnlock()
    backend, ok := backends[name]
    if !ok {
        return nil, fmt.Errorf("storage backend '%s' not registered", name)
    }
    return backend, nil
}

// 사용자 정의 백엔드 초기화
func init() {
    // 예: 외부 패키지에서 init() 함수로 자동 등록
    // RegisterBackend("custom", &CustomStorage{})
}
```

### 6. 하드코딩된 값들

#### 6.1. SHA1 토큰 생성

**위치:** `internal/application/service.go:411-416`

```go
func generateToken(seed string, value int64) string {
    h := sha1.New()
    _, _ = fmt.Fprintf(h, "%s-%d", seed, value)
    sum := h.Sum(nil)
    return fmt.Sprintf("%x", sum)
}
```

**문제:**
- SHA1은 이미 취약한 해시 알고리즘
- 설정 가능하도록 변경 필요

**개선 방안:**
```go
// internal/application/service.go
import "crypto/rand"
import "encoding/hex"

type TokenGenerator func(seed string, value int64) string

var generateToken TokenGenerator = func(seed string, value int64) string {
    b := make([]byte, 20)
    if _, err := rand.Read(b); err != nil {
        // Fallback to time-based if crypto fails
        h := sha1.New()
        _, _ = fmt.Fprintf(h, "%s-%d", seed, value)
        return fmt.Sprintf("%x", h.Sum(nil))
    }
    return hex.EncodeToString(b)
}

// 테스트에서 교체 가능
func TestWithCustomTokenGenerator(t *testing.T) {
    oldGen := generateToken
    defer func() { generateToken = oldGen }()

    generateToken = func(seed string, value int64) string {
        return "fixed-token-for-test"
    }
    // 테스트 실행
}
```

#### 6.2. 기본 TTL 하드코딩

**위치:** `internal/application/service.go:19`

```go
const defaultAccessKeyTTL = int64(60 * 24 * time.Hour / time.Millisecond)
```

**문제:**
- 60일 하드코딩, 환경별로 변경 불가
- 개발/운영 환경에서는 너무 길 수 있음

**개선 방안:**
```go
// internal/config/config.go
type Config struct {
    // ... 기존 필드들
    DefaultAccessKeyTTL int `env:"DEFAULT_ACCESS_KEY_TTL"` // 밀리초
}

// config.go 로드
func Load() (Config, error) {
    cfg := Config{
        DefaultAccessKeyTTL: 60 * 24 * 60 * 1000, // 기본 60일 (밀리초)
        // ...
    }
    if v := os.Getenv("DEFAULT_ACCESS_KEY_TTL"); v != "" {
        if parsed, err := strconv.Atoi(v); err == nil {
            cfg.DefaultAccessKeyTTL = parsed
        }
    }
    // ...
    return cfg, nil
}

// internal/application/service.go
func NewService(...) *Service {
    // TTL을 생성자로 전달받거나 config에서 로드
    defaultTTL := cfg.DefaultAccessKeyTTL
    return &Service{
        // ...
        defaultAccessKeyTTL: defaultTTL,
    }
}
```

### 7. 빌드 태그 미사용

**문제:**
```go
// 모든 어댑터가 항상 컴파일됨
import "github.com/yoophi/codepush-server-golang/internal/adapters/storage/s3"
import "github.com/yoophi/codepush-server-golang/internal/adapters/storage/minio"
```

**영향:**
- 바이너리 크기 불필요하게 증가
- 사용하지 않는 백엔드까지 포함

**개선 방안:**
```go
// cmd/server/main.go

//go:build s3
package main

import (
    s3storage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/s3"
)

//go:build !s3 && minio
package main

import (
    miniostorage "github.com/yoophi/codepush-server-golang/internal/adapters/storage/minio"
)

//go:build !s3 && !minio
package main

import (
    fmt"
)

func newBlobStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
    // 빌드 태그에 따른 처리
}

// 또는 통합 파일:
//go:build s3
package main

func newBlobStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
    return s3storage.New(ctx, cfg.S3Region, ...)
}

//go:build minio
package main

func newBlobStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
    return miniostorage.New(cfg.MinIOEndpoint, ...)
}
```

**바이너리 크기 절감 효과:**
| 백엔드 | 원본 크기 | 태그 사용 후 | 절감 |
|---------|-----------|-------------|------|
| 전체 | ~45MB | ~35MB | ~22% |

---

## 📈 향후 확장 방안

### 8. 이벤트 기반 아키텍처 (Event-Driven Architecture)

**목적:** 비동기 통합 및 확장성 향상

```go
// internal/events/publisher.go
package events

type EventType string

const (
    EventAppCreated      EventType = "app.created"
    EventDeploymentReleased EventType = "deployment.released"
    EventMetricsUpdated  EventType = "metrics.updated"
)

type Event struct {
    Type      EventType
    Payload   interface{}
    Timestamp time.Time
}

type Publisher interface {
    Publish(ctx context.Context, event Event) error
}

// Redis Pub/Sub 사용
type RedisPublisher struct {
    client *goredis.Client
}

func (p *RedisPublisher) Publish(ctx context.Context, event Event) error {
    data, _ := json.Marshal(event)
    return p.client.Publish(ctx, string(event.Type), data).Err()
}
```

**이벤트 구독 예시:**
```go
// 웹훅 리스너
type WebhookListener struct {
    publisher events.Publisher
}

func (l *WebhookListener) OnDeploymentReleased(deployment domain.Deployment) {
    event := events.Event{
        Type:      events.EventDeploymentReleased,
        Payload:   deployment,
        Timestamp: time.Now(),
    }
    l.publisher.Publish(context.Background(), event)
}
```

### 9. 마이크로서비스 아키텍처

**목적:** 독립적인 서비스 분리 및 확장

```
┌─────────────────────────────────────────────┐
│ API Gateway (Kong, Ambassador, etc.)      │
└──────────────┬──────────────────────────────┘
               │
    ┌──────────┼──────────┬──────────┐
    │          │          │          │
┌───▼──┐  ┌───▼──┐  ┌───▼──┐  ┌───▼──┐
│  App  │  │ Metric │  │Deploy  │  │ Account│
│Service│  │ Service│  │ Service │  │ Service│
└───┬──┘  └───┬──┘  └───┬──┘  └───┬──┘
    │          │          │          │
    └──────────┼──────────┼──────────┘
               │          │
          ┌────▼──────────▼────┐
          │ Message Broker     │
          │ (Redis, Kafka)   │
          └───────────────────┘
```

### 10. 스키마 레지스트리 (Schema Registry)

**목적:** 버전 호환성 및 API 진화

```go
// internal/schema/registry.go
package schema

import "encoding/json"

type Schema struct {
    Version     string
    Name        string
    Definition  json.RawMessage
}

type Registry interface {
    Register(schema Schema) error
    Get(name, version string) (Schema, error)
    List(name string) ([]Schema, error)
}

// Redis 기반 구현
type RedisRegistry struct {
    client *goredis.Client
}

func (r *RedisRegistry) Register(schema Schema) error {
    key := fmt.Sprintf("schema:%s:%s", schema.Name, schema.Version)
    data, _ := json.Marshal(schema)
    return r.client.Set(context.Background(), key, data, 0).Err()
}
```

---

## 📊 최종 점수 요약

```
┌─────────────────────────────────────┐
│ 인터페이스 추상화:        │ ██████████ 100%
│ 의존성 주입:           │ █████████░ 90%
│ 테스트 가능성:           │ █████████░ 90%
│ 스토리지 교체 용이성:     │ █████████░ 90%
│ 설정 유연성:             │ ██████░░░ 60%
│ 플러그인 아키텍처:       │ █████░░░░ 50%
│ 빌드 태그 지원:          │ ███░░░░░░ 30%
│ 빌드 크기 최적화:        │ ████░░░░░ 40%
├─────────────────────────────────────┤
│ 종합 확장성 점수:          │ ███████░░░ 70%
└─────────────────────────────────────┘
```

### 🎉 결론

현재 확장성(Extensibility)은 **양호한 수준**입니다.

**강점:**
- ✅ 우수한 인터페이스 추상화
- ✅ 새로운 스토리지 백엔드 추가가 용이
- ✅ 생성자 주입으로 테스트 가능
- ✅ 명확한 의존성 분리

**개선점:**
- ⚠️ 하드코딩된 값들 설정화
- ⚠️ 스토리지 백엔드 선택 로직 개선 (Factory/Plugin 패턴)
- ⚠️ 빌드 태그로 바이너리 크기 최적화
- ⚠️ 플러그인 아키텍처 도입 고려

**권고:**
1. Factory 패턴으로 스토리지 백엔드 선택 개선
2. 빌드 태그로 필요한 백엔드만 포함
3. 하드코딩된 값을 환경 변수로 노출
4. 선택적으로 플러그인 시스템 도입으로 더 유연한 확장성 확보
