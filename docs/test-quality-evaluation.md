# 테스트 코드 품질 평가

**평가 날짜:** 2026년 3월 18일
**평가자:** AI 아키텍처 분석
**전체 점수:** 8.6/10 ✅

---

## 📊 요약

| 평가 항목 | 점수 | 상태 |
|----------|------|------|
| 전체 커버리지 | 75.8% | ✅ 우수 |
| 테스트 패턴 | 9/10 | ✅ 우수 |
| Mock/Fake 구현 | 10/10 | ✅ 완벽 |
| 테스트 격리 | 10/10 | ✅ 완벽 |
| 테스트 인프라 | 9/10 | ✅ 우수 |
| 테스트 명명 | 10/10 | ✅ 완벽 |
| 커버리지 격차 | 6/10 | ⚠️ 부족 |
| **종합 점수** | **8.6/10** | **✅ 양호** |

---

## 📈 패키지별 커버리지 상세

| 패키지 | 커버리지 | 라인 수 | 테스트 파일 | 평가 |
|--------|----------|---------|-----------|------|
| `internal/adapters/http/middleware` | 100.0% | 37 | `auth_test.go` | ✅ 완벽 |
| `internal/adapters/storage/s3` | 94.7% | 67 | `storage_test.go` | ✅ 우수 |
| `internal/config` | 95.0% | 92 | `config_test.go` | ✅ 우수 |
| `internal/adapters/metrics/redis` | 91.7% | 139 | `metrics_test.go` | ✅ 우수 |
| `cmd/server` | 78.9% | 157 | `main_test.go` | ✅ 양호 |
| `internal/application` | 80.8% | 491 | `service_test.go` | ✅ 양호 |
| `internal/adapters/persistence/postgres` | 74.8% | 765 | `repository_test.go` | ⚠️ 개선 필요 |
| `internal/adapters/http/ginhandler` | 68.6% | 396 | `router_test.go` | ⚠️ 개선 필요 |
| `internal/adapters/storage/minio` | 63.6% | 52 | `storage_test.go` | ⚠️ 개선 필요 |
| `internal/testutil` | 22.0% | 111 | `containers_test.go` | ❌ 부족 |

**평균 커버리지:** 75.8%

---

## ✅ 우수한 테스트 패턴

### 1. 포괄적인 Fake/Mock 구현

**위치:** `internal/application/service_test.go`

```go
// 5개의 완전한 Fake 타입
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
func (f *fakeAccounts) GetByID(context.Context, string) (domain.Account, error) {
    if f.getByIDErr != nil {
        return domain.Account{}, f.getByIDErr
    }
    if f.getByIDAccount.ID != "" {
        return f.getByIDAccount, nil
    }
    return f.account, nil
}
// ... 나머지 메서드 구현

type fakeAccessKeys struct { ... }
type fakeApps struct { ... }
type fakeDeployments struct { ... }
type fakePackages struct { ... }
type fakeMetrics struct { ... }
```

**평가:**
- ✅ 포트 인터페이스를 완전하게 구현
- ✅ 호출 추적을 위한 필드 포함 (예: `resolveAccessKeyArg`)
- ✅ 에러 시뮬레이션 용이
- ✅ 각 메서드마다 독립적인 제어

**사용 예시:**
```go
func TestAuthenticate(t *testing.T) {
    accounts := &fakeAccounts{
        resolveAccountID: "acc-1",
        account: domain.Account{ID: "acc-1", Email: "user@example.com"},
    }

    service := NewService(accounts, &fakeAccessKeys{}, ...)

    account, err := service.Authenticate(context.Background(), "valid-token")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if accounts.resolveAccessKeyArg != "valid-token" {
        t.Fatalf("expected token to be passed")
    }
}
```

### 2. miniredis를 통한 Redis 모킹

**위치:** `internal/adapters/metrics/redis/metrics_test.go:12-21`

```go
func TestMetricsWithMiniRedis(t *testing.T) {
    ctx := context.Background()
    server, err := miniredis.Run()
    if err != nil {
        t.Fatalf("miniredis.Run() error = %v", err)
    }
    defer server.Close()

    metrics := New(server.Addr(), "", 0)
    defer metrics.Close()

    if err := metrics.CheckHealth(ctx); err != nil {
        t.Fatalf("CheckHealth() error = %v", err)
    }
    // ... 테스트 코드
}
```

