# C2/C3 — Redis cache-aside and explicit invalidation

## Exact read scope

- `GET /api/v1/availability/services/:serviceId?date=YYYY-MM-DD` — 15-second TTL.
- `GET /api/v1/services` — 2-minute TTL.
- `GET /api/v1/settings` — 1-minute TTL.

No other read path is cached. PostgreSQL remains authoritative for booking validation. The staff availability route continues to use its existing direct source path.

## Keys and freshness

All cache families use explicit `local-go:<resource>:...:v1` namespaces. Availability's SHA-256 key input includes service ID and date; raw inputs are not stored in Redis keys. The services list and singleton business settings use stable suffixes because their read routes have no result-affecting query parameters or tenant selector.

C3 adds shared, random generation tokens per namespace. Cache keys include the token observed before lookup; invalidation replaces the token with one Redis `SET`. Old keys are not scanned or deleted. This avoids blocking `SCAN` in request handlers and bounds cleanup to the TTL of unreachable entries. Generation tokens have no TTL, and a missing token is initialized with `SET NX`; a new random token prevents old cache keys from becoming reachable again if Redis evicts the generation key.

## Mutation map

Invalidation occurs only after the relevant repository operation returns success. Transactional staff-service assignment and schedule methods return only after `Commit`, so their invalidation follows commit success.

| Successful mutation | Invalidated namespace(s) |
| --- | --- |
| Service create, update, delete | Services list; service availability |
| Booking create, delete, status transition (including cancel/confirm/complete) | Service availability |
| Booking update that changes staff or scheduled slot | Service availability |
| Booking notes-only update | None |
| Staff delete | Service availability |
| Staff-service assignment | Service availability |
| Staff schedule replacement | Service availability |
| Staff holiday addition | Service availability |
| Settings update with actual field changes | Settings |
| Failed mutation / validation / no-op settings update | None |

Staff create and basic name/email/phone update do not invalidate availability: those fields do not participate in the cached service-slot calculation. Payment/refund writes do not mutate the three cached read models in the inspected source snapshot.

## Failure and concurrency behavior

Redis is an optimization. If Redis reads or generation lookup fail, the read bypasses cache and calls its source loader. Concurrent fallbacks for the same cache key/resource variant are coalesced with the same process-local `singleflight` group, using a bounded context detached from an individual waiter's cancellation. No cache write is attempted on this outage path. This limits a Redis outage from multiplying identical source reads within one API process; it is not distributed singleflight. A successful database mutation is never converted to an error because invalidation fails; the application emits a warning containing only the static cache namespace, not credentials or request data. In that case, the previous cache entry may remain visible until its original TTL expires: up to 15 seconds for availability, 2 minutes for services, or 1 minute for settings.

On successful invalidation, new reads that resolve the new generation cannot address keys written under the old token. An in-flight fill may finish and write its old-generation key, but subsequent readers cannot reach that key. Requests that already resolved the old generation before invalidation may still complete using the old view; this is not strict linearizable consistency. Cross-instance invalidation depends on all instances using the same Redis namespace/store. No database outbox or schema change was introduced, so invalidation remains best-effort if Redis is unavailable.

Cache misses and Redis-outage source fallbacks remain coalesced with `singleflight` within each process. This is not distributed singleflight. Shared source work uses a 10-second bounded context detached from one caller's cancellation; each waiter can still stop waiting on its own context. JSON/cache write failures do not fail a successful source read.

## C1 baseline and C2 performance comparison

The C1 harness and baseline documentation are preserved. The C1 documentation says no valid pre-cache measurements were collected, and the supplied archive does not contain a verified post-C2 live benchmark result. Therefore no C2 performance benefit or C3 overhead is claimed here. Do not treat unit tests as performance evidence. Run the existing harness against the same disposable dataset, API host, concurrency, token and fixed date before comparing p50/p95/p99, throughput, error counts, PostgreSQL query counts, or resource use.

## Verification limitations in the archive-review environment

The supplied archive excludes `.git`. The available Go executable is 1.23.2 while `go.mod` requires Go 1.26.0/toolchain 1.26.9; commands fail before compilation because that toolchain is not installed and network access is unavailable. Docker and Redis server executables are not installed here. Source formatting checks can run, but Go unit/race/vet/build tests, PostgreSQL/Redis integration tests, and live HTTP benchmarks must be run in the target checkout/environment.
