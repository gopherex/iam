---
id: observability
title: Observability & workers
sidebar_label: Observability
---

# Observability & workers

IAM is instrumented with OpenTelemetry (traces, metrics, logs) and runs its
asynchronous work in Postgres-leased background workers.

## OpenTelemetry

The server initializes OTel through `xtrace` using the **standard OTel
environment variables** — no bespoke config. Point it at a collector:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel-collector:4318
OTEL_EXPORTER_OTLP_HEADERS=authorization=Bearer%20...
# per-signal endpoints/headers/timeouts/compression/TLS are all supported:
#   OTEL_EXPORTER_OTLP_TRACES_ENDPOINT, OTEL_EXPORTER_OTLP_METRICS_ENDPOINT,
#   OTEL_EXPORTER_OTLP_LOGS_ENDPOINT, OTEL_EXPORTER_OTLP_*_HEADERS, ...
```

When an endpoint is set, the server exports:

- **traces** — server request spans,
- **metrics** — generated OAS HTTP metrics, Postgres metrics, host/runtime
  metrics,
- **logs** — correlated with trace IDs.

`service.name` / `version` / `instance.id` are derived from build metadata; add
extra resource attributes with `OTEL_RESOURCE_ATTRIBUTES`. Leave the endpoint
unset to disable export (the SDK treats empty values as unset).

## Logs

Without an OTLP endpoint, logs go to stdout. Control them with:

| Variable | Default | Notes |
| --- | --- | --- |
| `IAM_SERVICE_LOGGER_LEVEL` | `info` | `debug\|info\|warn\|error` |
| `IAM_SERVICE_LOGGER_FORMAT` | `json` | `json` for aggregation, `text` for local |

Every error response carries a `request_id` (see [Errors](/rest-api/errors)) —
correlate it against the request span/log.

## Health probes

There are two health surfaces, for two different consumers.

| Path | Port | For |
| --- | --- | --- |
| `/healthz/liveness` | `:8081` | orchestrator liveness — the process is up |
| `/healthz/readiness` | `:8081` | orchestrator readiness — Postgres reachable, migrations applied |
| `/v1/health` | `:8080` | public aggregate check, returns `{status, time, version}` |
| `/v1/health/live` | `:8080` | same as liveness, over the public API |
| `/v1/health/ready` | `:8080` | readiness with a per-dependency `checks[]` breakdown |

Point orchestrator probes at `/healthz/*`: it stays on its own listener, so a
saturated API port cannot make a healthy process look dead, and the port need not
be exposed. Use `/v1/health*` from outside, where only the API is reachable —
`/v1/health/ready` is the one that names which dependency is failing.

Set `IAM_SERVICE_HTTP_PROBE_ADDR` equal to the API address to collapse both onto
one listener.

## Background workers

Asynchronous work runs in-process, coordinated by Postgres leasing so it is
safe on every replica (only one holds each lease at a time):

| Worker | Responsibility |
| --- | --- |
| **outbox** | reliably dispatches emitted events / webhook deliveries (transactional outbox) |
| **webhook-retry** | re-attempts failed webhook deliveries with backoff |
| **GC** | reaps expired sessions, flows, challenges, tokens |
| **jobs** | runs async admin jobs (user import, data export, audit export) |

The outbox lease owner is stamped with the build metadata baked into the image,
which is handy for pinpointing which replica is dispatching.

## What to watch

- **Readiness flapping** → Postgres connectivity or migration failure.
- **Growing outbox backlog** → webhook endpoint down, or worker not running.
- **429 spikes** → rate limits; check `IAM_SERVICE_HTTP_TRUSTED_PROXIES` is set
  correctly so limits key off the real client IP, not your proxy.
