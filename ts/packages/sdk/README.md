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
`exchangeCodeForSession`, `signInWithWebAuthn`. Plus `getSession`, `getUser`,
`refreshSession`, `onAuthStateChange`, `signOut`.

## Raw operations (management / admin)

The generated functions are re-exported for everything else; they use the same
client configured by `createIamClient`, so the bearer token is attached
automatically.

```ts
import { getV1ProjectsByProjectIdAdminApiKeys } from '@gopherex/iam-sdk';

const { data } = await getV1ProjectsByProjectIdAdminApiKeys({
  path: { project_id: 'p1' },
  headers: { 'X-Client-Id': 'web' },
});
```

## Build

`yarn generate` regenerates `src/gen/` from the spec; `yarn build` bundles
`src/` to `dist/` (tsup, ESM + d.ts). Generated `src/gen/` and `dist/` are not
committed.
