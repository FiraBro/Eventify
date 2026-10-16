# Developer tasks. Install the tools once with `make tools` (needs $(go env GOPATH)/bin on PATH).

SHELL := /bin/bash

BINARY := bin/server
GOLANGCI_LINT_VERSION := v2.14.0
GOVULNCHECK_VERSION := v1.9.0

# Path INSIDE the api container (same one docker-entrypoint.sh uses; WORKDIR is /app)
MIGRATIONS_PATH := ./cmd/migrate/migration

# Number of migrations `make migrate-down` reverts. Override: make migrate-down STEPS=2
STEPS ?= 1

.PHONY: build run test lint fmt vuln tools migrate-up migrate-down schema-check

build:
	go build -trimpath -o $(BINARY) ./cmd/server

# Runs on the HOST. .env has DB_ADDR pointing at the compose hostname "db", which does not
# resolve outside Docker, so override it, e.g.:
#   DB_ADDR='postgres://local_go_user:PASSWORD@localhost:5433/local_go_db?sslmode=disable' make run
run:
	go run ./cmd/server

# -race needs cgo: if you see "-race requires cgo", run: sudo apt install build-essential
test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# NOTE: migrate-up / migrate-down run the `migrate` binary that already exists inside the
# api container, so `docker compose up` must be running. They read DB_ADDR from the container.
migrate-up:
	docker compose exec -T api sh -c '/go/bin/migrate -path=$(MIGRATIONS_PATH) -database "$$DB_ADDR" up'

migrate-down:
	docker compose exec -T api sh -c '/go/bin/migrate -path=$(MIGRATIONS_PATH) -database "$$DB_ADDR" down $(STEPS)'

# Proves that the migrations alone reproduce the live schema (scratch database; live is only read).
schema-check:
	bash scripts/schema-check.sh