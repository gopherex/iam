---
id: typescript
title: TypeScript SDK reference
sidebar_label: TypeScript SDK
---

# TypeScript SDK reference

`@gopherex/iam-sdk` — an ESM, framework-agnostic client. Every method returns
`{ data, error }` (or `{ error }`) and **never throws** for API failures — you
branch on `error`.

## Install

The package is published to the GitHub Packages registry, so point the
`@gopherex` scope at it in `.npmrc`:

```ini
# .npmrc
@gopherex:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

```bash
npm install @gopherex/iam-sdk
# or: yarn add @gopherex/iam-sdk / pnpm add @gopherex/iam-sdk
```

## Create a client

```ts
import { createIamClient } from '@gopherex/iam-sdk';

const iam = createIamClient({
  baseUrl: 'https://auth.example.com',
  clientId: 'app_web', // sent as X-Client-Id on every call
});
```

`createIamClient(options)` returns
`{ auth, config, account, mfa, webauthn, tokens, oidc, flow, client, init() }`.

### Options

| Option | Default | Purpose |
| --- | --- | --- |
| `baseUrl` | — | API base URL (required) |
| `clientId` | — | `X-Client-Id` (required) |
| `environment` | `"live"` | `X-Environment` |
| `deviceName` | — | `X-Device-Name`, persisted on new sessions |
| `deviceFingerprint` | — | `X-Device-Fingerprint`, refresh-theft detection |
| `storage` | localStorage (browser) / memory | custom `StorageAdapter` |
| `storageKey` | `"iam.session"` | persisted-session key |
| `persistSession` | `true` | write/read the session to storage |
| `autoRefresh` | `true` | refresh before expiry |
| `refreshMarginSeconds` | `30` | how early to refresh |
| `multiTab` | `true` | `BroadcastChannel` cross-tab sync |

### One-shot bootstrap

```ts
const { session, config, error } = await iam.init();
// restores the persisted session AND fetches the project's public config
```

## `iam.auth`

Sign-in/up, verification, OAuth, MFA-at-login, sessions. All return an
`AuthResponse` (`{ data: { session, user, nextStep?, factors?, flowToken? }, error }`)
unless noted.

```ts
// password
await iam.auth.signInWithPassword({ email, password });
await iam.auth.signUp({ email, password, name, locale, consents });

// passwordless
const { data } = await iam.auth.signInWithOtp({ identifier: email, channel: 'email', purpose: 'signin' });
await iam.auth.verifyOtp({ challengeId: data.challengeId, code });
await iam.auth.signInWithMagicLink({ email, redirectTo: '/welcome' });
await iam.auth.verifyMagicLink({ token });

// oauth social (redirects the browser)
iam.auth.signInWithOAuth({ provider: 'google', redirectTo: '/welcome' });
await iam.auth.exchangeOAuthCodeForSession({ code });

// webauthn / passkey (browser)
await iam.auth.signInWithWebAuthn({ email });

// mfa step-up at login
await iam.auth.challengeMfa({ flowToken, factorId });
await iam.auth.verifyMfa({ flowToken, code });
await iam.auth.stepUp({ purpose: 'delete_account', requiredAal: 2 });

// password lifecycle
await iam.auth.resetPasswordForEmail({ email, redirectTo });
await iam.auth.resetPassword({ newPassword, token }); // or { newPassword, challengeId, code }
await iam.auth.updatePassword({ currentPassword, newPassword, revokeOtherSessions: true });

// sessions
await iam.auth.getSession();
await iam.auth.refreshSession();
await iam.auth.signOut();
await iam.auth.signOutAll({ exceptCurrent: true });
```

### Auth state

```ts
const sub = iam.auth.onAuthStateChange((event, session) => {
  // event: INITIAL_SESSION | SIGNED_IN | SIGNED_OUT | TOKEN_REFRESHED | USER_UPDATED
});
sub.unsubscribe();
```

## `iam.flow` — resumable flows

The recommended path for signup/signin/recovery. See the
[Auth flows guide](/guides/auth-flows).

```ts
await iam.flow.start({ kind: 'signup', email, password, name });
await iam.flow.verifyEmail({ code });        // or { token }
await iam.flow.setPassword({ password });     // recovery
await iam.flow.submit('verify_mfa', { code }); // generic step
await iam.flow.resend();
await iam.flow.resume();                       // cookie, then localStorage
await iam.flow.resumeByToken(deepLinkToken);
await iam.flow.abandon();

