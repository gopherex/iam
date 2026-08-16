---
id: oauth-social
title: OAuth social login
sidebar_label: OAuth social login
---

# OAuth social login

Let users sign in with Google, GitHub, and other configured providers. Providers
are added by a project admin (`oauth-providers` admin API) and surfaced to your
UI via the public config.

## Discover enabled providers

```ts
const { data } = await iam.config.getPublicConfig();
// render a button per data.oauthProviders entry
```

Or the raw endpoint: `GET /v1/auth/oauth/providers`.

## Redirect flow (browser)

```ts
// 1. kick off — this navigates the browser to the provider
iam.auth.signInWithOAuth({
  provider: 'google',
  redirectTo: 'https://app.example.com/auth/callback',
});
```

The browser goes to `GET /v1/auth/oauth/google/start`, then to Google, then
back to your `redirectTo` with a `code` in the query string.

```ts
// 2. on your callback page
const code = new URL(location.href).searchParams.get('code');
const { data, error } = await iam.auth.exchangeOAuthCodeForSession({ code });
if (error) return showError(error.message); // provider_error / sso_error
// data.session is live
```

The server also supports a pure-redirect variant
(`GET /v1/auth/oauth/{provider}/callback`) that sets session cookies and
`302`s straight to your app — no JS exchange needed.

## Linking a provider to an existing account

For a signed-in user who wants to add "Sign in with GitHub":

```ts
// start/callback pair, then unlink
// GET  /v1/auth/oauth/{provider}/link/start
// GET  /v1/auth/oauth/{provider}/link/callback
// POST /v1/auth/oauth/{provider}/unlink
await iam.account.listIdentities(); // shows linked providers
```

## PKCE

The start endpoint accepts `code_challenge` / `code_verifier` for public
clients. The SDK generates and stores the verifier for you when you use
`signInWithOAuth`; if you build the redirect yourself, pass your own
`code_challenge` and keep the `code_verifier` for the exchange.

## Errors

| `error.code` | Meaning |
| --- | --- |
| `provider_error` | the upstream IdP rejected the request |
| `sso_error` | misconfigured connection (check admin `/test`) |
| `already_linked` | that identity is linked to another account |
