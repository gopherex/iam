---
id: flows
title: Resumable auth flows
sidebar_label: Auth flows
---

# Resumable auth flows

`/v1/auth/flows` is a **server-side state machine** for signup, signin, recovery
and email-change. The client holds only an opaque `flow_token` (prefixed `ftk_`)
and renders the returned `FlowState` verbatim; the server owns all state. This is
the recommended integration path — it survives reloads and works cross-device.

All flow endpoints are **public** (`security: []`) — the `flow_token` is the
credential — but require `X-Client-Id`.

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/v1/auth/flows` | Create a flow |
| `GET` | `/v1/auth/flows/current` | Resume via the `iam_flow` cookie |
| `GET` | `/v1/auth/flows/{flow_token}` | Resume explicitly (deep link) |
| `DELETE` | `/v1/auth/flows/{flow_token}` | Abandon → `204` |
| `POST` | `/v1/auth/flows/{flow_token}/submit` | Advance one step |
| `POST` | `/v1/auth/flows/{flow_token}/resend` | Re-issue the active challenge |

Each pending response also sets an HttpOnly `iam_flow` cookie (Path
`/v1/auth/flows`, 30-min TTL), cleared on completion.

## FlowState

Every response returns the full state:

```json
{
  "flow_token": "ftk_...",
  "kind": "signup",            // signup | signin | recovery | email_change
  "status": "pending",         // pending | completed | expired | aborted
  "step": "verify_email",      // see below
  "next_actions": ["submit_code", "resend"],
  "contact": { "email_masked": "a***@example.com" },
  "challenge": { "channel": "email", "expires_at": "...", "resend_at": "...", "attempts_left": 5 },
  "consents_required": [],
  "factors": [],
  "expires_at": "...",
  "error": null,               // { code, message } on bad input — status stays pending
  "session": null              // SessionTokens, present exactly once at status=completed
}
```

`step` ∈ `collect_credentials | verify_email | verify_phone | set_password |
mfa_required | step_up | accept_consents | request_access | awaiting_approval |
completed | blocked`.

Use `next_actions[]` to decide which action to `submit` next; `resend` is present
whenever a challenge is active.

## Create request

```json
{
  "kind": "signup",            // required
  "method": "password",        // password | phone_otp | magic_link | passkey | oauth (signin default: password)
  "email": "ada@example.com",
  "phone": "+15551234567",
  "provider": "google",        // for method: oauth
  "password": "hunter2pw",
  "name": "Ada",
  "captcha_token": "...",
  "redirect_to": "/welcome",
  "locale": "en",
  "invite_token": "inv_...",   // invite_only projects
  "consents": []
}
```

## Submit request

```json
{ "action": "submit_code", "payload": { "code": "123456" } }
```

## State machines

- **signup** — `collect_credentials → verify_email → [accept_consents] → completed`
- **signin** — `collect_credentials → [mfa_required] → completed`
- **recovery** — `collect_credentials → verify_email → set_password → completed`
- **email_change** — not a flow kind; use the dedicated `/v1/auth/email/change/*`
  endpoints (a flow create with `kind: email_change` returns `400`).

## Worked example: signup

```bash
# 1. create
curl -sX POST https://auth.example.com/v1/auth/flows \
  -H "X-Client-Id: app_web" -H "Content-Type: application/json" \
  -d '{"kind":"signup","email":"ada@example.com","password":"hunter2pw","name":"Ada"}'
```
```json
{ "flow_token": "ftk_9s2...", "kind": "signup", "status": "pending",
  "step": "verify_email", "next_actions": ["submit_code","resend"],
  "contact": { "email_masked": "a***@example.com" },
  "challenge": { "channel": "email", "attempts_left": 5 } }
```
```bash
# 2. submit the emailed code → completed + session
curl -sX POST https://auth.example.com/v1/auth/flows/ftk_9s2.../submit \
  -H "X-Client-Id: app_web" -H "Content-Type: application/json" \
  -d '{"action":"submit_code","payload":{"code":"123456"}}'
```
```json
{ "status": "completed", "step": "completed",
  "session": { "access_token": "eyJ...", "refresh_token": "rt_...",
               "expires_in": 1800, "token_type": "Bearer" } }
```

```bash
# resume via cookie / abandon / resend
curl https://auth.example.com/v1/auth/flows/current -H "X-Client-Id: app_web" --cookie "iam_flow=ftk_9s2..."
curl -X DELETE https://auth.example.com/v1/auth/flows/ftk_9s2... -H "X-Client-Id: app_web"     # 204
curl -X POST https://auth.example.com/v1/auth/flows/ftk_9s2.../resend -H "X-Client-Id: app_web"
```

## Error handling inside a flow

A wrong code does **not** reset the flow: the response stays `status: "pending"`
with `error: { code, message }` set and `challenge.attempts_left` decremented.
Branch on `error.code`, not the HTTP status. Resending too soon returns `429
flow_resend_too_soon`.

The [TypeScript SDK](/guides/auth-flows) wraps all of this in a `FlowController`.
