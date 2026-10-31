# Go pprof Development Guide

## Purpose
The Go pprof server provides runtime profiling endpoints for local development and performance investigation.

## Enable pprof
Set APP_ENV=development, PPROF_ENABLED=true, and PPROF_PORT=6060 in your development environment. Restart the API.

## Disable pprof
Set PPROF_ENABLED=false and restart the API.

## Access
http://127.0.0.1:6060/debug/pprof/

The port can be changed using PPROF_PORT.

## Security
- pprof is development-only and cannot be enabled outside APP_ENV=development.
- Docker publishes the host port on 127.0.0.1.
- Never expose pprof through a public interface, reverse proxy, or production deployment.

## Available profiles
- CPU
- Heap and allocations
- Goroutines
- Mutex contention
- Block contention
- Thread creation
