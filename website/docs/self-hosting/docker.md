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
  -e IAM_SERVICE_HTTP_PUBLIC_URL=https://auth.example.com \
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

## Bootstrapping a sibling service in `docker compose`

A common shape: deploy IAM alongside your own product container, and hand that
container an OAuth client secret so it can call IAM as itself. IAM has no
declarative "put this exact secret in" mechanism — a client secret is
sha256-hashed at rest, so there is no plaintext to put in a config file and read
back (see [Config as code](/guides/admin-config)). What it does have is
generate-once-on-create, which a bootstrap step captures and hands off through a
shared volume — the same pattern Terraform, Vault and Kubernetes Secrets all use
for write-only values.

Run a one-shot `iam-bootstrap` container between IAM becoming healthy and your
product starting, and pass the file on:

```yaml
services:
  iam:
    image: ghcr.io/gopherex/iam:latest
    environment:
      IAM_SERVICE_HTTP_PUBLIC_URL: http://iam:8080
      IAM_SERVICE_AUTH_MASTER_KEY: ${MASTER_KEY}
      IAM_SERVICE_AUTH_ENCRYPTION_KEY: ${ENCRYPTION_KEY}
      # ... Postgres vars
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/healthz/readiness"]

  iam-bootstrap:
    image: curlimages/curl
    depends_on:
      iam: { condition: service_healthy }
    volumes: ["bootstrap:/out"]
    entrypoint:
      - sh
      - -c
      - |
        set -eu
        base=http://iam:8080
        project=$(curl -sf -X POST "$base/mgmt/v1/projects" \
          -H "Authorization: Bearer $MASTER_KEY" -H "Content-Type: application/json" \
          -d '{"name":"Acme"}' | jq -r .project.id)
        token=$(curl -sf -X POST "$base/mgmt/v1/projects/$project/admin-tokens" \
          -H "Authorization: Bearer $MASTER_KEY" -H "Content-Type: application/json" \
          -d '{"name":"bootstrap"}' | jq -r .admin_token)
        app=$(curl -sf -X POST "$base/v1/projects/$project/admin/apps" \
          -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
          -d '{"name":"product","type":"web","redirect_uris":["https://app.example.com/callback"]}' \
          | jq -r .app.id)
        curl -sf -X POST "$base/v1/projects/$project/admin/apps/$app/secrets" \
          -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
          -d '{"name":"prod"}' | jq -r .client_secret > /out/client_secret
        echo "$project" > /out/project_id
        echo "$app" > /out/client_id

  product:
    depends_on:
      iam-bootstrap: { condition: service_completed_successfully }
    volumes: ["bootstrap:/secrets:ro"]
    entrypoint:
      - sh
      - -c
      - export IAM_CLIENT_ID=$(cat /secrets/client_id) IAM_CLIENT_SECRET=$(cat /secrets/client_secret); exec ./product

volumes:
  bootstrap:
```

Notes:

- `condition: service_completed_successfully` needs Compose v2.20+ / Docker
  Engine 24+; on older Compose, poll for the file's existence instead of
  `depends_on`.
- Run `iam-bootstrap` only once (a fresh project every run is not what you
  want): guard it — check whether a project with the expected name/slug
  already exists before creating one, or run it as a one-off (`docker compose
  run`) rather than every `up`.
- The secrets endpoints are ordinary [admin API](/rest-api/admin) calls;
  nothing above is bootstrap-specific. Swap `curl`/`jq` for the [TypeScript
  SDK](/sdk/typescript) or [Go SDK](/sdk/go) in a real init container if you'd
  rather not shell out.
- This mints a **new** secret every run of the bootstrap step. An app client
  can hold several live secrets at once (rotate in, then delete the old one)
  — see [Machine identity](/guides/machine-identity).

## Frontend iteration (no image rebuild)

Run the API on the host and the Vite dev server (proxies `/v1` + `/mgmt`):

```bash
IAM_SERVICE_HTTP_PUBLIC_URL=http://localhost:8080 \
IAM_INFRA_POSTGRES_SSLMODE=disable \
IAM_SERVICE_AUTH_SEED_ROOT=true \
IAM_SERVICE_AUTH_MASTER_KEY=dev \
IAM_SERVICE_AUTH_ENCRYPTION_KEY="$(openssl rand -base64 32)" \
go run ./cmd/iam
# in another shell:
cd web && yarn dev   # http://localhost:5173
```
