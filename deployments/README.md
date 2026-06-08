# deployments/

Production deployment material for IAM.

## Image

[`Dockerfile`](Dockerfile) is a multi-stage build (run from the repo root):

```sh
docker build -f deployments/Dockerfile -t iam .
```

It builds the TypeScript SDK + admin SPA, compiles the Go server with the SPA
embedded (`-tags embed`), and ships a minimal static `distroless` image. The
single binary serves the API and the admin SPA on `:8080` and probes on `:8081`
(`/healthz/liveness`, `/healthz/readiness`).

## Release

Run `make release` from a clean worktree. It bumps the single published npm
package (`@gopherex/iam-sdk`) to the selected `X.Y.Z`, commits `release vX.Y.Z`,
and pushes tag `vX.Y.Z`.

Pushing the tag triggers GitHub Actions release publishing:

| Artifact | Tags / version |
|---|---|
| Docker image `ghcr.io/gopherex/iam` | `X.Y.Z` and `latest` |
| npm package `@gopherex/iam-sdk` | `X.Y.Z` |

## Configuration

Everything is configured via `IAM_*` environment variables (see
[`config.example.yaml`](../config.example.yaml) for every key; the env name is
the upper-snake path, e.g. `service.auth.master_key` → `IAM_SERVICE_AUTH_MASTER_KEY`).
Minimum for production:

| Variable | Purpose |
|---|---|
| `IAM_INFRA_POSTGRES_{HOST,PORT,USERNAME,PASSWORD,DATABASE}` | Postgres connection |
| `IAM_INFRA_POSTGRES_SSLMODE` | `require` / `verify-full` in production |
| `IAM_SERVICE_HTTP_ADDR` | API listen address (default `:8080`) |
| `IAM_SERVICE_HTTP_PROBE_ADDR` | probe listen address (default `:8081`; set `=ADDR` to mount under `/healthz/` on the API port) |
| `IAM_SERVICE_AUTH_MASTER_KEY` | operator master key (admin panel + operator API) |
| `IAM_SERVICE_AUTH_ENCRYPTION_KEY` | base64 32-byte AES-256 key for secrets at rest |

Run it (Postgres reachable, migrations apply on boot):

```sh
docker run --rm -p 8080:8080 \
  -e IAM_INFRA_POSTGRES_HOST=db -e IAM_INFRA_POSTGRES_PASSWORD=... \
  -e IAM_SERVICE_AUTH_MASTER_KEY=... -e IAM_SERVICE_AUTH_ENCRYPTION_KEY=... \
  iam
```

## Local development

The repo-root [`docker-compose.yml`](../docker-compose.yml) is the dev stack:
`docker compose up` brings up Postgres and the IAM server (SPA embedded,
migrations applied, a `Root` project seeded). Open <http://localhost:8080> and
sign in with the master key (`dev-master-key` by default, overridable via
`IAM_MASTER_KEY`).

For frontend iteration without rebuilding the image, run the API on the host and
the Vite dev server (which proxies `/v1` + `/mgmt` to it):

```sh
IAM_SERVICE_AUTH_SEED_ROOT=true IAM_SERVICE_AUTH_MASTER_KEY=dev go run ./cmd/iam
cd web && yarn dev   # http://localhost:5173
```

## Orchestration

Kubernetes / Helm / Terraform manifests are intentionally not included yet — the
target platform is not fixed. The image + the env contract above are everything
an external deployment needs; orchestration is layered on top when the platform
is chosen.
