#!/usr/bin/env bash
# Proves that the migrations alone reproduce the live schema.
#   1. creates a scratch database (the live database is only READ, never changed)
#   2. applies every migration to it
#   3. round-trips the newest ROUNDTRIP migrations (default 3: down N, then up) to prove the down files work
#   4. compares the scratch schema with the live schema
# Needs `docker compose up` to be running. Exit code 0 = identical.
set -euo pipefail

LIVE="${LIVE_DB:-local_go_db}"
PGUSER_NAME="${PGUSER_NAME:-local_go_user}"
SCRATCH="schema_check_$$"
ROUNDTRIP="${ROUNDTRIP:-3}"

psql_admin() { docker compose exec -T db psql -v ON_ERROR_STOP=1 -U "$PGUSER_NAME" -d postgres -c "$1" >/dev/null; }
cleanup() { psql_admin "DROP DATABASE IF EXISTS ${SCRATCH};" || true; }
trap cleanup EXIT

migrate_scratch() {
  docker compose exec -T -e LIVE="$LIVE" -e SCRATCH="$SCRATCH" api sh -c \
    'D=$(echo "$DB_ADDR" | sed "s#/$LIVE?#/$SCRATCH?#"); /go/bin/migrate -path=./cmd/migrate/migration -database "$D" '"$1"
}

dump() {
  docker compose exec -T db pg_dump -U "$PGUSER_NAME" -d "$1" --schema-only --no-owner --no-privileges \
    | grep -v -E '^(-- Dumped|\\restrict|\\unrestrict)'
}

psql_admin "CREATE DATABASE ${SCRATCH};"
migrate_scratch "up"
migrate_scratch "down ${ROUNDTRIP}"
migrate_scratch "up"

if diff -u <(dump "$SCRATCH") <(dump "$LIVE"); then
  echo "schema-check: OK, a database built from the migrations is identical to ${LIVE}"
else
  echo "schema-check: FAILED, the differences are shown above" >&2
  exit 1
fi