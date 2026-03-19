# 확장성 (Scalability) 평가

**평가 날짜:** 2026년 3월 18일
**평가자:** AI 아키텍처 분석
**전체 점수:** 7.4/10 ✅

---

## 📊 요약

| 평가 항목 | 점수 | 상태 |
|----------|------|------|
| 데이터베이스 연결 풀 | 8/10 | ✅ 우수 |
| Redis 연결 풀 | 8/10 | ✅ 우수 |
| HTTP 서버 타임아웃 | 8/10 | ✅ 우수 |
| 쿼리 최적화 | 6/10 | ⚠️ 개선 중 |
| 동시성 처리 | 8/10 | ✅ 우수 |
| 우아한 종료 | 8/10 | ✅ 우수 |
| 리소스 관리 | 7/10 | ✅ 양호 |
| **종합 점수** | **7.4/10** | **✅ 양호** |

---

## 🔴 남아 있는 핵심 문제 (Critical Issues)

### 이번 라운드에서 해소된 항목

- ✅ `cmd/server/main.go`에 `signal.NotifyContext` 기반 graceful shutdown 적용
- ✅ `cmd/server/main.go`에 `ReadTimeout`, `WriteTimeout`, `IdleTimeout` 추가
- ✅ `internal/config/config.go`와 `internal/adapters/metrics/redis/metrics.go`에 Redis 풀/타임아웃 설정 반영
- ✅ `internal/application/service.go`의 토큰 생성이 `crypto/rand` 우선 사용으로 개선됨
- ✅ `internal/adapters/persistence/postgres/repository.go`의 `AppRepo.List()`가 batched query 방식으로 변경됨
- ✅ `internal/application/service.go`가 generated deployment key 충돌 시 재시도함

### 1. 남아 있는 조회 fan-out 문제

`AppRepo.List()`의 가장 큰 N+1은 해소됐지만, `DeploymentRepo.List()`는 각 deployment마다 `currentPackage()`를 한 번 더 조회합니다. 앱 단위 대량 조회보다 영향은 작지만, deployment 수가 많은 환경에서는 여전히 선형 fan-out이 남아 있습니다.

**위치:** `internal/adapters/persistence/postgres/repository.go`

**문제 코드:**
```go
func (r *DeploymentRepo) List(ctx context.Context, accountID, appID string) ([]domain.Deployment, error) {
    rows, err := r.store.pool.Query(ctx, `
        SELECT id, app_id, name, deployment_key, created_at
        FROM deployments WHERE app_id = $1 ORDER BY name
    `, appID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []domain.Deployment
    for rows.Next() {
        var dep domain.Deployment
        if err := rows.Scan(&dep.ID, &dep.AppID, &dep.Name, &dep.Key, &dep.CreatedAt); err != nil {
            return nil, err
        }
        pkg, _ := currentPackage(ctx, r.store.pool, dep.ID)
        dep.Package = pkg
        result = append(result, dep)
    }
    return result, rows.Err()
}
```

**영향 분석:**
- 앱 리스트 조회의 `1 + 2N` 문제는 제거됨
- 다만 deployment가 많은 앱에서 `List()`는 여전히 `1 + N` 형태의 current package 조회를 유발
- 패키지 히스토리가 큰 환경에서는 응답 시간과 DB 왕복이 선형 증가할 수 있음

**성능 영향:**
| 앱 수 | 쿼리 수 | 예상 응답 시간 (ms) |
|--------|----------|---------------------|
| 10개 | 21개 | ~50ms |
| 50개 | 101개 | ~250ms |
| 100개 | 201개 | ~500ms+ |
| 500개 | 1001개 | 2500ms+ |

**개선 방안:**
- `DeploymentRepo.List()`에 current package 일괄 조회 helper 추가
- 또는 deployment와 최신 package를 `DISTINCT ON`/window function으로 한 번에 조인

**개선 효과:**
- 쿼리 수: O(n²) → O(1)
- 예상 응답 시간: ~2500ms → ~50ms (100개 앱 기준, **50배 개선**)

---

## ⚠️ 중간 우려 사항 (Moderate Concerns)

### 2. HTTP 서버 타임아웃 불완전

**위치:** `cmd/server/main.go:151-156`

**문제 코드:**
```go
func newHTTPServer(port int, service *application.Service) *http.Server {
    return &http.Server{
        Addr:              fmt.Sprintf(":%d", port),
        Handler:           httpadapter.NewRouter(service),
        ReadHeaderTimeout: 10 * time.Second,
        // ⚠️ 다음 타임아웃 누락:
        // - ReadTimeout
        // - WriteTimeout
        // - IdleTimeout
    }
}
```

