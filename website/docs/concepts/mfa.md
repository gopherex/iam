---
id: mfa
title: Multi-factor authentication
sidebar_label: MFA
---

# Multi-factor authentication

IAM supports a second factor on top of any first-factor method, raising a session
to **AAL2** (see [Sessions](/concepts/sessions)).

## Factor types

`Factor.type` ∈ `totp | sms | email | webauthn`, plus **recovery codes** as a
fallback. A factor has a `status` of `pending` (enrolled, not yet verified) or
`active`.

| Factor | Enrollment | Verify at login |
| --- | --- | --- |
| **TOTP** | `POST /v1/auth/mfa/totp/enroll` → scan `otpauth://` → `/totp/verify` | 6-digit code |
| **SMS** | `POST /v1/auth/mfa/sms/enroll` (sends a code) | code |
| **Email** | `POST /v1/auth/mfa/email/enroll` | code |
| **WebAuthn** | `POST /v1/auth/mfa/webauthn/enroll/{options,verify}` | assertion |
| **Recovery codes** | `POST /v1/auth/mfa/recovery-codes/generate` | one single-use code |

## MFA policy

`config/mfa-policy` controls MFA behavior per project:

- `required_for_admins` — force MFA for privileged users.
- `allowed_factors` — which factor types users may enrol.
- `remember_device` — allow trusting a device to skip repeated MFA.

Set it at `PUT /v1/projects/{id}/admin/config/mfa-policy`.

## Step-up at sign-in

When a user with an active factor signs in with a first factor, the server does
**not** immediately mint a session — it returns a next-step and a `flow_token`:

```json
{ "result_type": "next_step", "next_step": "mfa_required",
  "flow_token": "flw_...", "factors": [{ "id": "fac_totp", "type": "totp", "status": "active" }] }
```

Then the client challenges and verifies the factor:

```bash
# (optional) issue a challenge for a delivery factor (email/sms)
curl -sX POST https://auth.example.com/v1/auth/mfa/challenge \
  -H "X-Client-Id: prj_7Fk2" -H "Content-Type: application/json" \
  -d '{"flow_token":"flw_...","factor_id":"fac_email"}'

# verify → AAL2 session
curl -sX POST https://auth.example.com/v1/auth/mfa/verify \
  -H "X-Client-Id: prj_7Fk2" -H "Content-Type: application/json" \
  -d '{"flow_token":"flw_...","code":"123456"}'
```

Inside a [resumable flow](/rest-api/flows) the same step appears as `step:
mfa_required`; submit `{"action":"verify_mfa","payload":{"code":"…"}}`.

## Security guarantees

- **Second factor is enforced** — you cannot swap the required second factor for a
  first-factor method mid-flow (no `switch_method` at `mfa_required`).
- **Per-challenge attempt limit** — a single MFA challenge is consumed after too
  many wrong codes, so it cannot be brute-forced.
- **AAL2 sessions are real sessions** — the minted AAL2 access token authenticates
  and its refresh token works (persisted like any session).

See the [MFA guide](/guides/mfa) for SDK enrollment code.
