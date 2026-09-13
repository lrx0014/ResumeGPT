.PHONY: help dev-api dev-worker dev-web test test-go test-web build build-go build-web fmt compose-up compose-down

help:
	@echo "ResumeGPT development commands"
	@echo "  make dev-api       Run the Go API"
	@echo "  make dev-worker    Run the Go worker"
	@echo "  make dev-web       Run the Vue development server"
	@echo "  make test          Run all tests"
	@echo "  make build         Build backend and frontend"
	@echo "  make compose-up    Start local infrastructure"
	@echo "  make compose-down  Stop local infrastructure"

dev-api:
	go run ./cmd/api

dev-worker:
	go run ./cmd/worker

dev-web:
	npm --prefix apps/web run dev

test: test-go test-web

test-go:
	go test ./...

test-web:
	npm --prefix apps/web run typecheck

build: build-go build-web

build-go:
	mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

build-web:
	npm --prefix apps/web run build

fmt:
	gofmt -w cmd internal

compose-up:
	docker compose up -d

compose-down:
	docker compose down

