# 헥사고날 아키텍처 준수 평가

**평가 날짜:** 2026년 3월 18일
**평가자:** AI 아키텍처 분석
**전체 점수:** 9.2/10 ✅

---

## 📊 요약

| 평가 항목 | 점수 | 상태 |
|----------|------|------|
| 포트 정의 | 9/10 | ✅ 우수 |
| 의존성 방향 | 9/10 | ✅ 우수 |
| 도메인 순수성 | 10/10 | ✅ 완벽 |
| 어댑터 구현 | 9/10 | ✅ 우수 |
| 계층 분리 | 9/10 | ✅ 우수 |
| Composition Root | 9/10 | ✅ 우수 |
| Driving Ports | 9/10 | ✅ 우수 |
| **종합 점수** | **9.2/10** | **✅ 우수** |

---

## ✅ 잘 구현된 부분

### 1. 포트 (Ports) - `internal/core/ports/ports.go`, `internal/core/ports/usecases.go`

```go
type HealthChecker interface {
    CheckHealth(context.Context) error
}

type AccountRepository interface {
    HealthChecker
    EnsureBootstrap(context.Context, domain.Account, domain.AccessKey) error
    GetByID(context.Context, string) (domain.Account, error)
    GetByEmail(context.Context, string) (domain.Account, error)
    ResolveAccountIDByAccessKey(context.Context, string) (string, error)
}

type AccessKeyRepository interface {
    List(context.Context, string) ([]domain.AccessKey, error)
    Create(context.Context, domain.AccessKey, string) (domain.AccessKey, error)
    GetByName(context.Context, string, string) (domain.AccessKey, error)
    Update(context.Context, domain.AccessKey) (domain.AccessKey, error)
    Delete(context.Context, string, string) error
    DeleteSessionsByCreator(context.Context, string, string) (int64, error)
}

type AppRepository interface {
    List(context.Context, string) ([]domain.App, error)
    Create(context.Context, string, domain.App) (domain.App, error)
    GetByName(context.Context, string, string) (domain.App, error)
    Update(context.Context, string, domain.App) (domain.App, error)
    Delete(context.Context, string, string) error
    Transfer(context.Context, string, string, string) error
    AddCollaborator(context.Context, string, string, string) error
    ListCollaborators(context.Context, string, string) (map[string]domain.CollaboratorProperties, error)
    RemoveCollaborator(context.Context, string, string, string) error
}

type DeploymentRepository interface {
    List(context.Context, string, string) ([]domain.Deployment, error)
    Create(context.Context, string, string, domain.Deployment) (domain.Deployment, error)
    GetByName(context.Context, string, string, string) (domain.Deployment, error)
    GetByKey(context.Context, string) (domain.Deployment, error)
    Update(context.Context, string, string, domain.Deployment) (domain.Deployment, error)
    Delete(context.Context, string, string, string) error
}

type PackageRepository interface {
    ListHistory(context.Context, string, string, string) ([]domain.Package, error)
    ClearHistory(context.Context, string, string, string) error
    CommitRollback(context.Context, string, string, string, domain.Package) (domain.Package, error)
}

type MetricsRepository interface {
    HealthChecker
    IncrementDownload(context.Context, string, string) error
    ReportDeploy(context.Context, domain.DeploymentStatusReport) error
    GetMetrics(context.Context, string) (map[string]domain.UpdateMetrics, error)
    Clear(context.Context, string) error
}

type BlobStorage interface {
    HealthChecker
    PutObject(context.Context, string, []byte, string) (string, error)
    DeleteObject(context.Context, string) error
}

type AuthService interface {
    Authenticate(context.Context, string) (domain.Account, error)
}

type HTTPAPI interface {
    AuthService
    Health(context.Context) error
    // ... HTTP adapter가 사용하는 use-case 메서드 집합
}
```

**평가:**
- ✅ 7개의 저장소 인터페이스 명확하게 정의됨
- ✅ `HealthChecker` 인터페이스가 임베딩되어 모든 어댑터에 헬스체크 요구
- ✅ 인터페이스가 Core 레이어에 위치 (외부 의존성 없음)
- ✅ Driving Port(`AuthService`, `HTTPAPI`)가 Core에 정의됨

### 2. 의존성 방향 확인

**의존성 그래프:**

