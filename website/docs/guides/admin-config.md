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

## Auth methods & registration

```bash
# enable password + otp + google, set registration to invite-only
curl -sX PUT https://auth.example.com/v1/projects/prj_7Fk2/admin/config/auth \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{
    "password_enabled": true,
    "otp_enabled": true,
    "oauth_providers": ["google"],
    "registration_mode": "invite_only"
  }'
```

Registration modes: `open`, `invite_only`, `request_access`, `closed` — see
[Registration & invites](/concepts/registration).

## Password policy

```bash
curl -sX PUT https://auth.example.com/v1/projects/prj_7Fk2/admin/config/password-policy \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"min_length":12,"require_breach_check":true,"reuse_window":5}'
```

## Session policy

```bash
curl -sX PUT https://auth.example.com/v1/projects/prj_7Fk2/admin/config/session-policy \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"access_ttl":900,"refresh_ttl":2592000,"idle_timeout":0,"reuse_detection":true}'
```

## MFA policy & rate limits

```bash
# require MFA for all users
curl -sX PUT .../admin/config/mfa-policy -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"required":true,"allowed_factors":["totp","webauthn"]}'

# tune rate limits
curl -sX PUT .../admin/config/rate-limits -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"sign_in":{"limit":10,"window_seconds":60}}'
```

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