**평가:**
- ✅ 외부 Redis 서버 없이 테스트 가능
- ✅ in-memory로 빠른 실행
- ✅ 독립된 테스트 격리
- ✅ 정리(defer) 보장

### 3. Fake DB 구현

**위치:** `internal/adapters/persistence/postgres/repository_test.go:16-136`

```go
// PostgreSQL pgx 인터페이스를 완전히 구현
type fakeDB struct {
    beginFn func(context.Context) (pgx.Tx, error)
    execFn  func(context.Context, string, ...any) (pgconn.CommandTag, error)
    queryFn func(context.Context, string, ...any) (pgx.Rows, error)
    rowFn   func(context.Context, string, ...any) pgx.Row
    pingErr error
    closed  bool
}

func (f *fakeDB) Begin(ctx context.Context) (pgx.Tx, error) {
    if f.beginFn != nil {
        return f.beginFn(ctx)
    }
    return nil, errors.New("unexpected Begin")
}

func (f *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
    if f.rowFn != nil {
        return f.rowFn(ctx, sql, args...)
    }
    return fakeRow{err: pgx.ErrNoRows}
}

// ... 나머지 메서드
```

**평가:**
- ✅ `pgx` 인터페이스 완전 구현
- ✅ 테스트에서 원하는 동작 시뮬레이션 가능
- ✅ 실제 DB 없이 테스트
- ✅ 행(Row), 결과 집합(Rows) 모두 모킹

### 4. HTTP 테스트 서버

**위치:** `internal/adapters/http/ginhandler/router_test.go`

```go
import (
    "net/http"
    "net/http/httptest"
    "github.com/gin-gonic/gin"
    "github.com/yoophi/codepush-server-golang/internal/application"
)

func setupTestServer(service *application.Service) (*httptest.Server, *gin.Engine) {
    gin.SetMode(gin.TestMode)
    router := NewRouter(service)
    server := httptest.NewServer(router)
    return server, router
}
```

**평가:**
- ✅ Gin 테스트 모드 사용
- ✅ httptest로 실제 HTTP 요청 시뮬레이션
- ✅ 독립된 서버 인스턴스

### 5. 컨테이너 기반 통합 테스트

**위치:** `internal/testutil/containers.go`

```go
type Stack struct {
    PostgreSQL *testcontainers.PostgreSQLContainer
    Redis      *testcontainers.GenericContainer
    MinIO      *testcontainers.GenericContainer
}

func StartStack(t *testing.T) (*Stack, error) {
    ctx := context.Background()

    // PostgreSQL 컨테이너 시작
    pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "postgres:16",
            ExposedPorts: []string{"5432/tcp"},
            Env: map[string]string{
                "POSTGRES_USER":     "postgres",
                "POSTGRES_PASSWORD": "postgres",
                "POSTGRES_DB":       "testdb",
            },
        },
        Started: true,
    })
    if err != nil {
        return nil, err
    }

    // Redis, MinIO 유사하게 시작

    return &Stack{
        PostgreSQL: pgContainer,
        Redis:      redisContainer,
        MinIO:      minioContainer,
    }, nil
}

func (s *Stack) Close() error {
    var errs []error
    if s.PostgreSQL != nil {
        if err := s.PostgreSQL.Terminate(context.Background()); err != nil {
            errs = append(errs, err)
        }
    }
    // Redis, MinIO 유사하게 종료
    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    return nil
}
```

**평가:**
- ✅ testcontainers-go로 진짜 서버 환경 시뮬레이션
- ✅ 완전한 테스트 격리
- ✅ 정리 메서드로 리소스 해제 보장

---

## 📋 테스트 파일 분석 (14개)

### 유닛 테스트 (Unit Tests) - 9개

| 파일 | 테스트 대상 | 도구 | 테스트 수 |
|------|-------------|------|----------|
| `internal/config/config_test.go` | 설정 로드 | stdlib | ~5개 |
| `internal/application/service_test.go` | Application 서비스 | Fake 구현 | ~20개 |
| `internal/adapters/persistence/postgres/repository_test.go` | PostgreSQL 저장소 | Fake DB | ~15개 |
| `internal/adapters/http/middleware/auth_test.go` | 인증 미들웨어 | Fake | ~3개 |
| `internal/adapters/http/ginhandler/router_test.go` | Gin 라우터 | httptest | ~8개 |
| `internal/adapters/storage/s3/storage_test.go` | S3 스토리지 | Fake AWS client | ~4개 |
| `internal/adapters/storage/minio/storage_test.go` | MinIO 스토리지 | Fake MinIO client | ~4개 |
| `internal/adapters/metrics/redis/metrics_test.go` | Redis 메트릭 | miniredis | ~3개 |
| `internal/testutil/containers_test.go` | 컨테이너 유틸리티 | testcontainers | ~2개 |