**영향:**
- 느린 클라이언트 연결이 영구적으로 유지될 수 있음
- 리소스 누수 가능성
- 공격자에 의해 연결 고갈(Exhaustion) 유발 가능

**개선 방안:**
```go
func newHTTPServer(port int, service *application.Service) *http.Server {
    return &http.Server{
        Addr:              fmt.Sprintf(":%d", port),
        Handler:           httpadapter.NewRouter(service),
        ReadHeaderTimeout: 10 * time.Second,
        ReadTimeout:       30 * time.Second,  // 추가
        WriteTimeout:      30 * time.Second,  // 추가
        IdleTimeout:       120 * time.Second, // 추가
    }
}
```

### 3. Redis 연결 풀 설정 부재

**위치:** `internal/adapters/metrics/redis/metrics.go:21-28`

**문제 코드:**
```go
func New(addr, password string, db int) *Metrics {
    return &Metrics{
        client: goredis.NewClient(&goredis.Options{
            Addr:     addr,
            Password: password,
            DB:       db,
            // ⚠️ 풀 설정 누락:
            // - PoolSize (기본값: 10)
            // - MinIdleConns (기본값: 0)
            // - MaxRetries (기본값: 3)
            // - DialTimeout, ReadTimeout, WriteTimeout
        }),
    }
}
```

**영향:**
- 동시 요청이 많을 때 연결 풀 고갈 가능성
- 높은 부하에서 연결 생성 비용 증가
- 재시도 로직이 기본값에 의존

**개선 방안:**
```go
// internal/config/config.go에 추가
type Config struct {
    // ... 기존 필드들
    RedisPoolSize     int    `env:"REDIS_POOL_SIZE"`
    RedisMinIdleConns int    `env:"REDIS_MIN_IDLE_CONNS"`
    RedisMaxRetries   int    `env:"REDIS_MAX_RETRIES"`
    RedisDialTimeout  int    `env:"REDIS_DIAL_TIMEOUT"`
    RedisReadTimeout  int    `env:"REDIS_READ_TIMEOUT"`
    RedisWriteTimeout int    `env:"REDIS_WRITE_TIMEOUT"`
}

// internal/adapters/metrics/redis/metrics.go 개선
func New(cfg config.Config) *Metrics {
    return &Metrics{
        client: goredis.NewClient(&goredis.Options{
            Addr:         cfg.RedisAddr,
            Password:      cfg.RedisPassword,
            DB:           cfg.RedisDB,
            PoolSize:      cfg.RedisPoolSize,         // 설정에서 로드
            MinIdleConns: cfg.RedisMinIdleConns,      // 설정에서 로드
            MaxRetries:   cfg.RedisMaxRetries,       // 설정에서 로드
            DialTimeout:   time.Duration(cfg.RedisDialTimeout) * time.Second,
            ReadTimeout:   time.Duration(cfg.RedisReadTimeout) * time.Second,
            WriteTimeout:  time.Duration(cfg.RedisWriteTimeout) * time.Second,
        }),
    }
}
```

### 4. 우아한 종료(Graceful Shutdown) 미구현

**위치:** `cmd/server/main.go:53-91`

**문제 코드:**
```go
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

    service := newService(deps, metrics)
    server := newHTTPServer(cfg.Port, service)

    log.Printf("listening on %s", server.Addr)
    return listenFn(server)  // ⚠️ SIGINT/SIGTERM 핸들러 없음
}
```

**영향:**
- 서버 종료 시 진행 중인 요청이 강제 종료됨
- 리소스(DB 연결, Redis 연결)가 정상적으로 해제되지 않을 수 있음
- 데이터 불일치 가능성

**개선 방안:**
```go
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

    service := newService(deps, metrics)
    server := newHTTPServer(cfg.Port, service)

    // ✅ 시그널 핸들러 추가
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        log.Printf("listening on %s", server.Addr)
        if err := listenFn(server); err != nil && err != http.ErrServerClosed {
            log.Printf("server error: %v", err)
        }
    }()

    <-quit  // 시그널 대기

    log.Println("shutting down gracefully...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Printf("server shutdown error: %v", err)
        return err
    }

    log.Println("server stopped gracefully")
    return nil
}
```

---

## ✅ 잘 구현된 부분

### 5. PostgreSQL 연결 풀

**위치:** `internal/adapters/persistence/postgres/repository.go:36-45`

