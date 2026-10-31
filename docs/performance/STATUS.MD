# Performance and cache-resilience status

This status reflects the artifacts present in the supplied project archive. The archive excludes `.git`, so the exact commit and uncommitted working-tree state cannot be established from it.

## Cache implementation

- C2 cache-aside is wired for service availability, services list, and settings.
- C3 shared generation-token invalidation is implemented for the documented mutation families.
- C4 resilience hardening: when Redis generation lookup or cache reads fail, concurrent source fallbacks for the same resource are coalesced within one process using `singleflight`. The source work is bounded by `CacheFillTimeout`; individual waiters can still cancel independently. This does not provide distributed singleflight and does not make invalidation strict/transactional.
- Unit tests for the Redis-outage fallback paths are in `internal/cacheaside/cache_test.go`. They must be run in the target checkout with the repository's pinned Go toolchain.

## HTTP performance artifacts

All timings below are client-observed and come from one run of 200 requests per endpoint at concurrency 10. They are not repository-only timings.

### C1 baseline (`baseline-results.json`, 2026-10-18)

| Endpoint | Successful requests | p50 | p95 | p99 | Wall-clock req/s |
| --- | ---: | ---: | ---: | ---: | ---: |
| Availability | 0/200 (HTTP 500) | 19.744 ms | 65.072 ms | 81.735 ms | 353.52 |
| Services list | 200/200 | 17.215 ms | 46.931 ms | 63.007 ms | 449.08 |
| Settings | 200/200 | 16.254 ms | 43.198 ms | 51.849 ms | 447.48 |

Availability is not a valid performance baseline because all requests failed. The services-list and settings measurements are successful C1 samples.

### Cached run (`c4-cached-results.json`, 2026-10-22)

| Endpoint | Successful requests | p50 | p95 | p99 | Wall-clock req/s |
| --- | ---: | ---: | ---: | ---: | ---: |
| Availability | 200/200 | 15.846 ms | 55.684 ms | 67.612 ms | 393.78 |
| Services list | 200/200 | 6.967 ms | 13.045 ms | 15.511 ms | 1292.76 |
| Settings | 200/200 | 11.515 ms | 39.260 ms | 42.335 ms | 570.67 |

This run is successful for all three endpoints. It is directionally useful, but the archive does not record a fixed database snapshot or source commit for both runs, so do not present the C1-to-C4 differences as a controlled causal comparison.

### Other artifacts

- `c4-results.json` is invalid for performance comparison: all 600 measured requests failed with `URLError`; no HTTP statuses were received.
- `step9-baseline-results.json` (2026-10-30) has 200/200 successful responses for all three endpoints, but it was collected after cache implementation and is not a pre-cache baseline.
- `read-path-baseline.md` and `cache_aside.md` retain historical statements from earlier review stages; this status supersedes their outdated overall verification claims where they conflict.

## Verification still required in the real checkout

Run with Go 1.26.9 (or the CI-pinned compatible toolchain) and a dedicated disposable test stack:

1. `gofmt` on changed Go files.
2. `go test -count=1 ./...`
3. `go test -race -count=1 ./...`
4. `go vet ./...`
5. `go test -tags=integration -count=1 -timeout=20m ./tests/integration`
6. `REDIS_URL` pointed only at a dedicated disposable Redis instance for `go test -tags=integration -count=1 ./internal/cacheaside`.
7. Re-run the same HTTP workload against a controlled data snapshot if a defensible before/after comparison is required.

Do not run destructive database/Redis cleanup commands against the normal development or production volumes. PostgreSQL remains authoritative; Redis is an optional, fail-open optimization.
