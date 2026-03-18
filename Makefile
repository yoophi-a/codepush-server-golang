APP ?= codepush-server-golang
COMPOSE ?= docker compose

.PHONY: tidy fmt build run lint test unit integration e2e coverage docker-build docker-run compose-up compose-down compose-logs

tidy:
	go mod tidy

fmt:
	gofmt -w .

build:
	go build ./...

run:
	go run ./cmd/server

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0 run ./...

test: unit integration e2e

unit:
	go test ./...

integration:
	go test -tags=integration ./tests/integration/...

e2e:
	go test -tags=e2e ./tests/e2e/...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

docker-build:
	docker build -t $(APP):local .

docker-run:
	docker run --rm -p 3000:3000 $(APP):local

compose-up:
	$(COMPOSE) up --build -d

compose-down:
	$(COMPOSE) down -v

compose-logs:
	$(COMPOSE) logs -f app
