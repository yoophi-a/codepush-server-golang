# Test Strategy And TODO

## Goal

현재 테스트는 일부 happy path 에 치우쳐 있고, 기본 `go test ./... -cover` 기준으로 핵심 패키지 대부분이 `0.0%` 입니다.

우선순위는 아래 순서로 잡습니다.

1. `internal/application` 의 비즈니스 규칙을 `unit test` 로 먼저 고정
2. `ginhandler` 와 `middleware` 의 HTTP 계약과 에러 매핑을 `integration test` 로 검증
3. `postgres`, `redis`, `minio/s3` 어댑터의 권한, 충돌, 상태 전이 규칙을 `integration test` 로 분리
4. `config` 와 `cmd/server` 는 smoke 수준으로 최소 검증

## Baseline

- `go test ./... -cover` 실행 완료
- 현재 커버리지 현황 확인 완료
- 기존 테스트 위치와 범위 확인 완료

### Current Coverage Snapshot

`go test ./... -coverprofile=/tmp/codepush-cover.out` 재실행 기준:

- `total`: `61.6%`

- `internal/application`: `83.3%`
- `internal/adapters/http/ginhandler`: `40.1%`
- `internal/adapters/http/middleware`: `100.0%`
- `internal/adapters/metrics/redis`: `91.9%`
- `internal/adapters/persistence/postgres`: `67.5%`
- `internal/adapters/storage/minio`: `63.6%`
- `internal/adapters/storage/s3`: `42.1%`
- `internal/config`: `93.8%`
- `cmd/server`: `0.0%`

참고:

- `postgres`, `redis`, `minio`, `s3` 는 `tests/integration` 패키지에서 태그 기반으로 검증 중이라 기본 package coverage 수치에는 직접 반영되지 않습니다.
- `go test -tags=integration ./tests/integration/...` 는 통과했습니다.
- 기본 커버리지 목표였던 `60%+` 달성 완료.

## Strategy

### 1. Unit Test First

테스트 비용 대비 효과가 가장 큰 곳은 `internal/application/service.go` 입니다.
이 계층은 외부 의존성을 fake 로 대체하기 쉬워서 빠르게 높은 커버리지와 회귀 방지 효과를 얻을 수 있습니다.

집중 대상:

- access key 생성/수정 규칙
- app/deployment 생성 규칙
- history clear 와 metrics clear 연동
- rollback 선택 규칙
- update check 선택 규칙
- semver / rollout / HTTP status 매핑 유틸리티

### 2. HTTP Contract Next

라우터는 엔드포인트 수가 많고, snake_case/camelCase alias, malformed JSON, 인증 실패, 에러 상태코드 매핑 같은 계약이 많습니다.
서비스 fake 를 붙인 `httptest` 기반 테스트로 빠르게 방어막을 칠 수 있습니다.

집중 대상:

- acquisition public API
- report status API
- authed management API 의 인증/에러 매핑
- middleware bearer token 처리

### 3. Repository And Adapter Integration

DB 와 Redis 계층은 단순 CRUD 보다 권한, 충돌, 소유권 이전, collaborator 처리, 만료 토큰, active metrics 이동 같은 규칙 검증이 핵심입니다.

집중 대상:

- Postgres 권한/조회/충돌 규칙
- Redis metrics 상태 전이
- MinIO/S3 storage 계약

### 4. Low-Cost Smoke Tests

`config.Load()` 는 table test 로 env 파싱만 검증하고, `cmd/server/main.go` 는 과도한 단위 테스트보다 wiring smoke 수준으로 제한합니다.

## TODO

### Phase 0. Audit And Planning

- [x] 현재 테스트 파일과 커버리지 상태를 조사한다.
- [x] 핵심 로직 파일과 테스트 공백을 분류한다.
- [x] 우선순위 기반 테스트 전략을 문서화한다.

### Phase 1. Application Unit Tests

- [x] `ListAccessKeys` 가 이름을 마스킹하고 생성시간 순으로 정렬하는지 테스트한다.
- [x] `CreateAccessKey` 의 기본 TTL, 자동 name 생성, `FriendlyName` 기본값, `createdBy=unknown` 규칙을 테스트한다.
- [x] `GetAccessKey` 와 `UpdateAccessKey` 가 민감정보를 숨기고 필드를 올바르게 갱신하는지 테스트한다.
- [x] `DeleteSessionsByCreator` 가 삭제 수 0 일 때 `ErrNotFound` 를 반환하는지 테스트한다.
- [x] `CreateApp` 이 기본 deployment 2개를 생성하는지 테스트한다.
- [x] `CreateApp` 이 `ManuallyProvisionDeployments=true` 일 때 자동 deployment 생성을 건너뛰는지 테스트한다.
- [x] `ListApps` 와 `ListDeployments` 가 정렬 규칙을 보장하는지 테스트한다.
- [x] `CreateDeployment` 가 key 미입력 시 자동 생성하는지 테스트한다.
- [x] `ClearHistory` 가 package clear 성공 후 metrics clear 를 호출하는지 테스트한다.
- [x] `ClearHistory` 가 package clear 실패 시 metrics clear 를 호출하지 않는지 테스트한다.
- [x] `Rollback` 의 no-history, single-history, target-label-not-found, app-version-conflict 케이스를 테스트한다.
- [x] `Rollback` 성공 시 clone 필드와 `ReleaseMethod` 가 올바른지 테스트한다.
- [x] `UpdateCheck` 가 disabled package 를 제외하는지 테스트한다.
- [x] `UpdateCheck` 가 semver range 를 올바르게 해석하는지 테스트한다.
- [x] `UpdateCheck` 가 동일 package hash 면 업데이트 없음으로 응답하는지 테스트한다.
- [x] `UpdateCheck` 가 rollout 과 `ClientUniqueID` 조건을 올바르게 적용하는지 테스트한다.
- [x] `ReportDeploy`, `ReportDownload`, `Health`, `Authenticate` 위임 동작을 테스트한다.
- [x] `matchesVersion`, `normalizeVersion`, `normalizeConstraint`, `HTTPStatus` 를 table test 로 추가한다.

