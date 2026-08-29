---
id: admin-config
title: Configuring a project
sidebar_label: Admin & config
---

# Configuring a project

Project behaviour — enabled auth methods, password/session/MFA policy,
registration mode, providers, webhooks — is controlled through the
**project-admin API** (`/v1/projects/{project_id}/admin/*`), authenticated with
an `adminToken`. Full surface in the [Admin API reference](/rest-api/admin).

Get an `adminToken` from the operator plane
(`POST /mgmt/v1/projects/{id}/admin-tokens`) — see the
[Operator guide](/self-hosting/operator).

## Desired state (plan & apply)

Every configuration document can also be read and written as one object, so an
external applicator does not have to GET, diff and PATCH each document itself.

```bash
# read the whole configuration
curl -s https://auth.example.com/v1/projects/prj_7Fk2/admin/config \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# plan: return the change set, write nothing
curl -sX PUT 'https://auth.example.com/v1/projects/prj_7Fk2/admin/config?dry_run=true' \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"auth":{"methods":["email"]},"password_policy":{"min_length":12}}'

# apply: all documents in one transaction, all-or-nothing
curl -sX PUT https://auth.example.com/v1/projects/prj_7Fk2/admin/config \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"auth":{"methods":["email"]},"password_policy":{"min_length":12}}'
```

Each document is validated exactly as its own endpoint validates it, and every
document is validated **before** anything is written, so a bundle that is bad
anywhere leaves the project untouched instead of half-applied. Documents you
omit are left alone.

App clients have the same shape:

```bash
# reconcile the client list; prune=true also deletes clients you left out
curl -sX PUT 'https://auth.example.com/v1/projects/prj_7Fk2/admin/clients?prune=true' \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"clients":[{"id":"app_web","name":"Web","type":"spa",
                   "redirect_uris":["https://app.example.com/cb"]}]}'
```

Clients are matched by `id`, and an `id` you supply is honoured on create, so a
repeated apply of the same file is a no-op. `prune` is off by default: a partial
list cannot delete clients it does not know about. Both PUTs accept
`?dry_run=true` and an `Idempotency-Key` header. The response lists what was
created, updated and deleted, with `before`/`after` for each object.

The document describes configuration, not credentials. Client secrets and the
registration access token of a
[self-registered client](/concepts/oidc-federation) are carried through an
apply untouched — otherwise every IaC run would silently revoke them. `prune`
does delete a self-registered client that is missing from the list, which is the
point of asking for `prune`.

## Roles

Roles are plain labels assigned per user per environment, and they are what the
OIDC `groups` scope projects into the `groups` claim:

```bash
curl -sX PUT https://auth.example.com/v1/projects/prj_7Fk2/admin/users/usr_9/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: live" \
  -H "Content-Type: application/json" \
  -d '{"roles":["ops","platform:admin"]}'
```

The PUT is a replacement: the user ends up with exactly the roles listed. See
[Roles in the token](/concepts/oidc-federation).

## The configuration documents

Five documents make up a project's behaviour. Each has its own
`GET`/`PATCH` endpoint under `admin/config/{document}`, and all five are what
the bulk `GET`/`PUT admin/config` above reads and writes.

Every document is parsed **strictly**: an unknown key is a `422`, not a silently
dropped field. A setting whose engine does not exist is refused rather than
accepted as a no-op — there is no way to switch on a security control that would
do nothing.

| Document | Endpoint | Controls |
| --- | --- | --- |
| `auth` | `config/auth` | which sign-in methods exist, registration, locales |
| `password_policy` | `config/password-policy` | password strength |
| `session_policy` | `config/session-policy` | token and session lifetimes |
| `mfa_policy` | `config/mfa-policy` | which factors, and who must use one |
| `rate_limits` | `config/rate-limits` | per-endpoint request limits |

### `auth`

```bash
curl -sX PATCH https://auth.example.com/v1/projects/prj_7Fk2/admin/config/auth \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: live" \
  -H "Content-Type: application/json" \
  -d '{
    "methods": ["email", "oauth", "passkey"],
    "registration": { "mode": "invite_only", "password_strategy": "password_first" },
    "app_base_url": "https://app.example.com",
    "default_locale": "en",
    "supported_locales": ["en", "ru"]
  }'
```

| Field | Values |
| --- | --- |
| `methods[]` | `email`, `oauth`, `passkey`, `magic_link`, `phone` |
| `registration.mode` | `open`, `invite_only`, `request_access`, `closed` |
| `registration.password_strategy` | `password_first` (collect at signup), `after_verify` (collect once the email is verified) |
| `app_base_url` | your hosted auth UI. The "continue on another device" email links to `{app_base_url}/continue?flow={token}`; empty disables that email unless the flow supplies an allowed `redirect_to` |
| `default_locale` | must be one of `supported_locales` when that list is set |
| `supported_locales[]` | BCP-47 tags; drives email language and the hosted screens |

