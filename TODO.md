# TODO

- [x] Analyze `../ref/code-push-server` routes and existing Swagger artifacts.
- [x] Generate an OpenAPI spec for the migration target, excluding endpoints that include release metadata payloads.
- [x] Document the migration scope, architecture, stack decisions, and delivery plan in `docs/*.md`.
- [ ] Bootstrap a Go service using Gin and a hexagonal architecture.
- [ ] Implement bearer-authenticated management APIs.
- [ ] Implement acquisition APIs and Redis-backed deployment metrics handling.
- [ ] Implement PostgreSQL persistence adapters.
- [ ] Implement AWS S3 and MinIO blob storage adapters.
- [ ] Add unit tests, integration tests, and e2e tests.
- [ ] Add lint, coverage, and Makefile automation.
- [ ] Commit at each major milestone.
