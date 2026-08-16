---
id: sdk-quickstart
title: SDK quickstart
sidebar_label: SDK quickstart
---

# SDK quickstart

Get a working sign-up → sign-in → signed-out loop in a browser app with
`@gopherex/iam-sdk`. Full surface in the [SDK reference](/sdk/typescript).

## 1. Install & configure the registry

```ini
# .npmrc
@gopherex:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

```bash
npm install @gopherex/iam-sdk
```

## 2. Create the client once

```ts
// iam.ts
import { createIamClient } from '@gopherex/iam-sdk';

export const iam = createIamClient({
  baseUrl: import.meta.env.VITE_IAM_URL, // e.g. https://auth.example.com
  clientId: 'app_web',                    // your project's app-client id
});
```

## 3. Restore any existing session on boot

```ts
const { session } = await iam.init();
if (session) {
  // already signed in — render the app
}
```

## 4. Sign up (recommended: the flow controller)

The [resumable flow](/guides/auth-flows) handles email verification, MFA and
consents for you:

```ts
iam.flow.onChange((state, error) => {
  render(state.step);          // e.g. 'verify_email'
  if (error) showError(error); // wrong code etc., flow stays alive
});

await iam.flow.start({ kind: 'signup', email, password, name });
// user gets an email → your UI collects the code:
await iam.flow.verifyEmail({ code });
// on completion the SDK emits SIGNED_IN automatically
```

## 5. Sign in with a password

```ts
const { data, error } = await iam.auth.signInWithPassword({ email, password });
if (error) return showError(error.message);

if (data.nextStep === 'mfa_required') {
  // step up — see the MFA guide
} else {
  // data.session is live
}
```

## 6. React to auth changes

```ts
iam.auth.onAuthStateChange((event, session) => {
  if (event === 'SIGNED_IN')  router.push('/app');
  if (event === 'SIGNED_OUT') router.push('/login');
});
```

## 7. Sign out

```ts
await iam.auth.signOut();               // this device
await iam.auth.signOutAll({ exceptCurrent: true }); // everywhere else
```

## What you get for free

- Access + refresh tokens persisted and auto-refreshed before expiry.
- A `401` on any call transparently refreshes once and retries.
- Cross-tab session sync via `BroadcastChannel`.

Next: [Auth flows](/guides/auth-flows) · [Passwordless](/guides/passwordless) ·
[MFA](/guides/mfa).
