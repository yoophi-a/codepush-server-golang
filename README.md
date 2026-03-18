# codepush-server-golang

Go rewrite of the management and acquisition portions of `../ref/code-push-server`.

## Status

Current implementation includes:

- Gin-based HTTP server
- hexagonal architecture
- PostgreSQL-backed repositories
- Redis-backed deployment metrics
- AWS S3 and MinIO storage adapters
- production-oriented Docker image
- local `docker compose` stack for PostgreSQL, Redis, MinIO, and the app
- OpenAPI spec in `api/openapi.yaml`
- unit, integration, and e2e tests
- `Makefile` and `golangci-lint` configuration

Intentionally excluded from the current scope:

- OAuth provider login flows are placeholders
- metadata-based release endpoints are skipped

Skipped endpoints:

- `POST /apps/{appName}/deployments/{deploymentName}/release`
- `PATCH /apps/{appName}/deployments/{deploymentName}/release`
- `POST /apps/{appName}/deployments/{sourceDeploymentName}/promote/{destDeploymentName}`

## Project Layout

- `cmd/server`: application entrypoint
- `internal/core/domain`: domain models and errors
- `internal/core/ports`: hexagonal ports
- `internal/application`: use cases and service orchestration
- `internal/adapters/http`: Gin router and auth middleware
- `internal/adapters/persistence/postgres`: PostgreSQL repositories and schema
- `internal/adapters/metrics/redis`: Redis metrics adapter
- `internal/adapters/storage/s3`: AWS S3 adapter
- `internal/adapters/storage/minio`: MinIO adapter
- `tests/integration`: container-based integration tests
- `tests/e2e`: end-to-end tests
- `docs/deployment.md`: deployment notes and container run examples

## Requirements

- Go 1.26+
- PostgreSQL
- Redis
- Docker for integration/e2e tests

Optional depending on storage backend:

- AWS S3
- MinIO

## Configuration

Required environment variables:

- `DATABASE_URL`

Common environment variables:

- `PORT`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `STORAGE_BACKEND` with `s3` or `minio`
- `BOOTSTRAP_ACCOUNT_EMAIL`
- `BOOTSTRAP_ACCOUNT_NAME`
- `BOOTSTRAP_ACCESS_KEY`

S3 configuration:

- `S3_BUCKET`
- `S3_REGION`
- `S3_ENDPOINT`
- `S3_ACCESS_KEY_ID`
- `S3_SECRET_ACCESS_KEY`
- `S3_USE_PATH_STYLE`

MinIO configuration:

- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `MINIO_BUCKET`
- `MINIO_USE_SSL`

## Run

```bash
make run
```

Or:

```bash
go run ./cmd/server
```

The server bootstraps one local account and bearer token from the `BOOTSTRAP_*` variables.

## Docker

Build:

```bash
make docker-build
```

Run:

```bash
docker run --rm -p 3000:3000 \
  -e DATABASE_URL='postgres://postgres:postgres@host.docker.internal:5432/codepush?sslmode=disable' \
  -e REDIS_ADDR='host.docker.internal:6379' \
  -e STORAGE_BACKEND='minio' \
  -e MINIO_ENDPOINT='host.docker.internal:9000' \
  -e MINIO_ACCESS_KEY='minioadmin' \
  -e MINIO_SECRET_KEY='minioadmin' \
  -e MINIO_BUCKET='codepush' \
  -e BOOTSTRAP_ACCESS_KEY='dev-access-key' \
  codepush-server-golang:local
```

Detailed deployment notes are in [docs/deployment.md](/Users/yoophi/proj/codepush-server-golang/docs/deployment.md).

## Docker Compose

Bring up PostgreSQL, Redis, MinIO, bucket initialization, and the app:

```bash
make compose-up
```

Check app logs:

```bash
make compose-logs
```

Stop and remove volumes:

```bash
make compose-down
```

Local endpoints:

- app: `http://localhost:3000`
- health: `http://localhost:3000/health`
- minio api: `http://localhost:9000`
- minio console: `http://localhost:9001`
- postgres: `localhost:5432`
- redis: `localhost:6379`

Quick verification after `make compose-up`:

```bash
curl http://localhost:3000/health
curl -H 'Authorization: Bearer dev-access-key' http://localhost:3000/account
```

## Deployment

Recommended deployment flow:

1. Build and push the Docker image.
2. Provide `DATABASE_URL`, `REDIS_ADDR`, storage backend variables, and bootstrap credentials through your secret manager.
3. Route platform health checks to `/health`.
4. Validate authenticated access using the bootstrap bearer token.

Current deployment caveats:

- schema migration runs automatically on startup
- OAuth login endpoints are placeholders
- metadata-based release and promote endpoints are intentionally not deployed because they are not implemented

## Quality Checks

```bash
make build
make lint
make unit
make integration
make e2e
make coverage
```

Notes:

- `make lint` downloads and runs `golangci-lint` via `go run`
- integration and e2e tests require Docker

Suggested test sequence before deployment:

```bash
make lint
make unit
make integration
make e2e
make coverage
```

Suggested quick runtime test with local infrastructure:

```bash
make compose-up
curl http://localhost:3000/health
curl -H 'Authorization: Bearer dev-access-key' http://localhost:3000/account
make compose-down
```

## API

- OpenAPI: [api/openapi.yaml](/Users/yoophi/proj/codepush-server-golang/api/openapi.yaml)
- Architecture notes: [docs/architecture.md](/Users/yoophi/proj/codepush-server-golang/docs/architecture.md)
- Deployment notes: [docs/deployment.md](/Users/yoophi/proj/codepush-server-golang/docs/deployment.md)
- Delivery spec: [docs/spec.md](/Users/yoophi/proj/codepush-server-golang/docs/spec.md)
- Stack decisions: [docs/stack.md](/Users/yoophi/proj/codepush-server-golang/docs/stack.md)

## Verified Commands

The following were run successfully during implementation:

- `go build ./...`
- `go test ./...`
- `go test -tags=integration ./tests/integration/...`
- `go test -tags=e2e ./tests/e2e/...`
- `go test -cover ./...`
- `make lint`
