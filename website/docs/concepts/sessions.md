---
id: sessions
title: Sessions & tokens
sidebar_label: Sessions & tokens
---

# Sessions & tokens

## Token pair

A successful sign-in yields a **`SessionTokens`** pair:

```json
{ "access_token": "eyJ...", "refresh_token": "rt_...", "expires_in": 1800, "token_type": "Bearer" }
```

- **Access token** — a short-lived signed **RS256 JWT** (default **10 minutes**).
  Verify it offline against the project JWKS; it carries `sub`, `sid`, `aal`,
  `amr`, `pid`, `env` (see [Principals](/concepts/principals)).
- **Refresh token** — a long-lived opaque secret (default **30 days**), stored
  **only as a hash** at rest.

## Sliding refresh

`POST /v1/auth/token/refresh` rotates the refresh token and issues a fresh access
token **without creating a new device session** — the session keeps its identity,
device metadata and creation time. Continuous use keeps a session alive
indefinitely up to the policy's absolute timeout.

Refresh-token **reuse is always rejected**: presenting an already-rotated refresh
token revokes the whole session chain (theft detection), independent of policy.

## AAL — assurance levels

A session records the **authenticator assurance level**:

- **AAL1** — a single factor (e.g. password or a passwordless method).
- **AAL2** — a second factor was verified (TOTP, WebAuthn, email/SMS code,
  recovery code).

The session also carries `amr` (methods used), `trusted`, and `current` flags.
Sensitive actions can require a higher level via **step-up**:

```bash
curl -sX POST https://auth.example.com/v1/auth/session/step-up \
  -H "X-Client-Id: prj_7Fk2" -H "Authorization: Bearer eyJ..." \
  -H "Content-Type: application/json" \
  -d '{"purpose":"delete_account","required_aal":2}'
```

## Session policy (per project, per environment)

The `session_policy` config doc controls token lifetimes and session behavior:

| Field | Meaning |
| --- | --- |
| `access_ttl` | access-token lifetime (default 10m) |
| `refresh_ttl` | refresh-token lifetime (default 30d) |
| `idle_timeout` | revoke after inactivity |
| `absolute_timeout` | hard cap on session age |
| `reuse_detection` | refresh-token reuse response (reuse is rejected regardless) |

Manage it at `PUT /v1/projects/{id}/admin/config/session-policy`.

## Device sessions

Each sign-in creates a **device session** the user can see and manage:

```bash
curl https://auth.example.com/v1/sessions -H "X-Client-Id: prj_7Fk2" -H "Authorization: Bearer eyJ..."
```

- `GET /v1/sessions`, `GET /v1/sessions/current`
- `DELETE /v1/sessions/{id}` — revoke one
- `POST /v1/sessions/{id}/trust` — mark trusted (skip repeated MFA)
- `POST /v1/auth/sign-out` / `POST /v1/auth/sign-out-all` — revoke this / all

Optional `X-Device-Name` and `X-Device-Fingerprint` headers on sign-in are
persisted with the session; the fingerprint is a refresh-theft signal.

## Introspection & verification

Resource servers can validate tokens:

- `POST /v1/tokens/introspect` — `{ active, ... }` (RFC 7662 style)
- `POST /v1/tokens/verify` — `{ valid, claims, error }` (signature + expiry + audience)
- `POST /v1/tokens/revoke` — revoke by token or session id
- `GET /v1/tokens/current` — claims for the presented bearer

Prefer **offline JWKS verification** for the access token on the hot path;
introspect only when you need live revocation checks.