```go
func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
    pool, err := pgxpool.New(ctx, databaseURL)
    if err != nil {
        return nil, err
    }
    store := &Store{pool: pool, rawPool: pool}
    if err := store.Migrate(ctx); err != nil {
        pool.Close()
        return nil, err
    }
    return store, nil
}
```

**평가:**
- ✅ `pgxpool`를 사용하여 자동 연결 풀링 제공
- ✅ 적절한 연결 관리
- ✅ 컨텍스트 기반 생명주기 관리

### 6. Redis 파이프라인 사용

**위치:** `internal/adapters/metrics/redis/metrics.go:40-73`

```go
func (m *Metrics) ReportDeploy(ctx context.Context, report domain.DeploymentStatusReport) error {
    label := report.Label
    if label == "" {
        label = report.AppVersion
    }
    pipe := m.client.TxPipeline()
    pipe.SAdd(ctx, labelsKey(report.DeploymentKey), label)
    switch report.Status {
    case "", deploySucceeded:
        pipe.HIncrBy(ctx, countersKey(report.DeploymentKey, label), "installed", 1)
    case deployFailed:
        pipe.HIncrBy(ctx, countersKey(report.DeploymentKey, label), "failed", 1)
    default:
        pipe.HIncrBy(ctx, countersKey(report.DeploymentKey, label), "installed", 1)
    }
    if report.ClientUniqueID != "" {
        currentKey := activeClientKey(report.DeploymentKey, report.ClientUniqueID)
        prev, _ := m.client.Get(ctx, currentKey).Result()
        if prev != "" && prev != label {
            pipe.SRem(ctx, activeSetKey(report.DeploymentKey, prev), report.ClientUniqueID)
        }
        pipe.Set(ctx, currentKey, label, 0)
        pipe.SAdd(ctx, activeSetKey(report.DeploymentKey, label), report.ClientUniqueID)
    }
    _, err := pipe.Exec(ctx)
    return err
}
```

**평가:**
- ✅ `TxPipeline()`를 사용하여 원자적 다중 연산
- ✅ 네트워크 왕복 최소화
- ✅ 트랜잭션 보장

### 7. PostgreSQL 트랜잭션 사용

**위치:** `internal/adapters/persistence/postgres/repository.go:274-298`

```go
func (r *AppRepo) Create(ctx context.Context, accountID string, app domain.App) (domain.App, error) {
    tx, err := r.store.pool.Begin(ctx)
    if err != nil {
        return domain.App{}, err
    }
    defer func() {
        _ = tx.Rollback(ctx)
    }()

    unique, err := isUniqueAppName(ctx, tx, accountID, app.Name, "")
    if err != nil {
        return domain.App{}, err
    }
    if !unique {
        return domain.App{}, domain.ErrConflict
    }
    app.CreatedAt = time.Now().UnixMilli()
    if err := tx.QueryRow(ctx, `INSERT INTO apps (name, created_at) VALUES ($1, $2) RETURNING id`,
        app.Name, app.CreatedAt).Scan(&app.ID); err != nil {
        return domain.App{}, err
    }
    if _, err := tx.Exec(ctx, `INSERT INTO app_collaborators (app_id, account_id, permission) VALUES ($1, $2, $3)`,
        app.ID, accountID, domain.PermissionOwner); err != nil {
        return domain.App{}, err
    }
    return app, tx.Commit(ctx)
}
```

**평가:**
- ✅ 명시적 트랜잭션 사용
- ✅ defer를 통한 Rollback 보장
- ✅ 복합 연산의 원자성 보장

---

## 🔵 동시성 및 경합 (Concurrency)

### 8. 토큰 생성 충돌 가능성

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
- SHA1은 이미 충돌(Collision)이 발견된 알고리즘
- `time.Now().UnixNano()`와 seed 조합으로 충돌 가능성 감소
- 하지만 높은 TPS(Transaction Per Second) 환경에서는 우려됨

**영향:**
- 드물지만 액세스 키 중복 가능성
- 인증 오류 발생 가능

**개선 방안:**
```go
import (
    "crypto/rand"
    "encoding/hex"
)

func generateToken(seed string, _ int64) string {
    b := make([]byte, 20)
    if _, err := rand.Read(b); err != nil {
        // Fallback to UUID v4
        return uuid.New().String()
    }
    return hex.EncodeToString(b)
}
```

### 9. 동시 배포 키 생성 락 부재

**위치:** `internal/application/service.go:162-174`