`username` is not a method — no such credential exists. See
[Registration & invites](/concepts/registration).

### `password_policy`

```bash
curl -sX PATCH .../admin/config/password-policy -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"min_length": 12, "zxcvbn_min_score": 3}'
```

| Field | Range | Notes |
| --- | --- | --- |
| `min_length` | 1–256 | |
| `zxcvbn_min_score` | 0–4 | strength estimate the password must reach |
| `breached_check` | `false` only | the breach corpus is not wired up; `true` is refused rather than accepted as a no-op |
| `history` | `0` only | reuse history is not implemented; any positive value is refused |

**Account lockout** is not configurable and is always on: **10** consecutive
wrong passwords lock that credential for **15 minutes**. It is deliberately
moderate — high enough that a fat-fingered password does not lock anyone out,
low enough to throttle credential stuffing that spreads across IPs and so slips
past [rate limits](#rate_limits). The short duration is what stops the lockout
itself becoming a denial of service against a chosen victim. Only password
sign-in is affected; passwordless and MFA paths are unaffected.

### `session_policy`

```bash
curl -sX PATCH .../admin/config/session-policy -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"access_ttl": 900, "refresh_ttl": 2592000, "idle_timeout": 86400, "reuse_detection": true}'
```

All values are **seconds**.

| Field | Max | Notes |
| --- | --- | --- |
| `access_ttl` | 86400 | access-token lifetime; also the `iam_session` cookie's Max-Age |
| `refresh_ttl` | 31536000 | refresh lifetime; the `iam_refresh` cookie's Max-Age, so this decides how often a person is sent back to a login screen |
| `idle_timeout` | 31536000 | a session unused for this long ends |
| `absolute_timeout` | 31536000 | a session ends this long after sign-in, however active |
| `reuse_detection` | bool | reuse is rejected regardless; this controls the response to it |

Ordering is enforced, so an unusable combination is refused up front:
`access_ttl < refresh_ttl`, `access_ttl ≤ idle_timeout ≤ absolute_timeout`, and
`idle_timeout ≤ refresh_ttl`.

These lifetimes apply to the OIDC provider's tokens too — see
[Revoking issued tokens](/concepts/oidc-federation).

### `mfa_policy`

```bash
curl -sX PATCH .../admin/config/mfa-policy -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"required_for_admins": true, "allowed_factors": ["totp", "webauthn"], "remember_device": true}'
```

| Field | Values |
| --- | --- |
| `allowed_factors[]` | `totp`, `sms`, `email_otp`, `webauthn`, `backup_codes` |
| `required_for_admins` | admins must hold a factor |
| `remember_device` | a verified device may skip the challenge for a while |

An omitted (or absent) `allowed_factors` means every implemented factor may be
enrolled. An **empty** list together with `required_for_admins: true` is refused:
that combination locks every admin out.

### `rate_limits`

```bash
curl -sX PATCH .../admin/config/rate-limits -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"rules": [
        {"endpoint": "/v1/auth/sign-in/password", "by": "ip", "limit": 10, "window_seconds": 60},
        {"endpoint": "/v1/auth/otp/start",        "by": "ip", "limit": 5,  "window_seconds": 60}
      ]}'
```

Each rule needs `endpoint`, `by`, `limit` (1–1 000 000) and `window_seconds`
(1–86 400). `(endpoint, by)` must be unique.

- `by` — only `ip`. The limiter keys by client IP and nothing else, so a rule
  naming any other subject is refused instead of quietly never matching. Behind
  a proxy this depends on `IAM_SERVICE_HTTP_TRUSTED_PROXIES` — see
  [Configuration](/self-hosting/configuration).
- `action` — unsupported; the limiter has no notion of one.
- `endpoint` — must be a path the limiter classifies. The full set:

  ```
  /v1/auth/guest
  /v1/auth/sign-up                       /v1/auth/sign-in/password
  /v1/auth/token/refresh                 /v1/auth/token/exchange
  /v1/auth/oauth/exchange                /v1/auth/access-requests
  /v1/auth/password/forgot               /v1/auth/password/reset
  /v1/auth/password/verify
  /v1/auth/email/verification/start      /v1/auth/email/verification/verify
  /v1/auth/phone/verification/start      /v1/auth/phone/verification/verify
  /v1/auth/otp/start                     /v1/auth/otp/verify
  /v1/auth/magic-link/start              /v1/auth/magic-link/verify
  /v1/auth/mfa/challenge                 /v1/auth/mfa/verify
  /v1/auth/webauthn/login/options        /v1/auth/webauthn/login/verify
  /v1/auth/webauthn/register/options     /v1/auth/webauthn/register/verify
  /v1/challenges/captcha/verify
  ```

  Anything else is refused, because a rule on an unclassified path could never
  fire.

## Feature toggles

`GET/PUT admin/features` is a flat `{"key": bool}` map that switches subsystems
off without reconfiguring them. Unknown keys are refused.

```bash
curl -sX PUT .../admin/features -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"guest": false, "impersonation": false, "access_requests": true}'
```

Keys: `password`, `otp`, `phone_login`, `magic_link`, `webauthn`, `oauth`,
`mfa`, `guest`, `email_verification`, `phone_verification`, `email_change`,
`phone_change`, `resumable_flows`, `impersonation`, `consent`,
`access_requests`, `step_up`.

## Public metadata

`GET/PUT admin/config/public-metadata` is a flat `map[string]string` published
at the unauthenticated `GET /v1/config/public` bootstrap call — the same call
your client already makes before anyone signs in, to render the login screen.
Unlike `features`, there is no closed key registry: this document is for your
own product flags (a feature flag, a maintenance banner, a minimum client
version), not a toggle for a subsystem IAM implements.

```bash
curl -sX PUT .../admin/config/public-metadata -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"beta_banner": "true", "min_client_version": "1.4.0"}'
```

`PUT` is a **replacement**: a key left out of the request is removed, same as
`consents` below. Bounded to 100 entries and 4096 characters per value, so it
cannot be grown into an unbounded document by accident.

**It is public.** Anyone who knows this project's client ID reads every key and
value — there is no admin-only variant. Put nothing here that is not meant to
be world-readable; a value that must stay authenticated belongs in a real
config document behind `adminToken`, or in a system built for secrets.

There is no separate flag-management service in IAM, and this is deliberately
not meant to become one — it exists because the bootstrap call is already the
cheapest place to carry a handful of public strings, not because IAM intends to
grow into general application configuration.

## Consent documents

`GET/PUT admin/consents` holds the documents a user must accept. A `required`
document is presented during signup and its acceptance is recorded against the
user.

```bash
curl -sX PUT .../admin/consents -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"documents": [
        {"key": "tos", "version": "2026-01", "title": "Terms of Service",
         "url": "https://example.com/tos", "locale": "en", "required": true},
        {"key": "marketing", "version": "1", "title": "Product emails", "required": false}
      ]}'
```

`key` and `version` are required. Bump `version` to re-ask everyone; the old
acceptance stays on record. Use `url` to link a hosted document, or `body` to
inline the text.

## Retention

`GET/PUT admin/retention-policy` decides how long data lives.

```bash
curl -sX PUT .../admin/retention-policy -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "inactive_user": {"after_days": 730, "action": "notify_then_suspend", "notify_before_days": 14},
    "audit_log_retention_days": 365,
    "event_retention_days": 90,
    "soft_delete_grace_days": 30,
    "purge_deleted_after_days": 90
  }'
```

`inactive_user.action` is `notify`, `notify_then_suspend` or
`notify_then_delete`. A deleted user is soft-deleted for
`soft_delete_grace_days` (restorable), then purged after
`purge_deleted_after_days`.

## Localized copy

`GET/PUT admin/i18n/{locale}` overrides UI and email strings for one locale.
The locale must be in `auth.supported_locales`. Email templates are separate —
see [Email & SMS](/guides/notifications).

## Invites

```bash
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/invites \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com"}'
# -> the invite token is shown ONCE in the response; deliver it to the user
```

Or from the SDK with `createIamInvitesAdmin` — see the
[SDK reference](/sdk/typescript#admin-invites-separate-client).

## Webhooks & blocking hooks

```bash
# async notification (Standard Webhooks HMAC)
curl -sX POST .../admin/webhooks -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"url":"https://api.example.com/iam","events":["user.created","session.created"]}'
```

Blocking hooks (fail-closed, can veto an operation) are configured under
`.../admin/hooks`. Concepts in [Webhooks & hooks](/concepts/webhooks-hooks).

## Environments

Every admin call accepts `X-Environment` (`live`, `test`, `staging`) and
operates on that environment's isolated data. Config is per-environment too — a
`test` environment can have `registration_mode: open` while `live` is
`invite_only`. See [Projects & environments](/concepts/projects-environments).

## Config as code

The operator plane can export/plan/apply a whole project config declaratively:

```bash
curl -sX POST https://auth.example.com/mgmt/v1/projects/prj_7Fk2/config:export \
  -H "Authorization: Bearer $MASTER_KEY" > project.json
# edit, then:
curl -sX POST https://auth.example.com/mgmt/v1/projects/prj_7Fk2/config:plan  -d @project.json ...
curl -sX POST https://auth.example.com/mgmt/v1/projects/prj_7Fk2/config:apply -d @project.json ...
```
