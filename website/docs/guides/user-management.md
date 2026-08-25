---
id: user-management
title: Managing users
sidebar_label: Managing users
---

# Managing users

Everything here is the project-admin surface,
`/v1/projects/{project_id}/admin/users`, authenticated with an `adminToken` and
scoped by `X-Environment`. The admin console drives the same endpoints.

```bash
export A="-H 'Authorization: Bearer $ADMIN_TOKEN' -H 'X-Environment: live'"
export U="https://auth.example.com/v1/projects/prj_7Fk2/admin/users"
```

## Find and read

```bash
# keyset-paginated; q is a free-text search, and email/phone/kind filter exactly
curl -s "$U?q=ada&limit=50" -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: live"
curl -s "$U/usr_9"          -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: live"
```

Paging is keyset-based: follow `next_cursor` from the response rather than
computing an offset.

## Create and edit

```bash
curl -sX POST "$U" -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: live" \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","password":"…","email_verified":true,"name":"Ada"}'

curl -sX PATCH "$U/usr_9" -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"name":"Ada L."}'
```

Creating a user with `email_verified: true` skips the verification email — right
for a migration, wrong for a self-service signup.

## Access control

| Action | Call | Effect |
| --- | --- | --- |
| Ban | `POST /{id}/ban` `{reason, until}` | sign-in refused; omit `until` for indefinite |
| Unban | `POST /{id}/unban` | lifts it |
| Set password | `POST /{id}/password` `{password, revoke_sessions}` | sets a password out of band; `revoke_sessions` signs the user out everywhere |
| Reset MFA | `POST /{id}/mfa/reset` `{factor_ids}` | removes the named factors, or every factor when omitted — the account-recovery path when someone loses their phone |
| Mark verified | `POST /{id}/verify-email`, `POST /{id}/verify-phone` | flips the flag without sending a code |

## Sessions

```bash
curl -s  "$U/usr_9/sessions"                # what devices are signed in
curl -sX DELETE "$U/usr_9/sessions/ses_2"   # end one
curl -sX POST   "$U/usr_9/sessions/revoke" \
  -H "Content-Type: application/json" -d '{"except_session_id":"ses_2","reason":"support"}'
```

Revoking a session ends its refresh tokens, and notifies any OIDC client holding
a grant on it through [back-channel logout](/concepts/oidc-federation).

## Identities and grants

`GET/DELETE /{id}/identities[/{identity_id}]` lists and unlinks the external
accounts (Google, GitHub, an upstream IdP) attached to the user. Unlinking the
last identity of a user with no password locks them out — check first.

`GET/DELETE /{id}/grants[/{grant_id}]` is the OAuth consent the user gave to your
apps. Deleting a grant means the client's next authorization request shows the
consent screen again.

## Roles

```bash
curl -sX PUT "$U/usr_9/roles" -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Environment: live" -H "Content-Type: application/json" \
  -d '{"roles":["ops","platform:admin"]}'
```

A replacement, not a merge, and scoped to the environment in the header. Roles
become the `groups` claim for clients granted that scope — see
[Roles in the token](/concepts/oidc-federation).

## Impersonation

```bash
curl -sX POST "$U/usr_9/impersonate" -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason":"ticket #4412","duration_seconds":900}'
# -> { "impersonation_url": "https://app.example.com/…?token=…", "expires_at": "…" }
```

`reason` and `duration_seconds` are both required — an impersonation without a
recorded reason is an audit hole. The URL carries a one-time token the app
redeems with `POST /v1/auth/impersonate/redeem`, which returns a normal session.
Both the start and the redemption are audited, and the feature can be switched
off entirely with the `impersonation` [feature flag](/guides/admin-config).

## Deleting and anonymizing

```bash
curl -sX DELETE "$U/usr_9"           # soft delete — restorable during the grace period
curl -sX DELETE "$U/usr_9?hard=true" # immediate, unrecoverable

curl -sX POST "$U/usr_9/anonymize" -H "Content-Type: application/json" \
  -d '{"reason":"GDPR erasure request"}'
```

Anonymizing strips the personal data but keeps the row, so audit history and
foreign keys stay intact. That is usually what an erasure request actually
needs; a hard delete is for a record that should never have existed. The grace
period and purge schedule come from
[`retention-policy`](/guides/admin-config).

`POST /{id}/export` starts a data-export job for a subject-access request — see
[Import & export](/guides/import-export).
