---
id: errors
title: Errors
sidebar_label: Errors
---

# Errors

## The envelope

Every error uses one shared shape, returned on any operation:

```json
{
  "error": {
    "code": "invalid_credentials",
    "message": "The email or password is incorrect.",
    "request_id": "req_...",
    "details": { }
  }
}
```

**Branch on `error.code`** — a stable machine string. Never parse the
`message` (localized) or key logic off the HTTP status.

:::note Errors inside a flow
Within a [resumable flow](/rest-api/flows), transient input errors surface as
`FlowState.error {code, message}` while `status` stays `pending` — do **not** map
those to an HTTP status.
:::

## Code ↔ status table

| HTTP | Codes |
| --- | --- |
| 400 | `bad_request`, `unsupported_grant` |
| 401 | `invalid_credentials`, `unauthorized`, `invalid_token`, `token_expired`, `token_revoked`, `session_expired`, `device_mismatch`, `mfa_invalid`, `invalid_otp`, `challenge_invalid` |
| 403 | `forbidden`, `account_suspended`, `account_banned`, `account_locked`, `email_not_verified`, `phone_not_verified`, `mfa_required`, `mfa_factor_not_allowed`, `step_up_required`, `registration_closed`, `invitation_required`, `captcha_required`, `captcha_invalid`, `consent_required`, `invalid_csrf` |
| 404 | `not_found`, `user_not_found`, `session_not_found`, `project_not_found`, `environment_not_found`, `client_not_found`, `connection_not_found`, `domain_not_found` |
| 409 | `conflict`, `email_exists`, `phone_exists`, `identity_exists`, `already_linked`, `domain_taken` |
| 410 | `challenge_expired`, `token_already_used`, `flow_not_found`, `flow_expired` |
| 422 | `validation_failed`, `weak_password`, `password_reused` |
| 429 | `rate_limited`, `flow_resend_too_soon` |
| 501 | `not_implemented` |
| 502 | `provider_error`, `sso_error`, `scim_error` |
| 503 | `service_unavailable` |
| 500 | `internal_error` |

## Handling pattern

```ts
const { data, error } = await iam.auth.signInWithPassword({ email, password });
if (error) {
  switch (error.code) {
    case 'invalid_credentials': return showError('Wrong email or password');
    case 'account_locked':      return showError('Too many attempts, try later');
    case 'email_not_verified':  return startEmailVerification();
    default:                    return showError(error.message);
  }
}
```
