#!/bin/sh
set -eu

if [ -z "${DB_ADDR:-}" ]; then
  export DB_ADDR="postgres://local_go_user:password@db:5432/local_go_db?sslmode=disable"
fi

if [ -x /go/bin/migrate ]; then
  /go/bin/migrate -path=./cmd/migrate/migration -database "$DB_ADDR" up
else
  echo "migrate tool not found; skipping migrations" >&2
fi

exec air -c .air.toml
