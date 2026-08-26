---
id: deployment
title: Production deployment
sidebar_label: Deployment
---

# Production deployment

The production artifact is one container plus a Postgres database. Everything
else (orchestration, TLS, secrets) is layered on top.

## Checklist

- [ ] **Postgres** reachable, `IAM_INFRA_POSTGRES_SSLMODE=require` (or
  `verify-full`).
- [ ] **`IAM_SERVICE_HTTP_PUBLIC_URL`** — the absolute public origin clients
      reach IAM at (`https://auth.example.com`). It becomes the OIDC issuer and
      every URL in the discovery document; the service will not start without it,
      and changing it later invalidates issuer pins in every relying party.
- [ ] **`IAM_SERVICE_AUTH_ENCRYPTION_KEY`** — a real `openssl rand -base64 32`
  value, stored in your secret manager, **stable forever** (rotating it orphans
  existing encrypted secrets).
- [ ] **`IAM_SERVICE_AUTH_MASTER_KEY`** — a long random secret; never the dev
  default.
- [ ] **`IAM_SERVICE_HTTP_TRUSTED_PROXIES`** set to your LB/ingress CIDR so
  `X-Forwarded-For` is trusted (otherwise rate limits key off the proxy IP).
- [ ] **`IAM_SERVICE_CORS_ALLOWED_ORIGINS`** set to your app origins if browser
  clients call the API cross-origin.
- [ ] **TLS terminated** at your ingress; forward to `:8080`.
- [ ] **Probes** wired to `:8081` `/healthz/liveness` + `/healthz/readiness`.
- [ ] **`/metrics`** on the same probe port scraped by your monitoring, and not
      exposed publicly by your ingress.

## Image & tags

Pull the published image:

```bash
docker pull ghcr.io/gopherex/iam:1.2.3   # or :latest
```

Releases are tag-driven (`make release` pushes `vX.Y.Z`), which publishes the
Docker image and the `@gopherex/iam-sdk` npm package together — pin the SDK and
the server to the same `X.Y.Z`.

## Health probes

| Probe | Path | Port |
| --- | --- | --- |
| Liveness | `/healthz/liveness` | `8081` (or API port if `PROBE_ADDR==ADDR`) |
| Readiness | `/healthz/readiness` | same |

Readiness fails until Postgres is reachable and migrations have applied, so it
is safe as a rollout gate.

## Migrations

Migrations are **embedded and applied on startup** — no separate migration step
or job. A new image version applies its migrations the first time it boots. See
[Migrations](/self-hosting/migrations) for the mechanics and rollout guidance.

## Scaling

- The server is stateless; run multiple replicas behind your LB.
- Background workers (outbox, GC, webhook-retry, jobs) use Postgres-based
  leasing, so it is safe to run them on every replica — only one holds each
  lease at a time. See [Observability](/self-hosting/observability).
- Size Postgres for your login volume; the app opens a pooled connection per
  replica.

## Kubernetes / Helm / Terraform

No manifests are shipped yet — the target platform isn't fixed. The image plus
the `IAM_*` env contract in [Configuration](/self-hosting/configuration) is
everything an external deployment needs; wrap it in your orchestrator of choice
(a Deployment + Service + Secret + a managed Postgres is enough).

## Zero-downtime rollout

1. Roll out the new image to one replica.
2. Its readiness probe stays down until migrations finish; the LB keeps sending
   traffic to old replicas.
3. Once ready, continue the rolling update.

Because migrations are additive-first, old and new replicas can serve
concurrently during the roll.
