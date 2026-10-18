# C1 — Pre-cache performance baseline

## Scope and routes

Routes confirmed in source (`cmd/server/main.go` mounts `/api/v1`):

- Availability: `GET /api/v1/availability/services/:serviceId?date=YYYY-MM-DD` or `GET /api/v1/availability/staff/:staffId?date=YYYY-MM-DD`. Both require a Bearer token. The handler rejects a missing `date` with HTTP 400. Use a real, existing service/staff UUID and a fixed date in the test dataset.
- Services list: `GET /api/v1/services` (public).
- Settings: `GET /api/v1/settings` (requires a Bearer token; authentication also checks the user in PostgreSQL).

The settings repository performs `INSERT INTO settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING` followed by a `SELECT`. Therefore, ensure the singleton settings row exists before timing, to keep first-use row creation out of measured samples.

## Harness

`scripts/performance/http_baseline.py` is a standard-library-only HTTP load harness. It sends GET requests only, consumes response bodies, records per-request client-observed latency, computes nearest-rank p50/p95/p99, status/error counts and throughput. It separately records warm-up and each repeated run. It does not mutate application data or call payment/email services.

Example (use a dedicated disposable test database/API and valid test token):

```bash
export BASE_URL=http://127.0.0.1:8080
export BASELINE_TOKEN='<valid test-user bearer token>'
python3 scripts/performance/http_baseline.py \
  --availability-path '/api/v1/availability/services/<existing-service-uuid>?date=2026-10-19' \
  --concurrency 8 --warmup 30 --requests 300 --runs 3 --timeout 5 \
  --output docs/performance/baseline-results.json
```

Replace the placeholder with a real fixture ID; do not use a real user or production database. For the staff availability variant, supply `/api/v1/availability/staff/<existing-staff-uuid>?date=YYYY-MM-DD`. Run one variant per baseline so its results are independently attributable. Keep Go/runtime, PostgreSQL version, data snapshot, API settings, host load, token, date, and concurrency fixed for post-cache comparison.

## Database activity

The HTTP client cannot reliably attribute PostgreSQL statement counts to one HTTP path. For a controlled run, take before/after snapshots from the dedicated test database using a monitoring connection. If `pg_stat_statements` is already enabled, reset statistics only on the disposable test instance, run a single endpoint's warm-up and measured phase, then query `pg_stat_statements` filtered by normalized query text. Otherwise, use PostgreSQL `pg_stat_activity` sampling/logging or temporary statement logging configured only on the test instance. Record the exact method and settings; do not compare counts from a shared/production database. This harness intentionally does not enable extensions or change DB config.

CPU/memory are not captured by the HTTP harness. If desired, sample the API process/container with `pidstat`/`/usr/bin/time -v` or Docker stats on the same host and run; record the tool and sampling interval.

## Environment and provenance

The supplied project archive declares `postgres:16.3` in `docker-compose.yml` and the existing integration test helper uses Testcontainers `postgres:16.3`. Redis client setup exists in `internal/config/redis.go`, but the three read paths do not currently use caching. The supplied ZIP does not contain `.git`, so original commit SHA and working-tree status are **unavailable**. Capture them in the real checkout with `git rev-parse HEAD`, `git status --short --branch`, and `git diff --binary > /tmp/pre-c1-working-tree.patch` before applying this harness.

## Results status

No baseline numbers are recorded by this document. They must come from a successful run of the harness against a reachable, correctly seeded test API. The current execution environment does not have Docker installed and cannot download the Go 1.26.9 toolchain, so the integration stack and Go test suite could not be executed here. Do not treat this artifact as a completed performance baseline until results JSON has been generated successfully in the target environment.

## Limitations

- Client-observed latency includes network, middleware, auth, database access and response transfer; it is not isolated repository-query latency.
- The settings and availability routes require valid authentication; token validation itself performs a PostgreSQL user lookup.
- The harness uses concurrent batches of a fixed request count; it is not a closed-loop constant-rate generator and throughput depends on request completion time.
- PostgreSQL query counts, CPU and memory require separately collected measurements.
