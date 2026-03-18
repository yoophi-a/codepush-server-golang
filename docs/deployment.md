# Deployment Guide

## Container Image

This project ships with a multi-stage Docker build:

- builder image: `golang:1.26-alpine`
- runtime image: `gcr.io/distroless/static-debian12:nonroot`

The resulting container:

- runs as non-root
- listens on port `3000`
- expects configuration entirely through environment variables

## Build

```bash
docker build -t codepush-server-golang:local .
```

## Local Compose Stack

For local runtime validation, use the checked-in [compose.yaml](/Users/yoophi/proj/codepush-server-golang/compose.yaml).

It starts:

- PostgreSQL
- Redis
- MinIO
- MinIO bucket initialization
- the Go service

Commands:

```bash
docker compose up --build -d
docker compose logs -f app
docker compose down -v
```

Quick validation:

```bash
curl http://localhost:3000/health
curl -H 'Authorization: Bearer dev-access-key' http://localhost:3000/account
```

## Run With S3

```bash
docker run --rm -p 3000:3000 \
  -e DATABASE_URL='postgres://postgres:postgres@host.docker.internal:5432/codepush?sslmode=disable' \
  -e REDIS_ADDR='host.docker.internal:6379' \
  -e STORAGE_BACKEND='s3' \
  -e S3_BUCKET='codepush' \
  -e S3_REGION='ap-northeast-2' \
  -e S3_ACCESS_KEY_ID='...' \
  -e S3_SECRET_ACCESS_KEY='...' \
  -e BOOTSTRAP_ACCOUNT_EMAIL='admin@example.com' \
  -e BOOTSTRAP_ACCOUNT_NAME='Admin' \
  -e BOOTSTRAP_ACCESS_KEY='dev-access-key' \
  codepush-server-golang:local
```

## Run With MinIO

```bash
docker run --rm -p 3000:3000 \
  -e DATABASE_URL='postgres://postgres:postgres@host.docker.internal:5432/codepush?sslmode=disable' \
  -e REDIS_ADDR='host.docker.internal:6379' \
  -e STORAGE_BACKEND='minio' \
  -e MINIO_ENDPOINT='host.docker.internal:9000' \
  -e MINIO_ACCESS_KEY='minioadmin' \
  -e MINIO_SECRET_KEY='minioadmin' \
  -e MINIO_BUCKET='codepush' \
  -e MINIO_USE_SSL='false' \
  -e BOOTSTRAP_ACCESS_KEY='dev-access-key' \
  codepush-server-golang:local
```

## Production Checklist

- provision PostgreSQL and Redis
- provision S3 or MinIO bucket
- set `DATABASE_URL`
- set `REDIS_ADDR`
- choose `STORAGE_BACKEND`
- set bootstrap account values
- expose port `3000`
- wire `/health` to the platform health check

## Recommended Deployment Flow

1. Build the image.
2. Push the image to your registry.
3. Deploy with environment variables from a secret manager.
4. Verify `/health`.
5. Verify authenticated access with the bootstrap bearer token.
6. Run smoke tests against the deployed service.

## Notes

- database schema is applied automatically on startup
- current release upload and promote endpoints are intentionally out of scope
- OAuth login endpoints are placeholders and should not be exposed as production auth flows yet
- local compose defaults use MinIO for file storage and `dev-access-key` as the bootstrap bearer token
