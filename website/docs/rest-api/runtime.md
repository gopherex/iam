---
id: runtime
title: Runtime endpoints
sidebar_label: Runtime endpoints
---

# Runtime endpoints

The end-user (`/v1/*`) surface. All accept `X-Client-Id` + `X-Environment`;
public endpoints resolve the caller from `X-Client-Id`, secured ones need a user
bearer/cookie. Prefer the [resumable flow](/rest-api/flows) for signup/signin —
these are the direct primitives it is built on.

## Sign-up / sign-in

| Method | Path | Body → Result |
| --- | --- | --- |
| `POST` | `/v1/auth/sign-up` | `SignUpRequest` → `AuthResult` |
| `POST` | `/v1/auth/sign-in/password` | `{email\|phone, password}` → `AuthResultOrNextStep` |
| `POST` | `/v1/auth/guest` | → anonymous guest session |

```bash
curl -sX POST https://auth.example.com/v1/auth/sign-in/password \
  -H "X-Client-Id: app_web" -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","password":"hunter2pw"}'
```
```json
// authenticated
{ "result_type": "authenticated",
  "user": { "id": "usr_9aQ2", "kind": "human", "status": "active",
            "primary_email": "ada@example.com", "email_verified": true },
  "session": { "access_token": "eyJ...", "refresh_token": "rt_...",
               "expires_in": 1800, "token_type": "Bearer" } }
// or MFA next-step
{ "result_type": "next_step", "next_step": "mfa_required",
  "flow_token": "flw_...", "factors": [{ "id": "fac_totp", "type": "totp", "status": "active" }] }
```

`next_step` ∈ `verify_email | verify_phone | mfa_required | step_up |
reconsent_required | signup_required | null`.

## Passwordless — OTP & magic link

| Method | Path |
| --- | --- |
| `POST` | `/v1/auth/otp/start` — `{identifier, channel: email\|sms\|whatsapp, purpose: signin\|signup\|verify}` → `Challenge` |
| `POST` | `/v1/auth/otp/verify` — `{challenge_id, code}` → `AuthResult` |
| `POST` | `/v1/auth/magic-link/start` — `{email, redirect_to, purpose}` |
| `POST` | `/v1/auth/magic-link/verify` — `{token}` → `AuthResult` |
| `GET` | `/v1/auth/magic-link/callback?token&redirect_to` → `302` + session cookies |

```bash
curl -sX POST https://auth.example.com/v1/auth/otp/start \
  -H "X-Client-Id: app_web" -H "Content-Type: application/json" \
  -d '{"identifier":"ada@example.com","channel":"email","purpose":"signin"}'
# -> { "challenge_id": "chl_e1", "expires_at": "...", "type": "otp" }
```

## MFA

| Method | Path |
| --- | --- |
| `GET` | `/v1/auth/mfa/factors` |
| `POST` | `/v1/auth/mfa/challenge` — `{flow_token?, factor_id?, type?}` → `Challenge` |
| `POST` | `/v1/auth/mfa/verify` — `{flow_token?, challenge_id?, factor_id?, code?, credential?, recovery_code?}` → `AuthResult` |
| `POST` | `/v1/auth/mfa/totp/{enroll,verify}`, `/mfa/sms/enroll`, `/mfa/email/enroll`, `/mfa/webauthn/enroll/{options,verify}`, `/mfa/recovery-codes/{generate,verify}` |
| `DELETE` | `/v1/auth/mfa/factors/{factor_id}` |

## WebAuthn / passkeys

`POST /v1/auth/webauthn/login/{options,verify}`,
`/webauthn/register/{options,verify}`,
`GET|DELETE /v1/auth/webauthn/credentials[/{credential_id}]`.

## OAuth social login

| Method | Path |
| --- | --- |
| `GET` | `/v1/auth/oauth/providers` — list enabled providers |
| `GET` | `/v1/auth/oauth/{provider}/start?client_id&redirect_to&state&code_challenge` → `302` to provider |
| `GET` | `/v1/auth/oauth/{provider}/callback?code&state` → `302` to app + session cookies |
| `POST` | `/v1/auth/oauth/exchange` — code → session |
| — | `/v1/auth/oauth/{provider}/link/{start,callback}`, `/unlink` |

## Tokens

| Method | Path |
| --- | --- |
| `POST` | `/v1/auth/token/refresh` — `{refresh_token}` or `iam_refresh` cookie → `AuthResult` (rotates refresh, no new session) |
| `POST` | `/v1/auth/token/exchange` — `{grant_type:"auth_code", code, code_verifier?}` → `AuthResult` |
| `POST` | `/v1/tokens/introspect` — `{token}` → `{active, ...}` |
| `POST` | `/v1/tokens/revoke` — `{token?, session_id?, reason?}` |
| `POST` | `/v1/tokens/verify` — `{token, audience?}` → `{valid, claims, error}` |
| `GET` | `/v1/tokens/current` — claims for the presented bearer |

```bash
curl -sX POST https://auth.example.com/v1/auth/token/refresh \
  -H "X-Client-Id: app_web" -H "Content-Type: application/json" \
  -d '{"refresh_token":"rt_..."}'
```

