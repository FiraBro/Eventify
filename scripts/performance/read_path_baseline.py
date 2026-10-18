#!/usr/bin/env python3
"""Reproducible HTTP baseline runner for the three pre-cache read paths.

Only sends GET requests. Supply a disposable/test API, a valid bearer token for
protected endpoints, and an existing service UUID/date for availability.
"""
import argparse
import concurrent.futures
import datetime as dt
import json
import math
import os
import platform
import statistics
import sys
import time
import urllib.error
import urllib.request


def percentile(values, p):
    if not values:
        return None
    ordered = sorted(values)
    # Nearest-rank percentile, documented for reproducibility.
    return ordered[max(0, math.ceil((p / 100) * len(ordered)) - 1)]


def request_once(url, token, timeout):
    headers = {"Accept": "application/json", "User-Agent": "local-go-c1-baseline/1.0"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(url, headers=headers, method="GET")
    started = time.perf_counter_ns()
    status, error = None, None
    try:
        with urllib.request.urlopen(req, timeout=timeout) as response:
            status = response.status
            response.read()
    except urllib.error.HTTPError as exc:
        status = exc.code
        try:
            exc.read()
        except Exception:
            pass
        error = "http_error"
    except Exception as exc:
        error = type(exc).__name__
    elapsed_ms = (time.perf_counter_ns() - started) / 1_000_000
    return {"latency_ms": elapsed_ms, "status": status, "error": error,
            "ok": status is not None and 200 <= status < 300 and error is None}


def summarize(results):
    latencies = [r["latency_ms"] for r in results]
    statuses = {}
    errors = {}
    for result in results:
        if result["status"] is not None:
            key = str(result["status"])
            statuses[key] = statuses.get(key, 0) + 1
        if result["error"]:
            errors[result["error"]] = errors.get(result["error"], 0) + 1
    total_seconds = sum(latencies) / 1000
    return {
        "requests": len(results), "successes": sum(r["ok"] for r in results),
        "failures": sum(not r["ok"] for r in results), "statuses": statuses,
        "errors": errors, "p50_ms": percentile(latencies, 50),
        "p95_ms": percentile(latencies, 95), "p99_ms": percentile(latencies, 99),
        "mean_ms": statistics.fmean(latencies) if latencies else None,
        "completed_ops_per_second": len(results) / total_seconds if total_seconds else None,
        "raw_latency_ms": latencies,
    }


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--base-url", default=os.getenv("BASE_URL", "http://127.0.0.1:8080"))
    p.add_argument("--token", default=os.getenv("BASELINE_TOKEN", ""))
    p.add_argument("--availability-path", default=os.getenv("AVAILABILITY_PATH", ""),
                   help="Exact existing GET path, including service/staff ID and date query")
    p.add_argument("--services-path", default="/api/v1/services")
    p.add_argument("--settings-path", default="/api/v1/settings")
    p.add_argument("--concurrency", type=int, default=8)
    p.add_argument("--warmup", type=int, default=30, help="warm-up requests per endpoint")
    p.add_argument("--requests", type=int, default=300, help="measured requests per endpoint per run")
    p.add_argument("--runs", type=int, default=3)
    p.add_argument("--timeout", type=float, default=5.0)
    p.add_argument("--output", default="baseline-results.json")
    args = p.parse_args()
    if args.concurrency < 1 or args.warmup < 0 or args.requests < 1 or args.runs < 1 or args.timeout <= 0:
        p.error("concurrency/requests/runs must be positive; warmup nonnegative; timeout positive")
    if not args.availability_path:
        p.error("--availability-path is required; use an existing /api/v1/availability/services/:serviceId?date=YYYY-MM-DD or /staff/:staffId?date=YYYY-MM-DD path")

    endpoints = {
        "availability": args.availability_path,
        "services_list": args.services_path,
        "settings": args.settings_path,
    }
    base = args.base_url.rstrip("/")
    for name, path in endpoints.items():
        if not path.startswith("/"):
            p.error(f"{name} path must begin with /")

    def hit(name):
        return request_once(base + endpoints[name], args.token, args.timeout)

    output = {
        "schema_version": 1, "created_at_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
        "base_url": base, "endpoints": endpoints,
        "workload": {"concurrency": args.concurrency, "warmup_requests_per_endpoint": args.warmup,
                     "measured_requests_per_endpoint_per_run": args.requests, "runs": args.runs,
                     "timeout_seconds": args.timeout, "method": "GET", "percentile_method": "nearest-rank"},
        "environment": {"python": sys.version, "platform": platform.platform()}, "results": {},
        "notes": ["latencies are client-observed end-to-end HTTP timings", "database query counts and CPU/memory are not measured by this client harness"]
    }
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as pool:
        for name in endpoints:
            warm = list(pool.map(lambda _: hit(name), range(args.warmup)))
            output["results"][name] = {"warmup": summarize(warm), "runs": []}
            for run_index in range(args.runs):
                start = time.perf_counter()
                results = list(pool.map(lambda _: hit(name), range(args.requests)))
                elapsed = time.perf_counter() - start
                summary = summarize(results)
                summary["wall_seconds"] = elapsed
                summary["wall_clock_ops_per_second"] = len(results) / elapsed if elapsed else None
                output["results"][name]["runs"].append(summary)
                print(f"{name} run {run_index + 1}/{args.runs}: {summary['successes']}/{summary['requests']} ok; "
                      f"p50={summary['p50_ms']:.3f}ms p95={summary['p95_ms']:.3f}ms "
                      f"p99={summary['p99_ms']:.3f}ms wall_rps={summary['wall_clock_ops_per_second']:.2f}")
    with open(args.output, "w", encoding="utf-8") as f:
        json.dump(output, f, indent=2)
        f.write("\n")
    print(f"JSON results written to {args.output}")
    failures = sum(run["failures"] for endpoint in output["results"].values() for run in endpoint["runs"])
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())