const off = iam.flow.onChange((state, error) => render(state));
```

When a flow completes, the client automatically calls `acceptFlowSession` and
emits `SIGNED_IN` (a client created by `createIamClient` is pre-wired to `auth`).

## `iam.account`

Profile and self-service for the signed-in user.

```ts
await iam.account.getProfile();
await iam.account.updateProfile({ name, avatarUrl, locale });
await iam.account.listSessions();
await iam.account.revokeSession(id);
await iam.account.revokeAllSessions({ exceptCurrent: true });
await iam.account.getConsents();
await iam.account.acceptConsents([{ key, version }]);
await iam.account.listIdentities();
await iam.account.changeEmailStart({ email, redirectTo });
await iam.account.changeEmailVerify({ code });     // or { token }
await iam.account.startExport();                    // GDPR data export job
await iam.account.deleteAccount({ password });
```

## `iam.mfa`

Factor enrollment (login challenge/verify live on `iam.auth`).

```ts
const { data } = await iam.mfa.enrollTotp({ name: 'Authenticator' });
// data: { factorId, secret, otpauthUrl, qrSvg }
await iam.mfa.verifyTotp({ factorId: data.factorId, code });

await iam.mfa.enrollEmail(email);   // { factorId, challengeId }
await iam.mfa.enrollSms(phone);
await iam.mfa.generateRecoveryCodes({ regenerate: true }); // { codes: [...] }
await iam.mfa.listFactors();
await iam.mfa.removeFactor(id);
```

## `iam.webauthn`

Passkey management (sign-in stays on `iam.auth.signInWithWebAuthn`).

```ts
await iam.webauthn.registerPasskey({ name: 'MacBook' }); // full browser ceremony
await iam.webauthn.listCredentials();
await iam.webauthn.renameCredential(id, 'Work laptop');
await iam.webauthn.deleteCredential(id);
```

## `iam.tokens`

For resource servers / tooling.

```ts
await iam.tokens.introspect(token);        // { active, ... }
await iam.tokens.verify(token, audience);  // signature + expiry + audience
await iam.tokens.revoke({ token });
await iam.tokens.getCurrent();
```

## `iam.oidc`

Build your own consent / device / interaction UI when IAM is your OIDC provider.

```ts
await iam.oidc.getDevice(userCode);
await iam.oidc.approveDevice(userCode);
await iam.oidc.getInteraction(interactionId);
await iam.oidc.consentInteraction(interactionId, { grantedScopes, remember: true });
await iam.oidc.loginInteraction(interactionId);
await iam.oidc.listGrants();
await iam.oidc.revokeGrant(grantId);
```

Consent and reject answer with **either** `redirect_to` **or** `form_post`,
depending on the response mode the client asked for. Handle both — a screen that
reads only `redirect_to` leaves `form_post` clients on a dead page:

```ts
const { data } = await iam.oidc.consentInteraction(interactionId, { grantedScopes });

if (data?.form_post) {
  const form = document.createElement('form');
  form.method = 'post';
  form.action = data.form_post.action;
  for (const [name, value] of Object.entries(data.form_post.fields)) {
    const input = document.createElement('input');
    input.type = 'hidden';
    input.name = name;
    input.value = value;
    form.appendChild(input);
  }
  document.body.appendChild(form);
  form.submit();
} else if (data?.redirect_to) {
  window.location.assign(data.redirect_to);
}
```

Every state-changing call here carries the cookie-mode **CSRF token** for you:
the namespace fetches it from `/v1/csrf`, caches it, attaches it, and retries
once if the server says it is stale. You do not need to perform that handshake
yourself, and a bearer-token caller is unaffected (IAM does not challenge a
request whose credential is not ambient).

A page that has no session of its own to manage — a hosted login, consent or
device screen — can build the namespace alone:

```ts
import { createIamOidc } from '@gopherex/iam-sdk';