**총계:** 유닛 테스트 ~64개

### 통합 테스트 (Integration Tests) - 2개

| 파일 | 테스트 대상 | 의존성 | 테스트 수 |
|------|-------------|----------|----------|
| `tests/integration/postgres_test.go` | PostgreSQL 통합 | Docker, pgx | ~5개 |
| `tests/integration/metrics_storage_test.go` | 메트릭/스토리지 | Docker, Redis, MinIO, S3 | ~3개 |

**총계:** 통합 테스트 ~8개

### E2E 테스트 (End-to-End Tests) - 1개

| 파일 | 테스트 대상 | 의존성 | 테스트 수 |
|------|-------------|----------|----------|
| `tests/e2e/server_test.go` | 전체 서버 | Docker, HTTP client | ~3개 |

**총계:** E2E 테스트 ~3개

### API 스펙 테스트 - 1개

| 파일 | 테스트 대상 | 의존성 | 테스트 수 |
|------|-------------|----------|----------|
| `api/openapi_test.go` | OpenAPI 스펙 | oapi-codegen | ~1개 |

---

## ⚠️ 커버리지 격차 개선 필요

### 6. 낮은 커버리지 패키지

#### 6.1. `internal/adapters/storage/minio` - 63.6%

**미테스트 코드 분석:**
```go
// internal/adapters/storage/minio/storage.go
type Storage struct {
    client client
    bucket string
}

func (s *Storage) CheckHealth(ctx context.Context) error {
    _, err := s.client.BucketExists(ctx, s.bucket)
    return err
}

func (s *Storage) PutObject(ctx context.Context, key string, payload []byte, contentType string) (string, error) {
    _, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(payload), int64(len(payload)), minio.PutObjectOptions{
        ContentType: contentType,
    })
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("minio://%s/%s", s.bucket, key), nil
}

func (s *Storage) DeleteObject(ctx context.Context, key string) error {
    return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
```

**추가 필요한 테스트:**
```go
// internal/adapters/storage/minio/storage_test.go
func TestMinioStorageBucketNotFound(t *testing.T) {
    // Bucket이 존재하지 않을 때
    fakeClient := &fakeClient{
        bucketExistsErr: errors.New("bucket not found"),
    }
    storage := &Storage{client: fakeClient, bucket: "test-bucket"}

    err := storage.CheckHealth(context.Background())
    if err == nil {
        t.Fatal("expected error for non-existent bucket")
    }
}

func TestMinioStorageUploadFailure(t *testing.T) {
    // 업로드 실패 시나리오
    fakeClient := &fakeClient{
        putObjectErr: errors.New("network error"),
    }
    storage := &Storage{client: fakeClient, bucket: "test-bucket"}

    _, err := storage.PutObject(context.Background(), "key", []byte("data"), "application/json")
    if err == nil {
        t.Fatal("expected upload error")
    }
}

func TestMinioStorageDeleteNonExistent(t *testing.T) {
    // 존재하지 않는 객체 삭제
    fakeClient := &fakeClient{
        removeObjectErr: errors.New("object not found"),
    }
    storage := &Storage{client: fakeClient, bucket: "test-bucket"}

    err := storage.DeleteObject(context.Background(), "nonexistent")
    if err == nil {
        t.Fatal("expected error for non-existent object")
    }
}

func TestMinioStorageConcurrentOperations(t *testing.T) {
    // 동시 작업 테스트
    storage := setupTestStorage(t)

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            key := fmt.Sprintf("key-%d", i)
            _, err := storage.PutObject(context.Background(), key, []byte("data"), "application/json")
            if err != nil {
                t.Errorf("upload failed: %v", err)
            }
        }(i)
    }
    wg.Wait()
}

// Fake client 구현
type fakeClient struct {
    bucketExistsErr error
    putObjectErr      error
    removeObjectErr  error
}

func (f *fakeClient) BucketExists(ctx context.Context, bucket string) (bool, error) {
    return f.bucketExistsErr == nil, f.bucketExistsErr
}

func (f *fakeClient) PutObject(...) error { return f.putObjectErr }
func (f *fakeClient) RemoveObject(...) error { return f.removeObjectErr }
```

