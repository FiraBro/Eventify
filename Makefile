# -----------------------------
# Migration configuration
# -----------------------------
MIGRATE_PATH=./cmd/migrate/migration

DB_USER=local_go_user
DB_PASSWORD=password
DB_NAME=local_go_db
DB_HOST=localhost
DB_PORT=5432

DB_ADDR=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# -----------------------------
# Commands
# -----------------------------
.PHONY: help migrate-up migrate-down migrate-version migrate-create

help:
	@echo ""
	@echo "Available commands:"
	@echo "  make migrate-up            Apply all migrations"
	@echo "  make migrate-down          Rollback last migration"
	@echo "  make migrate-version       Show migration version"
	@echo "  make migrate-create NAME=x Create new migration"
	@echo ""

migrate-up:
	migrate -path=$(MIGRATE_PATH) -database="$(DB_ADDR)" up

migrate-down:
	migrate -path=$(MIGRATE_PATH) -database="$(DB_ADDR)" down 1

migrate-version:
	migrate -path=$(MIGRATE_PATH) -database="$(DB_ADDR)" version

migrate-create:
ifndef NAME
	$(error NAME is not set. Usage: make migrate-create NAME=create_users)
endif
	migrate create -seq -ext sql -dir=$(MIGRATE_PATH) $(NAME)
