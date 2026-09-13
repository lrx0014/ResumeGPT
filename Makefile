.DEFAULT_GOAL := help

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: help dev-api dev-worker dev-web migrate test test-go test-web test-python build build-go build-web fmt compose-up compose-infra compose-logs compose-ps compose-down backup restore-check

help:
	@echo "ResumeGPT development commands"
	@echo "  make dev-api       Run the Go API"
	@echo "  make dev-worker    Run the Go worker"
	@echo "  make dev-web       Run the Vue development server"
	@echo "  make migrate       Apply database migrations"
	@echo "  make test          Run all tests"
	@echo "  make build         Build backend and frontend"
	@echo "  make compose-up    Build and start the complete local stack"
	@echo "  make compose-infra Start only PostgreSQL and MinIO"
	@echo "  make compose-logs  Follow local stack logs"
	@echo "  make compose-ps    Show local stack status"
	@echo "  make compose-down  Stop the local stack"
	@echo "  make backup        Create a PostgreSQL backup in .data/backups"
	@echo "  make restore-check Verify PostgreSQL backup and restore locally"

dev-api:
	go run ./cmd/api

dev-worker:
	go run ./cmd/worker

dev-web:
	npm --prefix apps/web run dev

migrate:
	go run ./cmd/migrate

test: test-go test-web test-python

test-go:
	go test ./...

test-web:
	npm --prefix apps/web run typecheck

test-python:
	cd services/document-worker && uv run pytest

build: build-go build-web

build-go:
	mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

build-web:
	npm --prefix apps/web run build

fmt:
	gofmt -w $$(find cmd internal migrations -name '*.go')

compose-up:
	docker compose up -d --build

compose-infra:
	docker compose up -d postgres minio

compose-logs:
	docker compose logs -f

compose-ps:
	docker compose ps

compose-down:
	docker compose down

backup:
	bash scripts/backup-postgres.sh

restore-check:
	bash scripts/check-postgres-restore.sh
