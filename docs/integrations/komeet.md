# Komeet integration contract

Komeet delegates account identity and authenticated device sessions to IAM. It
keeps its product profile, message data, WebSocket state and push destinations.
The stable join keys are the access-token `sub` (user) and `sid` (session/device).

## Tokens and sessions

- Human access tokens are signed JWTs with a default lifetime of 10 minutes.
  A project's `session_policy.access_ttl` may override that value.
- The core auth access token contains `iss`, `sub`, `sid`, `jti`, `pid`, `env`,
  `aud`, `aal`, `amr` and `typ=access`. `client_id` is present when the login is
  associated with a client. OAuth/OIDC tokens additionally carry their granted
  `scope`.
- The canonical issuer is `/p/{project_id}/e/{environment}`. Verify the JWT
  locally with `GET /p/{project_id}/e/{environment}/.well-known/jwks.json`, pin
  the expected issuer, project, environment, audience/client and access-token
  type, and cache keys by `kid`.
- `POST /v1/auth/token/refresh` rotates a refresh token and returns a new access
  token without creating a new device session. The default refresh lifetime is
  30 days. Reuse detection and idle/absolute limits are controlled by the
  session policy.
- The client may send `X-Device-Name` and `X-Device-Fingerprint` on sign-in.
  IAM persists them atomically with the new session. Device names are limited
  to 1024 Unicode code points and fingerprints to 256.
- `GET /v1/sessions` and `GET /v1/sessions/current` are the source of truth for
  the device list. Session rename, trust and revoke operations remain in IAM.

Email is intentionally not copied into every access token. Komeet reads it once
from authenticated `GET /v1/users/me`, then updates its delivery address on the
`email.changed` webhook. IAM remains the source of truth for the address and its
verification state.

## Public webhooks

The public event catalogue is deliberately allowlisted so internal events that
may contain OTPs, login links or credentials can never leak through a wildcard
subscription:

| Event | Data |
| --- | --- |
| `session.revoked` | `session_id`, `user_id`, `project_id` |
| `user.banned` | `user_id`, `status=banned` |
| `user.deleted` | `user_id` |
| `email.changed` | `user_id`, `email`, `email_verified` |

Every body is a versioned envelope with `id`, `type`, `version`, `occurred_at`,
`project_id`, `environment` and `data`. Delivery is at least once. Consumers
must deduplicate by event `id` and accept new fields in an existing version.

Create and manage subscriptions under
`/v1/projects/{project_id}/admin/webhooks`. Creation supports `Idempotency-Key`
and returns the signing secret only as part of the create response. The secret
is encrypted at rest. Rotation returns a new secret and keeps a 24-hour overlap
during which deliveries include signatures made by both secrets.

For each request IAM sends:

```text
webhook-id: <event id>
webhook-timestamp: <unix seconds>
webhook-signature: v1,<base64 hmac sha256>[ v1,<old-secret signature>]
```

IAM uses the Standard Webhooks format. The signed bytes are the event id, a
dot, the ASCII timestamp, a dot, and the exact raw HTTP body:

```text
HMAC-SHA256(signing_secret, event_id + "." + timestamp + "." + raw_body)
```

Use `sdk.NewWebhookVerifier` to verify at least one `v1` value with a
constant-time comparison before decoding the body. It rejects timestamps
outside a five-minute window by default. Consumers own replay persistence and
should record the verified event `id` before applying side effects.

Non-2xx responses and network errors are retried with exponential backoff,
capped at five minutes, for up to 10 outbox attempts. Successful deliveries are
not resent when another subscription failed. Delivery status and the bounded
response body are available from the admin delivery endpoint; failed deliveries
can be retried manually. Archived public events can also be replayed.

On `session.revoked`, Komeet should close the matching `sid` WebSocket, remove
the matching push destination, and require a fresh login. On `user.banned` or
`user.deleted`, it should disconnect every session belonging to `user_id`.

## Database upgrade

Migration `20260712135920777_webhook_delivery.sql` is additive. It adds columns
to the previously unused webhook/event tables and creates
`iam_webhook_deliveries`; it does not alter, rewrite or rebuild `iam_users`,
credentials, identities, factors, sessions or refresh tokens. The integration
suite applies it over the previous bootstrap schema containing an existing user
and verifies that the user's email and JSON data remain unchanged.
