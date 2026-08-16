---
id: webhooks-hooks
title: Webhooks & hooks
sidebar_label: Webhooks & hooks
---

# Webhooks & hooks

Two different integration mechanisms — don't confuse them.

| | **Webhooks** | **Blocking hooks** |
| --- | --- | --- |
| Timing | **Async**, after the fact | **Synchronous**, inline in the auth flow |
| Purpose | Notify your backend of events | **Veto / gate** an auth decision |
| Failure | Retried with backoff | Denies (fail-closed) or is skipped (fail-open) |
| Admin path | `/admin/webhooks` | `/admin/hooks` |

## Webhooks

Async, at-least-once event deliveries in **Standard Webhooks** format.

Create a subscription (the signing secret is returned **once**):

```bash
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/webhooks \
  -H "Authorization: Bearer <admin_token>" -H "Content-Type: application/json" \
  -d '{"url":"https://app.example.com/hooks/iam","events":["session.revoked","user.banned"]}'
```

### Signing

Each delivery is HMAC-SHA256 signed. Verify it as:

```
signature = HMAC_SHA256(secret, event_id + "." + timestamp + "." + raw_body)
```

Headers follow Standard Webhooks: `Webhook-Id`, `Webhook-Timestamp`,
`Webhook-Signature` (`v1,<base64>`). Secret rotation keeps the previous secret
valid for a 24h overlap (dual-signature).

### Event catalogue

The **public** event set is allowlisted — e.g. `session.revoked`, `user.banned`,
`user.deleted`, `email.changed`. Internal events that carry OTPs/magic links are
**never** delivered, so secrets can't leak through webhooks.

### Delivery & retry

Failed deliveries are retried with exponential backoff (capped), drained by a
background worker; you can inspect and manually retry via
`/admin/webhook-deliveries` and replay events via `/admin/events/{id}/replay`.

### SSRF protection

Webhook (and hook) URLs are delivered through an SSRF-guarded client that refuses
to connect to private, link-local, cloud-metadata (`169.254.169.254`), ULA, CGNAT
or unspecified addresses — including across redirects. Loopback is allowed only as
a local-dev escape hatch.

## Blocking hooks

A **blocking hook** is a signed HTTP callback IAM calls **synchronously** at an
auth decision point, and honors its verdict:

- `before_user_create`
- `before_sign_in`
- `before_token_issue`

A non-2xx reply (or a transport error/timeout) **denies** the action, unless the
hook is configured `fail_open`. **Fail-closed is the default.**

```bash
# register a hook that vetoes sign-ups
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/hooks \
  -H "Authorization: Bearer <admin_token>" -H "Content-Type: application/json" \
  -d '{"type":"before_user_create","url":"https://app.example.com/gate","enabled":true}'
# -> { "hook": {...}, "signing_secret": "whsec_… (shown once)" }

# test-fire it without applying any decision
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/hooks/<id>/test \
  -H "Authorization: Bearer <admin_token>" -H "Content-Type: application/json" \
  -d '{"payload":{"probe":true}}'
# -> { "status": 200, "response": {...}, "duration_ms": 42 }
```

Hooks are HMAC-signed (same scheme as webhooks) and SSRF-guarded, with a per-hook
timeout (default 3s, max 10s). A project with no hook of a given type is
unaffected.
