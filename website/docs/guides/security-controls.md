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

```bash
curl -sX POST .../admin/risk/rules -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"impossible travel","condition":"…","action":"require_step_up","enabled":true}'
```

:::caution Stored, not evaluated
Risk rules are a declarative store today — there is no evaluation engine, so a
rule never fires, and `GET admin/risk/events` returns an empty page. Use
**blocks** and **rate limits** for controls that actually take effect, and treat
rules as forward configuration only.
:::

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