#### 6.2. `internal/adapters/http/ginhandler` - 68.6%

**미테스트 경로 분석:**
```go
// internal/adapters/http/ginhandler/router.go
// 총 396줄
// 핸들러 함수: ~30개
// 현재 테스트: ~8개
```

**추가 필요한 테스트:**
```go
// internal/adapters/http/ginhandler/router_test.go

// 1. 인증되지 않은 요청 처리
func TestHealthEndpoint(t *testing.T) {
    server, router := setupTestServer(fakeService)
    defer server.Close()

    resp, err := http.Get(server.URL + "/health")
    if err != nil {
        t.Fatal(err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected 200, got %d", resp.StatusCode)
    }
}

// 2. OAuth 플레이스홀더 엔드포인트
func TestOAuthEndpointsNotImplemented(t *testing.T) {
    server, router := setupTestServer(fakeService)
    defer server.Close()

    endpoints := []string{
        "/auth/login/github",
        "/auth/login/microsoft",
        "/auth/callback/github",
    }

    for _, endpoint := range endpoints {
        resp, err := http.Post(server.URL + endpoint, "application/json", nil)
        if err != nil {
            t.Fatal(err)
        }
        if resp.StatusCode != http.StatusNotImplemented {
            t.Fatalf("expected 501 for %s", endpoint)
        }
    }
}

// 3. 버전 2개의 updateCheck 엔드포인트
func TestV01UpdateCheckEndpoint(t *testing.T) {
    server, router := setupTestServer(fakeService)
    defer server.Close()

    req, _ := http.NewRequest("GET", server.URL+"/v0.1/public/codepush/update_check", nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Fatal(err)
    }
    // 검증 로직
}

// 4. 에러 응답 형식
func TestErrorResponseFormat(t *testing.T) {
    // 잘못된 인증 정보로 요청
    // 응답 형식이 {error: "..."} 인지 확인
}

// 5. 쿼리 파라미터 처리
func TestQueryParameterVariants(t *testing.T) {
    // appVersion vs app_version, packageHash vs package_hash 등
}

// 6. 경로 파라미터 처리
func TestPathParameterValidation(t *testing.T) {
    // appName, deploymentName 등의 유효성 검사
}
```

#### 6.3. `internal/testutil` - 22.0%

**현재 테스트:**
```go
// internal/testutil/containers_test.go
func TestStartStack(t *testing.T) {
    // 컨테이너 시작/종료 테스트
}
```

**추가 필요한 테스트:**
```go
func TestStartStackPanicRecovery(t *testing.T) {
    // Panic 발생 시 정리가 호출되는지 테스트
    defer func() {
        if r := recover(); r != nil {
            t.Log("panic recovered:", r)
        }
    }()
}

func TestStackConnectionStrings(t *testing.T) {
    // 올바른 연결 문자열이 생성되는지 테스트
    stack, err := StartStack(t)
    if err != nil {
        t.Fatal(err)
    }
    defer stack.Close()

    // PostgreSQL 연결 문자열 검증
    pgURI, _ := stack.PostgreSQL.ConnectionString(context.Background())
    if !strings.Contains(pgURI, "sslmode=disable") {
        t.Errorf("expected sslmode=disable in connection string")
    }
}

func TestStackCloseAllContainers(t *testing.T) {
    // 모든 컨테이너가 종료되는지 테스트
    stack, err := StartStack(t)
    if err != nil {
        t.Fatal(err)
    }

    // 첫 번째 종료
    err = stack.PostgreSQL.Terminate(context.Background())
    if err != nil {
        t.Fatal(err)
    }

    // 전체 스택 종료 (실패해야 함)
    err = stack.Close()
    if err == nil {
        t.Error("expected error when closing already terminated container")
    }
}
```

---

## 🎯 테스트 명명 및 구조

### 테스트 명명 규칙 준수

| 규칙 | 준수 여부 | 예시 |
|--------|-----------|------|
| `TestXxx` 형식 | ✅ | `TestListAccessKeysMasksAndSorts` |
| `TestXxxYyy` (서브테스트) | ✅ | `TestRollbackScenarios/not found when history empty` |
| `BenchmarkXxx` | ⚠️ 없음 | 추가 권장 |
| `ExampleXxx` | ⚠️ 없음 | 추가 권장 |

### 서브테스트(t.Run) 활용