```
                    +------------------+
                    |   cmd/server     |  ← Composition Root
                    |   (main.go)      |
                    +--------+---------+
                             |
                             v
+------------------+  의존함  +--------------------+
|     HTTP         |----------->|   Driving Ports    |
|   (Gin Router)   |              |  (HTTPAPI/Auth)   |
+------------------+              +--------+-----------+
                                          ^
                                          | 구현
                                          v
                               +------------------------+
                               |   Application          |
                               |    Service             |
                               +----------+-------------+
                                          ^
                                          | 의존함
                                          v
                               +------------------------+
                               |    Core (Pure)        |
                               | +--------------------+|
                               | | ports/             ||
                               | | - Interfaces       ||
                               | +--------------------+|
                               | | domain/            ||
                               | | - Models           ||
                               | | - Errors           ||
                               | +--------------------+|
                               +----------+-----------+
                                          ^
           구현함              의존함
+------------------+  +------------------+-------------+
|   Adapters       |  |   Adapters       |   Adapters  |
| (http)           |  | (persistence)    |  (metrics)  |
| - Gin Router     |  | - PostgreSQL     |  - Redis    |
+------------------+  +------------------+  +-----------+
|   Adapters       |                                  |
| (storage)        |                                  |
| - S3             |----------------------------------+
| - MinIO          |
+------------------+
```

**평가:**
- ✅ Core 도메인이 외부 프레임워크에 의존하지 않음
- ✅ 어댑터가 Core 포트에 의존 (Core는 어댑터를 알지 못함)
- ✅ Application 레이어가 인터페이스에만 의존
- ✅ 의존성 역전(Dependency Inversion) 완벽히 준수

### 3. 도메인 모델 순수성

**위치:** `internal/core/domain/models.go`

**도메인 엔티티:**
- `Account` - 계정 정보
- `AccessKey` - 액세스 키
- `App` - 애플리케이션
- `CollaboratorProperties` - 협업자 속성
- `Deployment` - 배포 정보
- `Package` - 패키지 정보
- `UpdateMetrics` - 업데이트 메트릭
- `UpdateCheckRequest/Response` - 업데이트 확인 요청/응답
- `DeploymentStatusReport` - 배포 상태 보고
- `DownloadReport` - 다운로드 보고

**평가:**
- ✅ 모든 도메인 모델이 `internal/core/domain/`에 위치
- ✅ 외부 라이브러리 의존성 없음 (stdlib만 사용)
- ✅ 프레임워크(Gin, pgx, redis) 의존성 없음
- ✅ 도메인 로직 비즈니스 규칙 포함

### 4. 어댑터 구현

| 어댑터 | 위치 | 구현 포트 |
|--------|------|----------|
| `postgres/repository.go` | `internal/adapters/persistence/postgres/` | `AccountRepository`, `AccessKeyRepository`, `AppRepository`, `DeploymentRepository`, `PackageRepository` |
| `redis/metrics.go` | `internal/adapters/metrics/redis/` | `MetricsRepository` |
| `s3/storage.go` | `internal/adapters/storage/s3/` | `BlobStorage` |
| `minio/storage.go` | `internal/adapters/storage/minio/` | `BlobStorage` |
| `ginhandler/router.go` | `internal/adapters/http/ginhandler/` | HTTP 핸들러 (inbound adapter) |
| `middleware/auth.go` | `internal/adapters/http/middleware/` | 인증 미들웨어 |

**평가:**
- ✅ 모든 어댑터가 Core 포트를 정확히 구현
- ✅ 포트 인터페이스만 의존하여 교체 가능

### 5. Application 서비스 레이어

**위치:** `internal/application/service.go`

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
- ✅ 비즈니스 로직이 이 레이어에 집중

### 6. Composition Root

**위치:** `cmd/server/main.go`

```go
func newService(deps appDeps, metrics ports.MetricsRepository) *application.Service {
    return application.NewService(
        deps.accounts,
        deps.accessKeys,
        deps.apps,
        deps.deployments,
        deps.packages,
        metrics,
    )
}

func newBlobStorage(ctx context.Context, cfg config.Config) (ports.BlobStorage, error) {
    switch cfg.StorageBackend {
    case "minio":
        return miniostorage.New(...)
    default:
        return s3storage.New(...)
    }
}
```

**평가:**
- ✅ 의존성 주입이 명시적
- ✅ 함수 변수(`loadConfigFn`, `newDepsFn`, `newMetricsFn`, `newBlobStorageFn`, `listenFn`) 사용으로 테스트 용이
- ✅ 모든 어댑터가 한 곳에서 조립
- ✅ `newHTTPServer(port int, service ports.HTTPAPI)`로 inbound 포트 중심 wiring

---

## ⚠️ 남은 개선 포인트

### 1. BlobStorage 포트의 사용 범위가 아직 제한적

**현재 상태:**
```go
// internal/core/ports/ports.go
type BlobStorage interface {
    HealthChecker
    PutObject(context.Context, string, []byte, string) (string, error)
    DeleteObject(context.Context, string) error
}

// 현재는 cmd/server 에서 조립 및 health/wiring 용도로 사용
```

