# Detailed Delivery Spec

## API Contract

The service contract is defined by `api/openapi.yaml`, derived from the reference project route surface.

Rules:

- preserve path shapes and response envelopes where practical
- use JSON for API responses except the welcome and HTML auth pages
- require bearer auth on management endpoints
- omit release upload/update endpoints because they depend on metadata-bearing payloads that are out of scope

## Functional Requirements

### Management APIs

- account retrieval
- access key CRUD
- session deletion by creator
- app CRUD
- collaborator add/list/remove
- app transfer
- deployment CRUD
- deployment history fetch and clear
- deployment metrics fetch
- rollback to latest previous or explicit label

### Acquisition APIs

- update check
- deploy status report
- download status report
- legacy `v0.1/public/codepush/*` aliases

### Metrics

- track download counts
- track deploy success/failure counts
- track active release label per client
- clear deployment metrics when history is purged

## Non-Functional Requirements

- structured logging
- environment-driven configuration
- transactional repository operations
- deterministic tests
- CI-friendly lint and coverage commands

## Testing Requirements

- unit tests for application services and selector logic
- integration tests for PostgreSQL repositories, Redis metrics, and MinIO blob adapter
- e2e tests for authenticated management flows and acquisition flows
