---
id: test-mode
title: Test mode
sidebar_label: Test mode
---

# Test mode

`/v1/test/*` exists so an automated suite can drive real auth flows without a
mailbox, an SMS gateway, or a hand-maintained fixture database.

Every operation is refused unless `X-Environment` names a **non-`live`**
environment. Test mode must never touch live data, so this is a hard gate rather
than a convention.

All four calls need an authenticated principal (a project-admin token is the
usual choice) and the `X-Environment` header.

## Read the codes IAM sent

The single most useful one. Returns the out-of-band deliveries recorded for the
environment, newest first:

```bash
curl -s "https://auth.example.com/v1/test/messages?to=ada@example.com" \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: test"
# -> { "data": [ { "type": "auth.otp.started",
#                   "payload": { "to": "ada@example.com", "channel": "email",
#                                "code": "123456", "challenge_id": "…" } } ] }
```

Filter with `channel` (`email` / `sms`) and `to`. A test can therefore start a
passwordless sign-in, read the code back, and finish the flow — no provider
needed. It reads recorded events, so it works whether or not a real provider is
configured.

It surfaces the `auth.*` events, which is what carries a code or a link:
`auth.otp.started`, `auth.magiclink.started`, `auth.flow.continue` and their
`.verified` counterparts. Email-verification and password-reset codes are not
among them — drive those through the admin API (`POST admin/users/{id}/verify-email`)
or through the resumable flow instead.

## Reset the environment

```bash
curl -sX POST https://auth.example.com/v1/test/reset \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: test"
# -> { "ok": true, "deleted": 128 }
```

Wipes the environment's runtime data — users, sessions, refresh tokens,
credentials, factors, recovery codes, identities, challenges, flows,
authorization and device codes, activity. Project **configuration** is left
alone, so a suite can reset between runs without reconfiguring.

## Seed a user

```bash
curl -sX POST https://auth.example.com/v1/test/seed \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: test" \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","name":"Ada"}'
```

Creates an active, email-verified user. Idempotent: seeding the same address
twice is a no-op. Only `email` (required) and `name` are read.

## Clock

```bash
curl -sX POST https://auth.example.com/v1/test/clock \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: test" \
  -H "Content-Type: application/json" -d '{"advance_seconds": 3600}'
```

:::note Records the offset, does not yet move time
The offset is persisted for the environment, but it is not wired into the
service's clock — expiry still follows real time. To test expiry today, shorten
the relevant TTL in `session_policy` instead.
:::