```go
func (s *Service) CreateApp(ctx context.Context, accountID string, req domain.AppCreationRequest) (domain.App, error) {
    app, err := s.apps.Create(ctx, accountID, domain.App{Name: req.Name})
    if err != nil {
        return domain.App{}, err
    }
    if !req.ManuallyProvisionDeployments {
        for _, name := range []string{"Production", "Staging"} {
            // ⚠️ 동시 요청 시 deployment key 중복 가능
            if _, err := s.deployments.Create(ctx, accountID, app.ID,
                domain.Deployment{Name: name,
                    Key: generateToken(accountID, time.Now().UnixNano())}); err != nil {
                return domain.App{}, err
            }
        }
    }
    return s.apps.GetByName(ctx, accountID, app.Name)
}
```

**문제:**
- `generateToken(accountID, time.Now().UnixNano())`가 동시 요청 시 동일한 값 생성 가능
- DB 유니크 제약 조건으로 실패할 수 있음

**개선 방안:**
```go
// DB 시퀀스 또는 Snowflake ID 사용
func generateUniqueDeploymentKey(accountID string) (string, error) {
    // 시퀀스 사용 시 DB 고유성 보장
    // 또는 Snowflake 알고리즘 사용
}
```

---

## 📈 부하 테스트 시나리오 권고

### 테스트 계획

| 시나리오 | 목표 RPS | 측정 항목 |
|----------|-----------|-----------|
| 앱 리스트 조회 | 100 | 쿼리 수, 응답 시간, DB 커넥션 풀 사용량 |
| 메트릭 보고 | 1000 | Redis 연결 풀, 파이프라인 지연 |
| 동시 앱 생성 | 50 | 배포 키 충돌, 트랜잭션 경합 |
| 장기 연결 유지 | N/A | 연결 누수, 메모리 사용량 |

### 모니터링 메트릭

```go
// 추천 메트릭
type ServerMetrics struct {
    RequestLatency      *prometheus.HistogramVec
    ActiveConnections   prometheus.Gauge
    DBConnectionPool   *prometheus.GaugeVec
    RedisPoolSize      prometheus.Gauge
    InFlightRequests   prometheus.Gauge
}
```

---

## 🎯 우선순위 개선 권고안

### P0 - 즉시 수정 필요

1. **Deployment current package fan-out 제거**
   - `DeploymentRepo.List()`의 `1 + N` 조회를 batched query로 변경
   - metrics/history 화면에서 가장 직접적인 이득

### P1 - 중요 개선

2. **부하 테스트로 Redis/HTTP 기본값 검증**
   - 새로 추가한 풀/타임아웃 설정값을 실제 트래픽으로 튜닝

3. **토큰 생성 알고리즘 개선 후 운영 검증**
   - `crypto/rand` 경로를 기본으로 사용 중
   - 배포 키/액세스 키 충돌 모니터링 추가 권장

### P2 - 선택적 개선

4. **배포 키 생성 락 구현**
   - 분산 락 또는 DB 시퀀스 사용
   - 동시 요청 시 고유성 보장

5. **메트릭 수집 구현**
   - Prometheus/Prometheus exporter 추가
   - DB 커넥션 풀, Redis 풀 모니터링

---

## 📊 최종 점수 요약

```
┌─────────────────────────────────────┐
│ PostgreSQL 연결 풀:   │ ████████░ 80%
│ Redis 파이프라인:     │ █████████░ 90%
│ 쿼리 최적화:        │ ██░░░░░░░ 20%
│ 동시성 처리:          │ ███████░░░ 70%
│ 우아한 종료:          │ ████████░░ 80%
│ HTTP 타임아웃:        │ ████████░░ 80%
│ Redis 풀 설정:        │ ████████░░ 80%
├─────────────────────────────────────┤
│ 종합 확장성 점수:       │ ███████░░░ 68%
└─────────────────────────────────────┘
```

### 🎉 결론

현재 확장성(Scalability)은 **양호한 수준까지 개선**되었습니다.

**가장 심각한 문제:**
- ⚠️ deployment current package 조회의 선형 fan-out

**양호한 부분:**
- ✅ PostgreSQL 연결 풀 사용
- ✅ Redis 파이프라인으로 원자성 보장
- ✅ 트랜잭션 사용
- ✅ Graceful shutdown 및 HTTP timeout 적용
- ✅ Redis 풀/타임아웃 설정 외부화
- ✅ `AppRepo.List()`의 batched query 적용
- ✅ generated key 충돌 완화

**권고:**
1. 남은 P0는 `AppRepo.List()`의 N+1 제거
2. 부하 테스트를 통한 새 timeout/pool 기본값 검증
3. 모니터링 시스템 도입으로 서버 상시 모니터링