// project id comes from GET /v1/oauth/interaction/{id}, which the browser can
// call knowing nothing but the interaction id
const oidc = createIamOidc({ clientId: projectId });
await oidc.consentInteraction(interactionId, { grantedScopes, remember: true });
```

To establish the browser session those screens rely on, start the sign-in flow
in cookie mode — the completed session comes back as HttpOnly cookies instead of
in the response body:

```ts
await iam.flow.start({ kind: 'signin', email, password, cookieMode: true });
```

## Admin surface

The project-admin operations are the generated ones — there is no hand-written
wrapper, because an admin panel drives them directly:

```ts
import {
  putV1ProjectsByProjectIdAdminUsersByUserIdRoles,   // assign IAM roles -> groups claim
  getV1ProjectsByProjectIdAdminConfig,               // every config document at once
  putV1ProjectsByProjectIdAdminConfig,               // apply them, ?dry_run=true to plan
  putV1ProjectsByProjectIdAdminClients,              // desired-state client list, ?prune=true
} from '@gopherex/iam-sdk';
```

App clients carry the OIDC provider's per-client settings: `scopes` (allow-list),
`post_logout_redirect_uris`, `backchannel_logout_uri` and `disabled`.
`dynamically_registered` marks a client that registered itself — it holds a
registration access token and can rewrite the metadata edited here.

## Dynamic client registration (separate client)

Register OAuth clients programmatically (RFC 7591) and let them manage
themselves (RFC 7592):

```ts
import { createIamClientRegistration } from '@gopherex/iam-sdk';

// registering needs an initial access token — a project-admin token, which is
// also what decides the project the client lands in
const registration = createIamClientRegistration({ baseUrl, initialAccessToken: adminToken });

const { data } = await registration.register({
  client_name: 'Reporting',
  redirect_uris: ['https://reports.example.com/cb'],
  token_endpoint_auth_method: 'none',   // public client: no secret, PKCE instead
  scope: 'openid email',
});

// data.client_secret and data.registration_access_token are returned once —
// persist them now, only their digests are kept
await registration.read(data.client_id, data.registration_access_token);
await registration.update(data.client_id, token, metadata);  // replaces, not patches
await registration.delete(data.client_id, token);
```

The registration access token authorizes exactly the one client it was issued
for. `update` is a **replacement** (RFC 7592 §2.2): read first, edit the result,
send it back, or the fields you leave out are cleared.

## `iam.config`

```ts
const { data } = await iam.config.getPublicConfig();
// enabled methods, locales, registration mode — for rendering your login UI
```

## `iam.client` — raw generated operations

For anything not wrapped ergonomically (e.g. admin operations), use the
configured client with the generated functions:

```ts
import { createIamClient, getV1ProjectsByProjectIdAdminApiKeys } from '@gopherex/iam-sdk';

const iam = createIamClient({ baseUrl, clientId: 'web' });
const { data } = await getV1ProjectsByProjectIdAdminApiKeys({
  client: iam.client,
  path: { project_id: 'prj_7Fk2' },
});
```

`iam.client` carries `X-Client-Id`, the bearer, and the refresh-on-401
interceptor.

## Admin invites (separate client)

Invite management uses a project-admin token, not a user session:

```ts
import { createIamInvitesAdmin } from '@gopherex/iam-sdk';

const invites = createIamInvitesAdmin({ baseUrl, projectId: 'prj_7Fk2', adminToken });
await invites.create({ email: 'ada@example.com', expiresAt });
await invites.list();
await invites.revoke(inviteId);
```

## Error handling

```ts
import { IamAuthError } from '@gopherex/iam-sdk';

const { data, error } = await iam.auth.signInWithPassword({ email, password });
if (error) {
  // error is IamAuthError: { message, code, status? }
  console.error(error.code, error.status);
}
```

See the [Errors reference](/rest-api/errors) for the full code table.

## Sessions & persistence

- Access + refresh tokens persist under `localStorage["iam.session"]` (override
  with `storageKey`, or pass a custom `storage`).
- Auto-refresh fires `refreshMarginSeconds` before expiry; a `401` on any call
  triggers one refresh + retry.
- Cross-tab sync via `BroadcastChannel('iam:auth')`; disable with
  `multiTab: false`.
- For SSR/Node, import `MemoryStorage` or set `persistSession: false`.
