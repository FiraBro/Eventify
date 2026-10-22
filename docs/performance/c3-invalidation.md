# C3 — Cache invalidation review record

## Implementation

C3 adds best-effort generation-token invalidation for only the existing C2 cache families: service availability, services list, and settings. It does not change endpoints, HTTP contracts, cache TTLs, database schema, or the C1 benchmark harness. No Redis key scan is used.

Generation tokens are shared through Redis. Each cache lookup resolves a token before reading/filling a token-scoped key. Successful mutations replace the token after the repository operation returns successfully. Old-generation fills can complete but are unreachable to later lookups that see the new token. Already-running requests may still complete against the generation they previously read.

## Mutation coverage

- Service create/update/delete: services list + service availability.
- Booking create/delete/status transitions and slot/staff changes: service availability.
- Staff delete, service assignments, schedule replacement and holiday additions: service availability.
- Settings updates with actual changes: settings.
- Failed mutations, validation failures, and no-op settings updates: no invalidation.

Transactional service assignment and schedule replacement invalidate only after their repository methods return from successful commit. Basic staff profile edits do not affect the cached slot computation and are not invalidated.

## Failure semantics

Redis invalidation is best-effort and does not turn a successful database mutation into an API error. Failures emit a namespace-only warning. If invalidation fails, the previous entry may remain visible until its TTL expires (15 seconds availability, 2 minutes services list, 1 minute settings). This is not strict consistency and no outbox/schema change was added.

## Verification status

- `gofmt`: completed for changed Go files.
- `go test ./internal/cacheaside ./internal/services`: passed.
- `go test ./...`: passed.
- `go test -race ./...`: passed.
- `go vet ./...`: completed without reported errors.
- `git diff --check`: completed without reported errors.
- Integration suite (`go test -tags=integration -count=1 ./tests/integration`): passed in a previous run.
- Real Redis cross-client invalidation test: passed in a previous run with `REDIS_URL` configured.
- C2 performance benchmark rerun: not yet performed; no performance improvement is claimed.

Some successful test commands reused Go's test cache. Run final checks with `-count=1` before merging. Cache invalidation remains best-effort: Redis failures can leave old entries visible until their TTL expires, and already-running requests may finish using a previously resolved generation.
