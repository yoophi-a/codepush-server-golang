# Recommended Stack

## Runtime

- Go 1.26+
- Gin for HTTP routing and middleware
- `jackc/pgx/v5` with `pgxpool` for PostgreSQL
- `redis/go-redis/v9` for metrics and cache-style counters

## Architecture

- Hexagonal architecture
- `core/domain`: entities and policy logic
- `core/ports`: repository and infrastructure contracts
- `application`: use cases / services
- `adapters`: HTTP, persistence, metrics, file storage

## Storage

- PostgreSQL as the system of record
- AWS SDK v2 S3 adapter for AWS S3
- MinIO Go SDK adapter for MinIO
- Shared blob storage port so release artifact handling is backend-agnostic

## API and Validation

- OpenAPI 3.0 document stored in `api/openapi.yaml`
- `getkin/kin-openapi` for spec validation tests
- Request validation handled in Gin handlers and application layer

## Testing

- Unit tests with Go `testing`
- Integration tests with `testcontainers-go` for PostgreSQL, Redis, and MinIO
- E2E tests against an in-process Gin server with real backing services

## Quality Gates

- `golangci-lint`
- `go test -cover ./...`
- Make targets for setup, lint, unit, integration, e2e, and coverage
