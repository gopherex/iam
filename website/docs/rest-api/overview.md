---
id: overview
title: REST API overview
sidebar_label: Overview
---

# REST API overview

The HTTP API is the single source of truth. This section documents it with real
`curl` examples; the machine-readable contract is
[`openapi/openapi.yaml`](https://github.com/gopherex/iam/blob/master/openapi/openapi.yaml)
(OpenAPI 3.1).

## Base URL

Self-hosted, the base URL is your deployment, e.g. `http://localhost:8080` in dev
or `https://auth.example.com` in production. All paths below are relative to it.

## Namespaces

| Prefix | Audience | Credential |
| --- | --- | --- |
| `/v1/...` | Runtime (end users) | user session + `X-Client-Id` |
| `/v1/projects/{id}/admin/...` | Project admin | `adminToken` |
| `/mgmt/v1/...` | Operator | `masterKey` |
| `/oauth2/...`, `/p/{id}/e/{env}/.well-known/...` | OIDC provider | OAuth client creds |
| `/v1/scim/v2/{connection_id}/...` | SCIM | `scimToken` |

## Conventions

- **JSON in, JSON out.** `Content-Type: application/json` on request bodies.
- **Project & environment** come from the `X-Client-Id` and `X-Environment`
  headers on runtime calls — see [Authentication](/rest-api/authentication).
- **Errors** use a single stable envelope — see [Errors](/rest-api/errors).
  Branch on `error.code`, never on the HTTP status or the localized message.
- **Idempotency.** Create-style POSTs accept an optional `Idempotency-Key` header
  for safe retries.
- **Pagination.** List endpoints return `{ data, next_cursor, has_more }`; pass
  `?cursor=<next_cursor>&limit=<n>` for the next page.

## Where to start

- **[Authentication](/rest-api/authentication)** — headers, bearer vs cookie
  mode, CSRF.
- **[Auth flows](/rest-api/flows)** — the resumable `/v1/auth/flows` state
  machine (recommended integration path).
- **[Runtime endpoints](/rest-api/runtime)** — sign-up/in, OTP, magic link, MFA,
  WebAuthn, OAuth, tokens, sessions.
- **[Admin & operator](/rest-api/admin)** — managing projects, users, config,
  webhooks, and more.
- **[Errors](/rest-api/errors)** — the error envelope and the full code table.
