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

## Account (secured — `/v1/users/me`)

Profile (`GET/PATCH/DELETE /v1/users/me`), activity, export, consents,
identities, identity-merge, email/phone change, password check/verify/update.
The [TypeScript SDK](/sdk/typescript) `iam.account` namespace wraps these.
