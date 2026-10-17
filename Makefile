# Developer tasks for Eventify

SHELL := /bin/bash

BINARY := bin/server

# Tool versions
GOLANGCI_LINT_VERSION := v2.14.0
GOSEC_VERSION ?= v2.29.0
GOVULNCHECK_VERSION ?= v1.9.0

# Migration configuration
MIGRATIONS_PATH := ./cmd/migrate/migration
STEPS ?= 1

# Local security tool installation directory
GOBIN_LOCAL := $(CURDIR)/.bin

.PHONY: help build run test lint fmt vuln tools migrate-up migrate-down schema-check
.PHONY: security-install security-scan gosec govulncheck

help:
	@echo "Available commands:"
	@echo "  build             - Build the server binary"
	@echo "  run               - Run the server locally"
	@echo "  test              - Run tests with the race detector"
	@echo "  lint              - Run golangci-lint"
	@echo "  fmt               - Format Go source files"
	@echo "  vuln              - Run Go vulnerability checks"
	@echo "  tools             - Install golangci-lint"
	@echo "  migrate-up        - Apply database migrations"
	@echo "  migrate-down      - Revert database migrations"
	@echo "  schema-check      - Verify migration schema"
	@echo "  security-install  - Install gosec and govulncheck"
	@echo "  gosec             - Run static security analysis"
	@echo "  govulncheck       - Check reachable vulnerabilities"
	@echo "  security-scan     - Run both security scans"

# Build the application
build:
	mkdir -p bin
	go build -trimpath -o $(BINARY) ./cmd/server

# Run locally on the host.
# If .env uses the Docker hostname "db", override DB_ADDR for host execution.
# Example:
# DB_ADDR='postgres://local_go_user:PASSWORD@localhost:5433/local_go_db?sslmode=disable' make run
run:
	go run ./cmd/server

# The race detector requires CGO and a working C compiler.
# On Ubuntu, install one with: sudo apt install build-essential
test:
	go test -race -count=1 ./...

# Lint the project
lint:
	golangci-lint run ./...

# Format Go source files
fmt:
	golangci-lint fmt ./...

# Check dependencies against the Go vulnerability database
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

# Install golangci-lint
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# These migration targets run inside the existing API container.
# Start the Docker Compose services before using them.
migrate-up:
	docker compose exec -T api sh -c '/go/bin/migrate -path=$(MIGRATIONS_PATH) -database "$$DB_ADDR" up'

migrate-down:
	docker compose exec -T api sh -c '/go/bin/migrate -path=$(MIGRATIONS_PATH) -database "$$DB_ADDR" down $(STEPS)'

# Verify migrations against a scratch database.
# The live database should only be read by the schema-check script.
schema-check:
	bash scripts/schema-check

# Install pinned security tools into the project-local .bin directory.
security-install:
	@echo "Creating local tool directory at $(GOBIN_LOCAL)..."
	mkdir -p "$(GOBIN_LOCAL)"
	@echo "Installing gosec $(GOSEC_VERSION)..."
	GOBIN="$(GOBIN_LOCAL)" go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	@echo "Installing govulncheck $(GOVULNCHECK_VERSION)..."
	GOBIN="$(GOBIN_LOCAL)" go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	@echo "Security tools installed successfully."

# Run gosec static security analysis.

gosec:
	@test -x "$(GOBIN_LOCAL)/gosec" || (echo "gosec not found. Run 'make security-install' first."; exit 1)
	"$(GOBIN_LOCAL)/gosec" -exclude-dir=tests -fmt=text ./...

govulncheck:
	@test -x "$(GOBIN_LOCAL)/govulncheck" || (echo "govulncheck not found. Run 'make security-install' first."; exit 1)
	"$(GOBIN_LOCAL)/govulncheck" ./...

security-scan: gosec govulncheck
	@echo "Security scans completed successfully."

