.DEFAULT_GOAL := help

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: help dev-api dev-worker dev-web migrate test test-go test-web build build-go build-web fmt compose-up compose-down backup restore-check

help:
	@echo "ResumeGPT development commands"
	@echo "  make dev-api       Run the Go API"
	@echo "  make dev-worker    Run the Go worker"
	@echo "  make dev-web       Run the Vue development server"
	@echo "  make migrate       Apply database migrations"
	@echo "  make test          Run all tests"
	@echo "  make build         Build backend and frontend"
	@echo "  make compose-up    Start local infrastructure"
	@echo "  make compose-down  Stop local infrastructure"
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
	gofmt -w $$(find cmd internal migrations -name '*.go')

compose-up:
	docker compose up -d

compose-down:
	docker compose down

backup:
	bash scripts/backup-postgres.sh

restore-check:
	bash scripts/check-postgres-restore.sh
