---
id: security-controls
title: Abuse controls & audit
sidebar_label: Abuse & audit
---

# Abuse controls & audit

Four separate mechanisms, deliberately not one "security" setting: rate limits
throttle everybody, blocks stop a named subject, the audit log records what
administrators did, and the event log records what the system did.

## Rate limits

Per-endpoint limits are a [config document](/guides/admin-config#rate_limits).
They key on the **client IP** and nothing else, which makes the proxy setting
load-bearing: behind an ingress, set `IAM_SERVICE_HTTP_TRUSTED_PROXIES` to its
CIDR, or every request looks like it came from the proxy and one user can
exhaust the limit for all of them. See
[Configuration](/self-hosting/configuration).

A throttled request gets `429` with the standard rate-limit headers.

## Blocks

A block refuses an authentication flow outright, for a named subject rather than
a rate:

```bash
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/rate-limit/blocks \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"type":"ip","value":"203.0.113.7","reason":"credential stuffing","expires_at":"2026-09-01T00:00:00Z"}'

curl -sX DELETE .../admin/rate-limit/blocks/blk_1 -H "Authorization: Bearer $ADMIN_TOKEN"
```

`type` is `ip`, `email`, `phone` or `asn`. Omit `expires_at` for an indefinite
block.

Blocks are checked when an auth flow starts: the caller's IP and the target
email or phone are matched against unexpired blocks, and a match is refused with
`403 blocked`. `asn` blocks are stored but not resolved at request time — IAM
does not look up the ASN of a connecting IP — so use `ip` for anything that must
actually take effect.

## Risk rules

A rule pairs a **signal** with what to do when it fires, and is evaluated the
moment a password verifies — before a session exists, which is the only point
where "step up" is still a decision rather than the retraction of a session
already handed out.

```bash
curl -sX POST .../admin/risk/rules -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"unfamiliar device","signal":"new_device","action":"require_step_up","enabled":true}'
```

| Signal | Fires when |
| --- | --- |
| `new_device` | no earlier session of this user carried this device fingerprint |
| `new_ip` | no earlier session of this user came from this address |
| `recent_failures` | the password was wrong at least once since the last successful sign-in |

| Action | Effect |
| --- | --- |
| `require_step_up` | the sign-in returns `mfa_required` instead of a session |
| `block` | the sign-in is refused |
| `notify` | recorded only |
| `allow` | an explicit exception — it beats every other rule that fired |

Signals are a **closed set**, not an expression language. A rule naming anything
else is refused when it is written, so an enabled rule is never a control that
silently never fires. `new_device` needs the client to send a device
fingerprint; without one it never fires rather than firing on everybody.

`require_step_up` only takes effect when the account actually has a second
factor — demanding one it does not have would lock the user out of their own
account, which is not what the rule asked for.

Rules are evaluated together and the **strongest** action wins, except that
`allow` overrides everything: that is how you carve an exception without
deleting the rule it is an exception to.

`GET admin/risk/events` returns what fired, newest first — the rule, the signal,
the action and the account. The record is written outside the sign-in's
transaction, so a `block` still leaves a trace even though it aborted everything
else.

`condition` is the field's released spelling. It is still accepted and still
evaluated, but it must now name a signal; prefer `signal`.

## Audit log

Every administrative action is recorded with its actor, its target and its
payload:

```bash
curl -s ".../admin/audit-logs?actor_id=adm_1&type=user.banned" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
curl -s  .../admin/audit-logs/aud_42 -H "Authorization: Bearer $ADMIN_TOKEN"
```

Filter by `actor_id`, `target_id` and `type`; the list is keyset-paginated.
Export a window with [`POST admin/audit/export`](/guides/import-export). How long
entries are kept is `audit_log_retention_days` in
[`retention-policy`](/guides/admin-config).

Impersonation is the case this exists for: both the start (with the required
`reason`) and the redemption are recorded.

## Event log

`GET admin/events` is the system's own event stream — sign-ins, sessions,
credential changes, OIDC token issuance, federation activity — filterable by
`type` and `user_id`. `POST admin/events/{event_id}/replay` re-dispatches one
through the outbox, which is how you recover a webhook delivery that was lost
while your endpoint was down.

This is a superset of what webhooks deliver. Subscriptions are limited to four
event types on purpose — see [Webhooks & hooks](/concepts/webhooks-hooks) — so
anything carrying a code or a one-time proof stays inside IAM.

## What is not here

Authorization. IAM decides *who is calling* and how strongly that was proved
(`aal`, `amr`, `groups`); deciding *what they may do* belongs to your
authorization layer, which builds on the identity IAM issues.
