# API observability and lifecycle

## Structured logs and request correlation

The server writes structured JSON logs to stdout using Go's `log/slog`. `LOG_LEVEL` accepts `debug`, `info`, `warn`, or `error` and defaults to `info`. Standard-library `log` calls are redirected through the default slog logger after startup configuration is loaded.

Every HTTP request receives an `X-Request-ID` response header. A supplied request ID is accepted only when it is a canonical UUID; otherwise the server generates a new UUID. The ID is stored in the request context and automatically added to context-aware slog records. Request-completion logs include the method, Gin route pattern, status, duration, and request ID. Raw URLs, query strings, headers, and bodies are not logged by this middleware. Health checks and `/metrics` are excluded from completion logs to avoid probe/scrape noise.

## Health and readiness

- `GET /healthz` returns `200 {"status":"ok"}` when the process can serve HTTP. It does not query dependencies.
- `GET /readyz` returns `200 {"status":"ready"}` only when PostgreSQL answers a bounded `PingContext`; otherwise it returns `503 {"status":"not_ready"}`.
- Redis is deliberately excluded from readiness because the application treats Redis as an optional, fail-open optimization. Redis outages should not trigger deployment restart loops by themselves.

Readiness checks use `HEALTH_CHECK_TIMEOUT` (default `2s`). Responses never include dependency errors or connection details.

## Prometheus metrics

`GET /metrics` exposes Prometheus text format. HTTP request counters and duration histograms use only normalized HTTP methods, Gin route patterns, and status codes. The in-flight gauge is `http_requests_in_flight`. `http_errors_total` counts responses with status codes of 400 or higher. `/metrics` does not instrument itself. No user identifiers, raw paths, query parameters, or request IDs are metric labels.

Scrape the endpoint only from a trusted monitoring network or behind the deployment's existing network access controls. The endpoint intentionally does not add a separate authentication scheme or expose business data.

## HTTP server shutdown

The server uses bounded read-header, read, write, and idle timeouts. On `SIGINT` or `SIGTERM`, it stops accepting new connections and waits for active requests up to `SHUTDOWN_TIMEOUT` (default `10s`). If graceful shutdown expires, the server is force-closed and the application context is cancelled. PostgreSQL and the initialized Redis client are closed after the HTTP server has drained or been force-closed.

Existing timeout settings are `READ_HEADER_TIMEOUT` (`5s`), `READ_TIMEOUT` (`10s`), and `WRITE_TIMEOUT` (`15s`). All timeout values must be positive Go duration strings, such as `5s` or `1500ms`.

## Environment additions

Add these entries to `.env.example` if they are not already present:

```dotenv
LOG_LEVEL=info
IDLE_TIMEOUT=60s
HEALTH_CHECK_TIMEOUT=2s
SHUTDOWN_TIMEOUT=10s
```

See this document from the repository README under the operational/deployment documentation section.
