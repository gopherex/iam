# @gopherex/iam-sdk

TypeScript SDK for the IAM API. Two layers:

- **Generated client** — one typed function per API operation plus the
  request/response types, generated from `openapi/openapi.yaml` with
  [`@hey-api/openapi-ts`](https://heyapi.dev).
- **`iam.auth`** — a Supabase-style auth client that manages the session,
  refreshes tokens automatically, and syncs across browser tabs.

## Auth (high-level)

```ts
import { createIamClient } from '@gopherex/iam-sdk';

const iam = createIamClient({
  baseUrl: 'https://iam.acme.com',
  clientId: 'web', // sent as X-Client-Id
});

// React to auth changes (fires INITIAL_SESSION immediately).
iam.auth.onAuthStateChange((event, session) => {
  // SIGNED_IN | SIGNED_OUT | TOKEN_REFRESHED | USER_UPDATED | INITIAL_SESSION
  console.log(event, session?.user.id);
});

// Password sign-in.
const { data, error } = await iam.auth.signInWithPassword({
  email: 'a@b.com',
  password: 'secret',
});
if (error) throw error;
if (data.nextStep) {
  // e.g. 'mfa_required' | 'verify_email' — drive the next step
}
// data.session / data.user are set on success.

await iam.auth.signOut();
```

What `iam.auth` does for you:

- **Persists** the session (localStorage by default; pass `storage` for
  SSR / React Native / Node).
- **Auto-refreshes** the access token before it expires, and retries once on a
  `401` (single-flight, no storms).
- **Attaches** the bearer token to every authenticated request automatically.
- **Syncs across tabs** via `BroadcastChannel` (sign-in/out/refresh propagate).
- Surfaces server **next steps** (`mfa_required`, `verify_email`, …) instead of
  faking success.

Sign-in flows: `signInWithPassword`, `signUp`, `signInWithOtp` / `verifyOtp`,
`signInWithMagicLink` / `verifyMagicLink`, `signInWithOAuth` (browser redirect) /
`exchangeOAuthCodeForSession`, `signInWithWebAuthn`, `signInAnonymously`,
email/phone verification, password reset/update, step-up, and MFA challenge /
verify. Plus `getSession`, `getUser`, `refreshSession`, `onAuthStateChange`,
`signOut`.

Pass `locale` (`'ru'`, `'en'`, `'ru-RU'`, etc.) on email-producing calls when
the UI language is known. It is forwarded to email/OTP/magic-link/flow delivery;
otherwise IAM falls back to the user's stored locale, then the project default,
then English.

```ts
await iam.auth.signUp({ email: 'a@b.com', password: 'secret', locale: 'ru' });
await iam.auth.signInWithOtp({ identifier: 'a@b.com', channel: 'email', locale: 'ru' });
await iam.auth.signInWithMagicLink({
  email: 'a@b.com',
  redirectTo: `${window.location.origin}/auth/callback`,
  locale: 'ru',
});
await iam.flow.start({ kind: 'signup', email: 'a@b.com', password: 'secret', locale: 'ru' });
```

Email verification can be completed by the numeric code or by the link token:

```ts
await iam.auth.verifyEmail({ challengeId, code: '260129' });
await iam.auth.verifyEmail({ token });

// For resumable flows, the typed helper submits the current verify_email step.
await iam.flow.verifyEmail({ code: '260129' });
await iam.flow.verifyEmail({ token });
```

Password reset supports the same code-or-link-token pattern:

```ts
await iam.auth.resetPasswordForEmail({
  email: 'a@b.com',
  redirectTo: `${window.location.origin}/auth/reset-password`,
});

await iam.auth.resetPassword({ token, newPassword: 'N3wStr0ng!Pass99' });
await iam.auth.resetPassword({ challengeId, code: '260129', newPassword: 'N3wStr0ng!Pass99' });

// Recovery flow callback: first consume the email link token, then collect the
// new password on the set_password step.
await iam.flow.verifyEmail({ token });
await iam.flow.setPassword({ password: 'N3wStr0ng!Pass99' });
```

## Persistence

By default the SDK persists the session under `iam.session` in `localStorage`
when Web Storage is available. The persisted value is JSON containing:

- `access_token`
- `refresh_token` when issued
- `token_type`
- absolute `expires_at` in epoch milliseconds
- the resolved `user`

Storage is validated on load; malformed values and expired sessions without a
refresh token are ignored and removed.

Disable persistence entirely with `persistSession: false`. In that mode the SDK
does not read, write, or probe `localStorage`, and cross-tab sync is disabled.

```ts
const iam = createIamClient({
  baseUrl: 'https://iam.acme.com',
  clientId: 'web',
  persistSession: false,
});
```

Use `storage` and `storageKey` to plug in React Native, SSR, encrypted storage,
or test storage. Storage methods may be sync or async.

## Raw operations (management / admin)

The generated functions are re-exported for everything else. Pass the configured
`client` returned by `createIamClient`; it attaches `X-Client-Id`, bearer auth,
and the refresh-on-401 interceptor.

```ts
import { createIamClient, getV1ProjectsByProjectIdAdminApiKeys } from '@gopherex/iam-sdk';

const iam = createIamClient({ baseUrl: 'https://iam.acme.com', clientId: 'web' });
const { data } = await getV1ProjectsByProjectIdAdminApiKeys({
  client: iam.client,
  path: { project_id: 'p1' },
});
```

## Build

`yarn generate` regenerates `src/gen/` from the spec; `yarn build` bundles
`src/` to `dist/` (tsup, ESM + d.ts). Generated `src/gen/` and `dist/` are not
committed.