**우수한 예시:**
```go
// internal/application/service_test.go
func TestRollbackScenarios(t *testing.T) {
    t.Run("not found when history empty", func(t *testing.T) { ... })
    t.Run("not found when only latest exists and no target label", func(t *testing.T) { ... })
    t.Run("not found when target label missing", func(t *testing.T) { ... })
    t.Run("conflict when app version differs", func(t *testing.T) { ... })
    t.Run("success clones target package as rollback", func(t *testing.T) { ... })
}
```

**개선 권고:**
```go
// 테이블 기반 테스트로 더 구조화
func TestHTTPStatus(t *testing.T) {
    tests := []struct {
        name     string
        err      error
        expected int
    }{
        {"ok", nil, 200},
        {"unauthorized", domain.ErrUnauthorized, 401},
        {"expired", domain.ErrExpired, 401},
        {"forbidden", domain.ErrForbidden, 403},
        {"not found", domain.ErrNotFound, 404},
        {"conflict", domain.ErrConflict, 409},
        {"malformed", domain.ErrMalformedRequest, 400},
        {"internal", errors.New("boom"), 500},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := HTTPStatus(tt.err)
            if got != tt.expected {
                t.Errorf("HTTPStatus(%v) = %d, want %d", tt.err, got, tt.expected)
            }
        })
    }
}
```

---

## 🔧 테스트 인프라

### Makefile 설정

**위치:** `Makefile:18-34`

```makefile
test: unit integration e2e

unit:
    go test ./...

integration:
    go test -tags=integration ./tests/integration/...

e2e:
    go test -tags=e2e ./tests/e2e/...

coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
```

**평가:**
- ✅ 빌드 태그로 테스트 유형 분리
- ✅ 커버리지 리포트 생성
- ✅ 단일 명령으로 전체 테스트 실행

### 빌드 태그 사용

```go
// tests/integration/postgres_test.go
//go:build integration
package integration

// tests/e2e/server_test.go
//go:build e2e
package e2e
```

**평가:**
- ✅ 통합 테스트: Docker 필요 시에만 실행
- ✅ E2E 테스트: 전체 스택 필요 시에만 실행
- ✅ 유닛 테스트: 태그 없이 항상 실행

---

## 💡 추가 테스트 전략

### 7. 속성 기반 테스트 (Property-Based Testing)

**도구:** `github.com/leanovate/gomock` 또는 `github.com/stretchr/testify`

```go
// internal/application/rollback_property_test.go
//go:build !integration
package application

import (
    "testing"
    "quick"
    "github.com/yoophi/codepush-server-golang/internal/core/domain"
)

func TestRollbackProperties(t *testing.T) {
    property := prop.ForAll(
        // Package 생성
        genPackage(),
        // App version
        genVersion(),
        // Rollback 대상
        genRollbackTarget(),
    )

    property.Check(t, rollbackProperty)
}

func rollbackProperty(
    pkg domain.Package,
    appVersion string,
    target string,
) bool {
    // 롤백이 항상 같은 앱 버전에서만 가능하다는 속성 검증
    return pkg.AppVersion == appVersion || target != ""
}
```

### 8. 부하 테스트 (Load Testing)

**도구:** `ghz` 또는 `vegeta`

```bash
# api/benchmark/update_check.sh
#!/bin/bash

ghz -n 1000 -c 10 \
    -m POST \
    -H "Authorization: Bearer test-token" \
    -d '{"deploymentKey":"test-key","appVersion":"1.0.0"}' \
    http://localhost:3000/updateCheck

# 결과:
# Requests      [total, rate, throughput]
# Latency      [min, mean, p50, p95, p99, max]
# Latency Distribution [50%, 75%, 95%, 99%]
# Status codes  [code:count, code:count]
```

### 9. 퍼지 테스트 (Fuzz Testing)

**도구:** Go 1.18+ 내장 `testing/fuzz`

```go
// internal/application/version_match_fuzz.go
//go:build !integration
package application

import (
    "context"
    "testing"
)

func FuzzVersionMatching(f *testing.F) {
    target := "1.0.0"
    current := "1.2.3"

    f.Add(target)
    f.Add(current)

    f.Fuzz(func(t *testing.T, targetInput, currentInput []byte) {
        targetStr := string(targetInput)
        currentStr := string(currentInput)

        result := matchesVersion(targetStr, currentStr)
        // 충돌 감지 등
        _ = result
    })
}
```

