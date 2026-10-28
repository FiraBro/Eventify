#!/usr/bin/env bash
set -Eeuo pipefail

COMPOSE_FILE="docker-compose.prod.yml"
BACKUP_DIR="${BACKUP_DIR:-backups}"

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
umask 077

if ! docker compose -f "$COMPOSE_FILE" ps --status running db \
    --format '{{.Service}}' | grep -qx 'db'; then
    echo "ERROR: Production database container is not running." >&2
    exit 1
fi

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup_file="${BACKUP_DIR}/eventify-postgres-${timestamp}.dump"

if [[ -e "$backup_file" ]]; then
    echo "ERROR: Backup already exists; refusing to overwrite it." >&2
    exit 1
fi

echo "Creating PostgreSQL backup..."

if ! docker compose -f "$COMPOSE_FILE" exec -T db \
    sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' \
    > "$backup_file"; then
    rm -f "$backup_file"
    echo "ERROR: PostgreSQL backup failed." >&2
    exit 1
fi

if [[ ! -s "$backup_file" ]]; then
    rm -f "$backup_file"
    echo "ERROR: Backup file is empty." >&2
    exit 1
fi

if ! docker compose -f "$COMPOSE_FILE" exec -T db \
    pg_restore --list < /dev/null >/dev/null 2>&1; then
    # The command above only checks tool availability; validate the actual dump below.
    :
fi

if ! docker compose -f "$COMPOSE_FILE" exec -T db \
    pg_restore --list < "$backup_file" >/dev/null; then
    rm -f "$backup_file"
    echo "ERROR: Backup validation failed." >&2
    exit 1
fi

chmod 600 "$backup_file"

echo "Backup created and validated: $backup_file"
du -h "$backup_file"
