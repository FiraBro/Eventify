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

.PHONY: help build run test test-unit test-race coverage lint fmt fmt-check vuln tools migrate-up migrate-down schema-check
.PHONY: security-install security-scan security gosec govulncheck

help:
	@echo "Available commands:"
	@echo "  build             - Build the server binary"
	@echo "  run               - Run the server locally"
	@echo "  test              - Run tests with the race detector (compatibility alias)"
	@echo "  test-unit         - Run the ordinary Go test suite"
	@echo "  test-race         - Run tests with the race detector"
	@echo "  coverage          - Run tests and write coverage.out + summary"
	@echo "  lint              - Run golangci-lint"
	@echo "  fmt               - Format Go source files"
	@echo "  fmt-check         - Fail if Go files need gofmt"
	@echo "  vuln              - Run Go vulnerability checks"
	@echo "  tools             - Install golangci-lint"
	@echo "  migrate-up        - Apply database migrations"
	@echo "  migrate-down      - Revert database migrations"
	@echo "  schema-check      - Verify migration schema"
	@echo "  security-install  - Install gosec and govulncheck"
	@echo "  gosec             - Run static security analysis"
	@echo "  govulncheck       - Check reachable vulnerabilities"
	@echo "  security-scan     - Run both security scans"
	@echo "  security          - Alias for security-scan"

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
test: test-race

test-unit:
	go test -count=1 ./...

test-race:
	go test -race -count=1 ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tee coverage-summary.txt

# Lint the project
lint:
	golangci-lint run ./...

# Fail when any Go source needs formatting; do not mutate files.
fmt-check:
	@files="$$(find . -type f -name '*.go' -not -path './vendor/*' -not -path './.git/*' -print0 | xargs -0 gofmt -l)"; if [ -n "$$files" ]; then printf '%s\n' 'Go files need gofmt:' "$$files"; exit 1; fi

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


security: security-scan