### 10. 병행 테스트

**도구:** Go 1.7+ `t.Parallel()`

```go
// internal/adapters/metrics/redis/metrics_test.go
func TestMetricsConcurrency(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping concurrency test in short mode")
    }

    ctx := context.Background()
    server, err := miniredis.Run()
    if err != nil {
        t.Fatalf("miniredis.Run() error = %v", err)
    }
    defer server.Close()

    metrics := New(server.Addr(), "", 0)
    defer metrics.Close()

    // 병행 테스트
    t.Parallel()

    // 10개의 고루틴에서 동시 요청
    for i := 0; i < 10; i++ {
        i := i
        t.Run(fmt.Sprintf("concurrent-%d", i), func(t *testing.T) {
            t.Parallel()
            report := domain.DeploymentStatusReport{
                DeploymentKey:  fmt.Sprintf("dep-%d", i),
                ClientUniqueID: fmt.Sprintf("client-%d", i),
                Label:          fmt.Sprintf("v%d", i),
                Status:         deploySucceeded,
            }
            if err := metrics.ReportDeploy(ctx, report); err != nil {
                t.Errorf("ReportDeploy() error = %v", err)
            }
        })
    }
}
```

---

## 📋 CI/CD 통합 권고안

### GitHub Actions 예시

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  unit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.26'
      - name: Run unit tests
        run: make unit
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out

  integration:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.26'
      - name: Run integration tests
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable
          REDIS_ADDR: localhost:6379
        run: make integration

  e2e:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        storage: [s3, minio]
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: testdb
      redis:
        image: redis:7
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.26'
      - name: Run E2E tests
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable
          REDIS_ADDR: localhost:6379
          STORAGE_BACKEND: ${{ matrix.storage }}
        run: make e2e
```

---

## 🎯 우선순위 개선 권고안

### P0 - 즉시 개선

1. **testutil 커버리지 향상**
   - `StartStack` 실패/정리 경로 보강
   - endpoint 생성 보조 로직 분리 후 단위 테스트 가능하게 조정

2. **MinIO/Gin 남은 경계 케이스 보강**
   - MinIO 동시 작업/추가 실패 시나리오
   - Gin handler의 세부 status mapping과 malformed input 케이스 추가

### P1 - 중요 개선

3. **테이블 기반 테스트 확장**
   - 버전 매칭 로직
   - 스토리지 backend validation
   - Redis 옵션 파싱

4. **통합 테스트의 계약 검증 보강**
   - batched query 결과 구조 확인
   - graceful shutdown 관련 smoke 경로 보강

### P2 - 선택적 개선

5. **벤치마크 테스트 추가**
   - updateCheck 엔드포인트
   - metrics 엔드포인트
   - 주요 API 성능 기준 마련

6. **퍼지 테스트 도입**
   - 문자열 파싱
   - 버전 비교
   - 토큰 생성

7. **부하 테스트 설정**
   - ghz 또는 vegeta 설정
   - CI/CD 통합

---

## 📊 최종 점수 요약

```
┌───────────────────────────────────────┐
│ 전체 커버리지:            │ ███████░░░ 77%
│ 테스트 패턴:              │ █████████░ 90%
│ Mock/Fake 구현:           │ ██████████ 100%
│ 테스트 격리:              │ ██████████ 100%
│ 테스트 인프라:            │ █████████░ 90%
│ 테스트 명명:              │ ██████████ 100%
│ 커버리지 격차:            │ ██████░░░░ 60%
├───────────────────────────────────────┤
│ 종합 테스트 점수:           │ ████████░░ 85%
└───────────────────────────────────────┘
```

### 🎉 결론

현재 테스트 코드 품질은 **우수한 수준**입니다.

**강점:**
- ✅ 우수한 Mock/Fake 구현
- ✅ 완벽한 테스트 격리 (miniredis, testcontainers)
- ✅ 빌드 태그로 명확한 분리
- ✅ 높은 커버리지 (77%)
- ✅ 명확한 테스트 명명 규칙

**개선점:**
- ⚠️ 일부 어댑터의 커버리지 격차
- ⚠️ testutil 커버리지 부족
- ⚠️ 일부 테스트 패턴 부분적 사용

**권고:**
1. 낮은 커버리지 패키지에 테스트 추가 (MinIO, Gin)
2. 테이블 기반 테스트로 구조화
3. 벤치마크 및 부하 테스트 도입
4. CI/CD 파이프라인 완성
