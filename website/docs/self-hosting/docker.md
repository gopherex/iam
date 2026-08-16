---
id: docker
title: Run with Docker
sidebar_label: Docker
---

# Run with Docker

IAM ships as a single static binary in a `distroless` image. The same image runs
locally (docker compose) and in production (configured purely via `IAM_*`
environment variables). Migrations apply on boot; the admin SPA is embedded.

## Local dev stack (fastest start)

The repo-root `docker-compose.yml` brings up Postgres 17 + the IAM server, seeds
a `Root` project, and applies migrations:

```bash
docker compose up
```

Then open [http://localhost:8080](http://localhost:8080) and sign in with the
master key `dev`.

| Service | Host port | Notes |
| --- | --- | --- |
| IAM API + admin SPA | `127.0.0.1:8080` | probes mounted on the same port |
| Postgres 17 | `127.0.0.1:5436` | maps to container `5432` |

:::warning Dev defaults are not production-safe
Compose defaults to master key `dev` and a **committed** AES encryption key.
Never use these outside localhost — override every `IAM_SERVICE_AUTH_*` value.
:::

## Build the image

Multi-stage build (SDK → admin SPA → Go binary with SPA embedded), run from the
repo root:

```bash
docker build -f deployments/Dockerfile -t iam .
```

Optionally inject build metadata (surfaced in startup logs and the outbox lease
owner):

```bash
docker build -f deployments/Dockerfile -t iam \
  --build-arg VERSION=1.2.3 \
  --build-arg COMMIT="$(git rev-parse HEAD)" \
  --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
```

Official images are published to `ghcr.io/gopherex/iam` (`X.Y.Z` + `latest`).

## Run the image (production shape)

Point it at a reachable Postgres and supply the two required secrets:

```bash
docker run --rm -p 8080:8080 -p 8081:8081 \
  -e IAM_INFRA_POSTGRES_HOST=db \
  -e IAM_INFRA_POSTGRES_PASSWORD=... \
  -e IAM_INFRA_POSTGRES_SSLMODE=require \
  -e IAM_SERVICE_AUTH_MASTER_KEY="$(openssl rand -hex 32)" \
  -e IAM_SERVICE_AUTH_ENCRYPTION_KEY="$(openssl rand -base64 32)" \
  ghcr.io/gopherex/iam:latest
```

Ports: `8080` = API + admin SPA, `8081` = liveness/readiness probes
(`/healthz/liveness`, `/healthz/readiness`). Set
`IAM_SERVICE_HTTP_PROBE_ADDR=:8080` to mount probes on the API port instead
(single listener).

Full env-var contract: [Configuration](/self-hosting/configuration).

## Frontend iteration (no image rebuild)

Run the API on the host and the Vite dev server (proxies `/v1` + `/mgmt`):

```bash
IAM_INFRA_POSTGRES_SSLMODE=disable \
IAM_SERVICE_AUTH_SEED_ROOT=true \
IAM_SERVICE_AUTH_MASTER_KEY=dev \
IAM_SERVICE_AUTH_ENCRYPTION_KEY="$(openssl rand -base64 32)" \
go run ./cmd/iam
# in another shell:
cd web && yarn dev   # http://localhost:5173
```
