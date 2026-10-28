# Eventify Production Deployment Runbook

## 1. Prerequisites

- Docker Desktop or Docker Engine with Docker Compose.
- A configured `.env` file with the required production secrets.
- `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `JWT_SECRET`, `CHAPA_SECRET_KEY`, and `CHAPA_WEBHOOK_SECRET`.
- A secure backup destination outside this machine.

Never commit `.env` or database backup files to Git.

## 2. Validate and start production

Validate the Compose configuration:

```bash
docker compose -f docker-compose.prod.yml config --quiet
```

Build and start the production stack:

```bash
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml ps
```

The API is published on `127.0.0.1:8081`. This is a local-only binding, not a public internet deployment. Configure a secured reverse proxy and TLS before exposing the service publicly.

## 3. Check API health

```bash
scripts/deployment/smoke-test.sh
```

The script checks `/healthz` and `/readyz` on port 8081 by default.

## 4. Create a PostgreSQL backup

```bash
scripts/deployment/backup-db.sh
```

Backups are written to `backups/` using timestamped filenames. The script validates the archive and refuses to overwrite an existing backup file.

Keep backups secure and copy important backups to storage outside the host. Backups on the same machine do not protect against disk or host failure.

## 5. Validate a backup archive

Replace the placeholder filename with the actual backup filename:

```bash
docker compose -f docker-compose.prod.yml exec -T db \
  pg_restore --list < backups/YOUR-BACKUP-FILENAME.dump
```

## 6. Restore-drill procedure

**Never restore a test backup into the production database.** Use a uniquely named test database.

Create a test database only if it does not already exist. Example:

```bash
docker compose -f docker-compose.prod.yml exec -T db \
  psql -v ON_ERROR_STOP=1 -U local_go_user -d postgres \
  -c 'CREATE DATABASE eventify_restore_drill_YYYYMMDD TEMPLATE template0;'
```

Replace `YYYYMMDD` with a unique date suffix before running the command.

Restore the selected backup into that test database:

```bash
docker compose -f docker-compose.prod.yml exec -T db \
  pg_restore --exit-on-error --no-owner --no-privileges \
  -U local_go_user -d eventify_restore_drill_YYYYMMDD \
  < backups/YOUR-BACKUP-FILENAME.dump
```

Verify the tables and migration state:

```bash
docker compose -f docker-compose.prod.yml exec -T db \
  psql -U local_go_user -d eventify_restore_drill_YYYYMMDD -c '\dt'

docker compose -f docker-compose.prod.yml exec -T db \
  psql -U local_go_user -d eventify_restore_drill_YYYYMMDD \
  -c 'SELECT version, dirty FROM schema_migrations;'
```

Replace the database and backup filename placeholders consistently. Confirm the target database name before every restore.

## 7. Application validation

Run from the project root:

```bash
go test ./...
go vet ./...
go build ./...
git diff --check
docker compose -f docker-compose.prod.yml config --quiet
scripts/deployment/smoke-test.sh
```

## 8. Important safeguards

- Never run `docker compose down -v` as routine deployment cleanup.
- Never remove production database or Redis volumes during deployment.
- Never restore a backup into production as a test.
- Keep development and production Compose stacks and data separate.
- Test backup restoration periodically and document the result.
- Before public deployment, configure HTTPS, a reverse proxy, appropriate CORS origins, secure secret management, and off-host backups.