### Phase 2. HTTP And Middleware Tests

- [x] `RequireAuth` 가 bearer header 누락 시 `401` 을 반환하는지 테스트한다.
- [x] `RequireAuth` 가 token trim 과 대소문자 bearer prefix 를 허용하는지 테스트한다.
- [x] `RequireAuth` 가 인증 실패 에러를 적절한 HTTP status 로 변환하는지 테스트한다.
- [x] `/health` 가 성공/실패에 따라 `OK` 와 `ERROR` 를 반환하는지 테스트한다.
- [x] `/updateCheck` 와 `/v0.1/public/codepush/update_check` 가 query alias 를 동일하게 해석하는지 테스트한다.
- [x] `/reportStatus/deploy` 와 `/reportStatus/download` 가 malformed JSON 에 대해 `400` 을 반환하는지 테스트한다.
- [x] `/accessKeys`, `/apps`, `/deployments`, `/history`, `/metrics`, `/rollback` 주요 엔드포인트의 성공/실패 status mapping 을 기본 경로 기준으로 테스트한다.
- [x] `firstNonEmpty` 와 `writeError` helper 동작을 테스트한다.

### Phase 3. Postgres Integration Tests

- [x] `EnsureBootstrap` 이 계정/키를 중복 생성하지 않는지 테스트한다.
- [x] `ResolveAccountIDByAccessKey` 의 정상, 미존재, 만료 토큰 케이스를 테스트한다.
- [x] `AccessKeyRepo.Create` 가 중복 token 에 대해 `ErrConflict` 를 반환하는지 테스트한다.
- [x] `AccessKeyRepo.Delete` 와 `DeleteSessionsByCreator` 의 not-found 동작을 테스트한다.
- [x] `AppRepo.Create` 가 동일 사용자 관점에서 중복 app 이름을 막는지 테스트한다.
- [x] `AppRepo.GetByName` 이 `ownerEmail:appName` 형식을 처리하는지 테스트한다.
- [x] `AppRepo.GetByName` 이 후보가 여러 개일 때 현재 owner app 을 우선 선택하는지 테스트한다.
- [x] `AppRepo.Transfer` 가 owner 를 collaborator 로 내리고 새 owner 를 설정하는지 테스트한다.
- [x] `AppRepo.RemoveCollaborator` 가 owner 자기 자신 제거를 금지하는지 테스트한다.
- [x] `AppRepo.RemoveCollaborator` 가 collaborator 자기 자신 제거를 허용하는지 테스트한다.
- [x] `DeploymentRepo` 의 owner/collaborator 권한 경계를 테스트한다.
- [x] `PackageRepo.ListHistory`, `ClearHistory`, `CommitRollback` 의 ordinal/label 증가 규칙을 테스트한다.
- [ ] `requirePermission`, `currentPackage`, `nextPackageOrdinal` 에 간접적으로 도달하는 시나리오를 보강한다.

### Phase 4. Redis And Storage Integration Tests

- [x] `ReportDeploy` 가 성공/실패 status 에 따라 installed/failed 카운터를 다르게 누적하는지 테스트한다.
- [x] `ReportDeploy` 가 label 비어있을 때 `AppVersion` 을 fallback 으로 사용하는지 테스트한다.
- [x] 동일 client 가 다른 label 로 이동할 때 active set 이 이전 label 에서 제거되는지 테스트한다.
- [x] `GetMetrics` 가 downloaded/installed/failed/active 를 정확히 집계하는지 테스트한다.
- [x] `Clear` 가 labels, counters, active set 을 삭제하는지 테스트한다.
- [x] MinIO `CheckHealth`, `PutObject`, `DeleteObject` 계약을 테스트한다.
- [x] S3 backend 는 local endpoint 기반 계약 테스트 또는 최소 smoke test 범위를 정의한다.

### Phase 5. Config And Boot Smoke Tests

- [x] `config.Load()` 의 필수 env, fallback 값, bool/int 파싱을 table test 로 추가한다.
- [ ] storage backend 선택 로직의 최소 smoke 검증 범위를 정의한다.
- [ ] `cmd/server/main.go` 는 직접 단위 테스트보다 wiring smoke 범위만 유지한다.

## Result

- 기본 목표였던 전체 coverage `60%+` 달성
- `go test ./...` 통과
- `go test ./... -coverprofile=/tmp/codepush-cover.out` 기준 `61.6%`
- `go test -tags=integration ./tests/integration/...` 통과

## Completion Rules

- TODO 는 실제 테스트가 추가되고 실행 확인이 끝난 뒤에만 완료 처리한다.
- 구현 중 새 리스크가 발견되면 TODO 를 추가한다.
- 큰 happy path 하나보다 실패 케이스를 작은 테스트로 분리한다.

## Suggested Milestones

1. `internal/application` 을 먼저 `60%+` 까지 올린다.
2. `ginhandler` 와 `middleware` 를 핵심 계약 기준 `40%+` 로 올린다.
3. `postgres` 와 `redis` 는 권한/상태 전이 시나리오 위주로 보강한다.
4. `config` 와 storage backend 는 smoke 수준으로 마무리한다.
