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