**영향:**
- 포트 자체는 정당하지만, 현재 Application 유스케이스에 직접 연결되지는 않음
- release/upload 기능이 들어오면 자연스럽게 Application으로 편입될 예정

**권고:**
- release/upload 유스케이스 구현 시 `HTTPAPI` 표면에 업로드 관련 use-case를 추가
- 현재는 주석으로 역할을 명시한 상태 유지

### 2. `main()` 자체는 여전히 테스트 대상이 아니라 진입점으로 유지

**현재 상태:**
```go
// cmd/server/main.go
func main() {
    if err := run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

**영향:**
- `run()`은 테스트 가능하게 분리되었지만 `main()` 자체는 프로세스 진입점 특성상 직접 unit test 하지 않음
- 이는 일반적인 Go 애플리케이션 관례와도 부합

**권고:**
- 현재 구조 유지
- process lifecycle 훅이 더 늘어나면 별도 bootstrap 패키지로 분리 검토

### 3. 저장소 포트가 여전히 넓은 편

**문제:**
- `AccountRepository`, `AppRepository` 등 일부 포트가 CRUD와 권한/협업 규칙을 넓게 포괄
- 기능이 계속 늘어나면 인터페이스 비대화 가능성 존재

**영향:**
- 테스트는 쉽지만, 장기적으로는 읽기/쓰기 또는 aggregate별 세분화가 더 유리할 수 있음

**권고:**
```go
// 예시: 읽기/쓰기 포트 분리 또는 aggregate별 분리
type AppReader interface { ... }
type AppWriter interface { ... }
```

---

## 🎯 계층별 책임 분석

| 레이어 | 위치 | 책임 | 의존성 |
|--------|------|--------|---------|
| **Core/Domain** | `internal/core/domain/` | 순수 도메인 모델, 비즈니스 규칙 | 외부 의존성 없음 |
| **Core/Ports** | `internal/core/ports/` | 인터페이스 정의 (inbound & outbound) | stdlib만 |
| **Application** | `internal/application/` | 유스케이스, 비즈니스 오케스트레이션 | Ports 인터페이스 |
| **Adapters/Inbound** | `internal/adapters/http/` | HTTP 핸들러 (Gin) | Driving Ports (`HTTPAPI`, `AuthService`) |
| **Adapters/Outbound** | `internal/adapters/persistence/`, `metrics/`, `storage/` | PostgreSQL, Redis, S3/MinIO 구현 | Ports 인터페이스 |

---

## 📋 규칙 준수 체크리스트

| 규칙 | 준수 여부 | 비고 |
|--------|-----------|------|
| Core 도메인 프레임워크 독립성 | ✅ | Gin, pgx, redis 의존 없음 |
| 포트가 인터페이스 정의 | ✅ | 모든 포트가 인터페이스로 정의됨 |
| 어댑터가 포트 구현 | ✅ | 모든 어댑터가 포트를 구현 |
| 의존성이 내부를 향함 | ✅ | Core → Application → Adapters 방향 |
| Core가 어댑터를 알지 못함 | ✅ | Core 패키지가 어댑터를 import하지 않음 |
| Composition Root가 분리됨 | ✅ | cmd/server에서 모든 의존성 조립 |
| 생성자 주입 사용 | ✅ | 모든 의존성이 생성자로 주입됨 |
| Driving Port 정의 | ✅ | `AuthService`, `HTTPAPI` 추가됨 |

---

## 💡 개선 권고안

### 우선순위 1 (높음)

1. **BlobStorage를 release/upload use-case와 연결**
   - artifact 업로드 기능이 구현되면 `HTTPAPI`에 명시적으로 편입

### 우선순위 2 (중간)

2. **Repository 포트 세분화 검토**
   - aggregate별 읽기/쓰기 포트 분리 고려

3. **bootstrap 전용 패키지 분리 검토**
   - `cmd/server`의 lifecycle/wiring이 더 복잡해질 경우

---

## 🎉 결론

이 프로젝트는 **헥사고날 아키텍처 원칙을 잘 준수**하고 있습니다.

**강점:**
- ✅ Core 도메인이 프레임워크 독립적임
- ✅ 포트 인터페이스가 명확하게 정의됨
- ✅ 어댑터가 포트를 잘 구현함
- ✅ 의존성 방향이 올바름
- ✅ Composition Root가 명확함
- ✅ Driving Ports가 추가되어 inbound 방향 의존성도 명시적임
- ✅ HTTP 관심사가 Application에서 제거됨

**남은 개선점:**
- ⚠️ BlobStorage를 실제 release/upload 유스케이스에 연결
- ⚠️ Repository 포트 세분화 여부 검토

**최종 평가:** 아키텍처가 매우 건실하고 헥사고날 원칙을 높은 수준으로 준수하고 있어 유지보수와 확장성이 좋습니다.
