---
id: registration
title: Registration
sidebar_label: Registration
---

# Registration

How new users are allowed to join a project is controlled by the `auth` config
doc's `registration` block (`PUT /v1/projects/{id}/admin/config/auth`).

## Registration modes

`registration.mode` ∈ `open | invite_only | request_access | closed`:

| Mode | Behavior |
| --- | --- |
| `open` | Anyone may sign up. |
| `invite_only` | Sign-up requires a valid `inv_…` invite token; otherwise the flow reaches step `blocked` with error `invite_required`. |
| `request_access` | Sign-up parks at step `request_access` instead of creating an account; the prospect submits an access request for an admin to decide on. |
| `closed` | Sign-up is disabled; the flow reaches `blocked` with reason `registration_closed`. |

The mode is enforced uniformly across the [resumable flow](/rest-api/flows)
and the direct `/v1/auth/sign-up` endpoint.

## Invitations

Admins issue invites; the raw `inv_…` token is returned **once** on create:

```bash
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/invites \
  -H "Authorization: Bearer <admin_token>" -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","expires_at":"2026-09-01T00:00:00Z"}'
# -> { "invite": {...}, "token": "inv_… (shown once)" }
```

Redeem it during signup by passing `invite_token` on the flow create (or
`invitation_token` on `/v1/auth/sign-up`). Redemption is atomic — a single-use
invite can never yield two accounts.

> Invites carry only email-binding and expiry. Roles/permissions belong to the
> separate AuthZ product, not IAM.

## Access requests

Under `request_access`, sign-up is a request rather than an account. A prospect
submits one, an admin decides, and approval turns into an invitation:

```bash
# the prospect (public — needs only X-Client-Id)
curl -sX POST https://auth.example.com/v1/auth/access-requests \
  -H "X-Client-Id: prj_7Fk2" -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","reason":"joining the ops team","fields":{"team":"platform"}}'

# the admin
curl -s  .../admin/access-requests                       -H "Authorization: Bearer $ADMIN_TOKEN"
curl -sX POST .../admin/access-requests/req_1/approve    -H "Authorization: Bearer $ADMIN_TOKEN"
curl -sX POST .../admin/access-requests/req_1/deny       -H "Authorization: Bearer $ADMIN_TOKEN"
```

`fields` is a free-form object, so a project can ask for whatever it needs to
make the decision (company, team, referrer) without a schema change.

A flow started in this mode parks at step `request_access` rather than failing,
so the same client code handles it as a state, not an error. Submitting the
request is the client's job — parking the flow does not create the record.

:::note Approval is a decision, not an onboarding
`approve` marks the request `approved` and emits `access_request.approved`. It
does **not** issue an invite, create the account, or resume a parked flow. To
actually let the person in, follow an approval with an
[invite](#invitations) — or subscribe to the event and do it from your own
backend.
:::

The whole feature can be switched off with the `access_requests`
[feature flag](/guides/admin-config).

## Password strategy

`registration.password_strategy` ∈ `password_first | after_verify`:

- **`password_first`** (default) — the user sets a password up front.
- **`after_verify`** — the account is created without a password; the user sets
  one only *after* verifying their email (flow step `set_password`).

## Consents

If the project configures required consent documents (`config/consents`), signup
gates on them: the flow surfaces `consents_required[]` and parks at
`accept_consents` until the user accepts. Users manage consents later via
`/v1/users/me/consents`.

## Captcha

When a captcha provider is configured for the project, the unauthenticated entry
points (signup, password-recovery, flow create) require a valid `captcha_token`;
projects without a configured provider are unaffected.
