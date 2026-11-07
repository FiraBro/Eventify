#!/bin/sh
set -eu

TARGET_DB_URL="${DATABASE_URL:-${DB_ADDR:-}}"
if [ -z "$TARGET_DB_URL" ]; then
  TARGET_DB_URL="postgres://local_go_user:password@127.0.0.1:5433/local_go_db?sslmode=disable"
fi

export DB_ADDR="$TARGET_DB_URL"
export DATABASE_URL="$TARGET_DB_URL"

if [ -x /go/bin/migrate ]; then
  echo "Applying pending database migrations..."
  /go/bin/migrate -path=./cmd/migrate/migration -database "$TARGET_DB_URL" up
else
  echo "migrate tool not found at /go/bin/migrate; skipping migrations" >&2
fi

echo "Starting Eventify API via Air live reload..."
exec air -c .air.toml
