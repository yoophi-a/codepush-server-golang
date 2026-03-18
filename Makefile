APP ?= codepush-server-golang

.PHONY: tidy fmt build run lint test unit integration e2e coverage

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
