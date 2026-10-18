package performance

# C2 — Redis cache-aside implementation

## Exact scope

- `GET /api/v1/availability/services/:serviceId?date=YYYY-MM-DD` — 15-second TTL.
- `GET /api/v1/services` — 2-minute TTL.
- `GET /api/v1/settings` — 1-minute TTL.

No other read path is cached. Booking writes, cancellation, payment/refund/webhook paths and booking-conflict checks are not cached. PostgreSQL remains authoritative for booking validation.

## Keys and freshness

All keys use explicit `local-go:<resource>:...:v1:` namespaces. Availability's SHA-256 key input includes both service ID and date; the raw inputs are not stored in Redis keys. Services list and singleton business settings have fixed resource keys because their existing routes have no result-affecting query parameters or tenant selector. Versioned namespaces allow a future serialization/key-contract change without reading incompatible entries.

C2 intentionally adds no write invalidation. Until C3 lands, successful service mutations may leave the service list stale for up to 2 minutes; settings updates may leave settings stale for up to 1 minute; schedule, holiday, and staff-service assignment changes may leave service availability stale for up to 15 seconds. Expired entries are lazily replaced on reads. This is a temporary bounded-staleness policy, not a freshness guarantee. Never use cached availability to accept or guarantee a booking.

## Failure and concurrency behavior

Redis read failures bypass cache coordination and use the existing source path. Misses are coalesced per key in-process with `singleflight`; the operation rechecks Redis after joining. The shared fill uses a 10-second bounded context detached from any one request's cancellation, while each waiter can stop waiting when its own context is canceled. This is process-local only, not cross-instance coordination. Redis writes and JSON encoding failures do not fail successful source reads. Malformed cached JSON is treated as a miss. Cache errors are not stored.

## C1 baseline preservation and comparison

The C1 files remain unchanged:

- `scripts/performance/read_path_baseline.py`
- `docs/performance/read-path-baseline.md`

C1's documentation says no pre-cache results have yet been collected. No before/after comparison can be calculated until the baseline and post-cache runs are collected against the same live API, dataset, machine, service ID/date, and settings. Use the same command documented in `read-path-baseline.md`, with distinct output paths, and repeat at least three times. Capture PostgreSQL metrics and Redis availability/configuration separately if available. Do not interpret the unit tests as performance evidence.

## Verification limitations in this execution

The supplied project archive has no `.git` directory. The available Go executable is 1.23.2, but `go.mod` requires Go 1.26.9; attempting to invoke the module caused an automatic toolchain download, which failed because network/DNS is unavailable. Docker is not installed in this execution environment. Therefore Go tests, race tests, vet, build, database/Redis integration tests, and live HTTP benchmarks could not be run here. This document records limitations rather than inventing measurements.
