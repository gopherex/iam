---
id: passwordless
title: Passwordless (OTP & magic link)
sidebar_label: Passwordless
---

# Passwordless sign-in

Two passwordless methods share the same primitives: **OTP** (a code the user
types) and **magic link** (a link the user clicks). Both must be enabled in the
project's auth config.

## Email / SMS OTP

```ts
// 1. request a code
const { data, error } = await iam.auth.signInWithOtp({
  identifier: 'ada@example.com',
  channel: 'email',      // email | sms | whatsapp
  purpose: 'signin',     // signin | signup | verify
});
if (error) return showError(error.message);

// 2. user enters the emailed/texted code
const res = await iam.auth.verifyOtp({ challengeId: data.challengeId, code });
if (res.error) return showError(res.error.message); // invalid_otp / challenge_expired
// res.data.session is live
```

Under the hood: `POST /v1/auth/otp/start` → `Challenge`, then
`POST /v1/auth/otp/verify` → `AuthResult`.

## Magic link

```ts
// 1. send the link (server emails a URL back to redirectTo with a token)
await iam.auth.signInWithMagicLink({
  email: 'ada@example.com',
  redirectTo: 'https://app.example.com/auth/callback',
});

// 2. on your callback page, exchange the token in the URL
const token = new URL(location.href).searchParams.get('token');
const { data, error } = await iam.auth.verifyMagicLink({ token });
```

There is also a server-side `GET /v1/auth/magic-link/callback?token&redirect_to`
that sets session cookies and `302`s — use it when you prefer cookie sessions
over token handling in JS.

## Errors to handle

| `error.code` | Meaning |
| --- | --- |
| `invalid_otp` | wrong code — let the user retry |
| `challenge_expired` | code timed out — call start again |
| `rate_limited` | too many requests — back off |
| `registration_closed` | `purpose: signup` but the project is closed |

:::tip Prefer the flow controller
For signup/signin, `iam.flow` with `method: 'phone_otp'` or `'magic_link'`
handles the challenge/verify/resend cycle for you — see
[Auth flows](/guides/auth-flows).
:::
