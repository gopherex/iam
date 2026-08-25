---
id: configuration
title: Configuration
sidebar_label: Configuration
---

# Configuration

IAM reads a config file (`config.yaml`/`.json`/`.toml`) from `$CONFIG_PATH` (a
directory or an exact file), and **every key can be overridden by an environment
variable**. The env name is the upper-snake path prefixed with `IAM_`:

```
service.auth.encryption_key   →   IAM_SERVICE_AUTH_ENCRYPTION_KEY
infra.postgres.password       →   IAM_INFRA_POSTGRES_PASSWORD
```

Env variables win over file values, so production can run the image with **no
config file at all** — just `IAM_*` env.

## Required in production

| Variable | Purpose |
| --- | --- |
| `IAM_SERVICE_HTTP_PUBLIC_URL` | **absolute base URL this deployment is reachable at** (e.g. `https://auth.example.com`, no trailing slash). It is the OIDC issuer prefix and every absolute URL in the discovery document. The service refuses to start without it. |
| `IAM_SERVICE_AUTH_ENCRYPTION_KEY` | **base64 32-byte AES-256 key**; encrypts reversible secrets at rest. The service refuses to start without it. |
| `IAM_SERVICE_AUTH_MASTER_KEY` | operator credential for the admin panel + `/mgmt/v1/*` API. Empty disables the `masterKey` scheme. |
| `IAM_INFRA_POSTGRES_*` | Postgres connection (below) |

Generate the encryption key once and keep it secret and stable — rotating it
without re-encrypting existing secrets makes them unreadable:

```bash
openssl rand -base64 32
```

## Postgres

| Variable | Default | Notes |
| --- | --- | --- |
| `IAM_INFRA_POSTGRES_HOST` | `localhost` | |
| `IAM_INFRA_POSTGRES_PORT` | `5432` | |
| `IAM_INFRA_POSTGRES_USERNAME` | `iam` | |
| `IAM_INFRA_POSTGRES_PASSWORD` | `iam` | override in prod |
| `IAM_INFRA_POSTGRES_DATABASE` | `iam` | |
| `IAM_INFRA_POSTGRES_SSLMODE` | `require` | `disable\|require\|verify-ca\|verify-full` |
| `IAM_INFRA_POSTGRES_LOG_LEVEL` | `info` | `debug\|info\|warn\|error` |

## HTTP server

| Variable | Default | Notes |
| --- | --- | --- |
| `IAM_SERVICE_HTTP_ADDR` | `:8080` | API + admin SPA |
| `IAM_SERVICE_HTTP_PUBLIC_URL` | — | **required.** Public origin (+ optional path prefix) clients reach IAM at. Issuer = `{public_url}/p/{project_id}/e/{env}`. Deliberately never derived from `Host` / `X-Forwarded-*`: a request header must not be able to decide the issuer clients pin. |
| `IAM_SERVICE_HTTP_PROBE_ADDR` | `:8081` | probe listener; set `=ADDR` (or `:8080`) to mount `/healthz/*` on the API port |
| `IAM_SERVICE_HTTP_READ_TIMEOUT_SEC` | `15` | |
| `IAM_SERVICE_HTTP_WRITE_TIMEOUT_SEC` | `30` | |
| `IAM_SERVICE_HTTP_SHUTDOWN_SEC` | `15` | graceful drain |
| `IAM_SERVICE_HTTP_TRUSTED_PROXIES` | `[]` | CIDRs/IPs of your LB/ingress to trust for `X-Forwarded-For`. **Empty = use the real TCP peer**, so clients can't spoof IPs to dodge IP-keyed rate limits. Set this when behind a proxy. |

## Logging & CORS

| Variable | Default | Notes |
| --- | --- | --- |
| `IAM_SERVICE_LOGGER_LEVEL` | `info` | `debug\|info\|warn\|error` |
| `IAM_SERVICE_LOGGER_FORMAT` | `json` | `json\|text` |
| `IAM_SERVICE_CORS_ALLOWED_ORIGINS` | `[]` | `*` or comma-separated origins |

## Dev conveniences

| Variable | Default | Notes |
| --- | --- | --- |
| `IAM_SERVICE_AUTH_SEED_ROOT` | `false` | ensure a `Root` project exists on startup so the operator has something to manage |

## Browser session lifetime

The session the hosted provider screens establish is a pair of HttpOnly cookies,
and both lifetimes come from the project's `session_policy`, not from the
service config:

| Cookie | Max-Age | Scope |
| --- | --- | --- |
| `iam_session` | `session_policy.access_ttl` (default 10m) | `/` |
| `iam_refresh` | `session_policy.refresh_ttl` (default 30d) | `/v1/auth/token/refresh` |

The access cookie is short by design; the refresh cookie is what keeps a person
signed in, so **`refresh_ttl` is the knob that decides how often an operator is
sent back to a login screen**. Raise it for an internal console; lower it for
anything handling sensitive data. Both are set per project (and per environment)
through the admin API, so `live` and `test` can differ.

## Note on TTLs and environment

Token TTLs are **not** service config — access/refresh TTLs come from each
project's `session_policy` config doc (set via the admin API), and the acting
environment is chosen per request via the `X-Environment` header. See
[Projects & environments](/concepts/projects-environments) and
[Admin & config](/guides/admin-config).

## Example config file

```yaml
# config.yaml — every key is also an IAM_* env var
infra:
  postgres:
    host: db
    password: ${POSTGRES_PASSWORD}
    sslmode: require
service:
  http:
    addr: ":8080"
    public_url: "https://auth.example.com"   # required; OIDC issuer base
    trusted_proxies: ["10.0.0.0/8"]
  auth:
    master_key: ""          # prefer IAM_SERVICE_AUTH_MASTER_KEY
    encryption_key: ""      # prefer IAM_SERVICE_AUTH_ENCRYPTION_KEY
```
