# Migration Architecture

## Scope

This project rewrites the management and acquisition portions of the reference `code-push-server` into Go.

The migration target includes:

- health check
- bearer-authenticated account, access key, app, collaborator, deployment, history, metrics, transfer, rollback, and acquisition endpoints
- legacy and current acquisition endpoints

The migration explicitly excludes endpoints that carry release metadata payloads:

- `POST /apps/{appName}/deployments/{deploymentName}/release`
- `PATCH /apps/{appName}/deployments/{deploymentName}/release`
- `POST /apps/{appName}/deployments/{sourceDeploymentName}/promote/{destDeploymentName}`

## Layering

### Domain

Pure domain models:

- Account
- AccessKey
- App
- Collaborator
- Deployment
- Package
- Metrics aggregates

### Ports

Primary ports:

- account/app/deployment/package management services
- acquisition service

Secondary ports:

- account repository
- app repository
- deployment repository
- access key repository
- package repository
- metrics repository
- blob storage
- clock
- id/key generator

### Adapters

- Gin HTTP adapter
- PostgreSQL repositories using transactional SQL
- Redis metrics adapter
- AWS S3 adapter
- MinIO adapter

## Persistence Model

PostgreSQL tables:

- `accounts`
- `access_keys`
- `apps`
- `app_collaborators`
- `deployments`
- `packages`
- `package_history`

Redis keyspace:

- deployment status counters
- active label tracking by client

## Authentication Model

- Bearer token authentication with access-key lookup
- Optional HTML auth endpoints remain lightweight placeholders for parity with the reference route surface

## Delivery Plan

1. Write OpenAPI and project docs.
2. Bootstrap application, config, and domain contracts.
3. Implement repositories and adapters.
4. Implement HTTP handlers.
5. Add tests and automation.