## Session & sign-out (secured)

| Method | Path |
| --- | --- |
| `GET` | `/v1/auth/session` → `{user, session}` |
| `POST` | `/v1/auth/sign-out` — `{everywhere?}` |
| `POST` | `/v1/auth/sign-out-all` — `{except_current?}` → `{revoked_count}` |
| `POST` | `/v1/auth/session/step-up` — `{purpose, required_aal?, max_age_seconds?}` |
| `POST` | `/v1/auth/session/switch-group` |
| `GET/DELETE` | `/v1/sessions`, `/v1/sessions/current`, `/v1/sessions/{id}`, `POST /v1/sessions/{id}/trust` |

## Verification

| Method | Path |
| --- | --- |
| `POST` | `/v1/auth/email/verification/start` — `{email?}` → `Challenge` |
| `POST` | `/v1/auth/email/verification/verify` — `{challenge_id, code}` or `{token}` |
| `GET` | `/v1/auth/email/verification/callback?token&redirect_to` → `302` |
| `POST` | `/v1/auth/phone/verification/{start,verify}` |

## Password

| Method | Path |
| --- | --- |
| `POST` | `/v1/auth/password/forgot` — `{email}`; always `200`, so it cannot be used to test whether an address exists |
| `POST` | `/v1/auth/password/reset` — `{token, password}` |
| `POST` | `/v1/auth/password/verify` — re-prove the current password (step-up) |
| `POST` | `/v1/auth/password/check` — `{password}` → `{valid, score, violations}`, the policy check to run live in a signup form |
| `POST` | `/v1/auth/password/change` — `{current_password, new_password}` (secured) |

`check` is the only one that never changes anything: use it to show strength
feedback without submitting.

## Changing email or phone (secured)

| Method | Path |
| --- | --- |
| `POST` | `/v1/auth/email/change/start` — `{new_email, redirect_to?}` → `Challenge` sent to the **new** address |
| `POST` | `/v1/auth/email/change/verify` — `{challenge_id, code}` or `{token}` → the updated user; the new address is marked verified and `email.changed` fires |
| `GET` | `/v1/auth/email/change/cancel?token` — voids a pending change |
| `POST` | `/v1/auth/phone/change/{start,verify}` — `{new_phone, channel}`, then `{challenge_id, code}` |

:::caution The old address is not warned
Starting a change proves control of the **new** address only. IAM does not
require a step-up and does not mail the old address, so a hijacked session can
move the account quietly. `cancel` exists and works, but nothing delivers its
token to the previous owner — subscribe to `email.changed` and notify from your
own backend if that matters.
:::

## Identities (secured)

| Method | Path |
| --- | --- |
| `GET` | `/v1/auth/identities` — linked external accounts |
| `DELETE` | `/v1/auth/identities/{identity_id}` — unlink |
| `POST` | `/v1/auth/identities/merge/start` — `{target_identifier}` → `{challenge_id}` |
| `POST` | `/v1/auth/identities/merge/confirm` — `{challenge_id, code}` |

Merge folds a second account (someone who signed up with a password and later
with Google) into the current one, after proving control of the target.

## Access requests

When [registration mode](/concepts/registration) is `request_access`, sign-up
becomes a request an admin decides on:

```bash
curl -sX POST https://auth.example.com/v1/auth/access-requests \
  -H "X-Client-Id: prj_7Fk2" -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","reason":"joining the ops team"}'
```

Admins list and decide with `GET admin/access-requests` and
`POST admin/access-requests/{id}/{approve|deny}`. Approving records the decision
and nothing more — follow it with an invite to actually onboard the person. See
[Registration](/concepts/registration).

## Account (secured — `/v1/users/me`)

| Method | Path |
| --- | --- |
| `GET/PATCH/DELETE` | `/v1/users/me` — profile |
| `GET` | `/v1/users/me/activity` — the user's own security events, paginated |
| `POST` | `/v1/users/me/export` → `{job_id}`; `GET /v1/users/me/export/{job_id}` → `{status, download_url}` |
| `GET` | `/v1/account/capabilities` — what this user may self-manage, for rendering an account screen without guessing |
| — | consents, identities, identity-merge, email/phone change, password |

The [TypeScript SDK](/sdk/typescript) `iam.account` namespace wraps these.

`POST /v1/users/me/export` returns a `job_id`; poll
`GET /v1/users/me/export/{job_id}` for `{status, download_url}`. The document
excludes credential material — see
[Import & export](/guides/import-export).

## Bootstrap & system (public)

| Method | Path |
| --- | --- |
| `GET` | `/v1/config/public` — enabled methods, locales, registration mode: everything a login UI needs before anyone signs in |
| `GET` | `/v1/csrf` — a CSRF token for cookie-mode calls (see [Authentication](/rest-api/authentication)) |
| `GET` | `/v1/health`, `/v1/health/live`, `/v1/health/ready` — public health, on the API port |

`/v1/health*` is the API-port health surface; the deployment probes are
`/healthz/liveness` and `/healthz/readiness` on the probe port. Point orchestrator
probes at `/healthz/*` and use `/v1/health` when you only have the public API
reachable — see [Observability](/self-hosting/observability).
