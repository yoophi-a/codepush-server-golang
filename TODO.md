# TODO

## Done

- [x] Analyze `../ref/code-push-server` routes and existing Swagger artifacts.
- [x] Generate an OpenAPI spec for the migration target.
- [x] Exclude metadata-based endpoints from the migration scope.
- [x] Document architecture, stack decisions, and delivery plan under `docs/`.
- [x] Bootstrap a Go service with Gin and hexagonal architecture.
- [x] Implement bearer-authenticated management APIs.
- [x] Implement acquisition APIs and Redis-backed metrics handling.
- [x] Implement PostgreSQL persistence adapters.
- [x] Implement AWS S3 and MinIO storage adapters.
- [x] Add unit, integration, and e2e tests.
- [x] Add `Makefile`, coverage command, and lint configuration.
- [x] Add Dockerfile and deployment documentation.
- [x] Add `docker compose` stack for PostgreSQL, Redis, MinIO, and the application.
- [x] Create milestone commits during delivery.

## Remaining

- [ ] Implement real OAuth login/callback flows for GitHub, Microsoft, and Azure AD.
- [ ] Implement metadata-based release endpoints:
  `POST /apps/{appName}/deployments/{deploymentName}/release`
  `PATCH /apps/{appName}/deployments/{deploymentName}/release`
  `POST /apps/{appName}/deployments/{sourceDeploymentName}/promote/{destDeploymentName}`
- [ ] Connect blob storage adapters to release upload flows once metadata endpoints are in scope.
- [ ] Add repository and handler tests with broader behavioral coverage.
- [ ] Add CI workflow for lint, unit, integration, e2e, and coverage reporting.
