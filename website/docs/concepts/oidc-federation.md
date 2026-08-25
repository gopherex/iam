---
id: oidc-federation
title: OIDC provider & federation
sidebar_label: OIDC & federation
---

# OIDC provider & federation

IAM plays both sides of identity federation: it can **be** an OpenID Connect
provider for your apps, and it can **consume** upstream IdPs and SCIM.

## IAM as an OIDC provider

Your applications can integrate IAM as a standard OAuth 2.0 / OpenID Connect
provider. The provider endpoints live under `/oauth2/*`, with per-project
discovery and JWKS:

```
/oauth2/authorize
/oauth2/token
/oauth2/userinfo
/oauth2/introspect
/oauth2/revoke
/oauth2/logout
/oauth2/par                    # RFC 9126 pushed authorization requests
/oauth2/device_authorization   # RFC 8628 device grant

/p/{project_id}/e/{env}/.well-known/openid-configuration
/p/{project_id}/e/{env}/.well-known/jwks.json
```

The canonical **issuer** is the absolute URL
`{public_url}/p/{project_id}/e/{environment}`, built from the deployment's
configured public base URL (`service.http.public_url`). Every URI in the
discovery document is absolute, and the issuer is a literal prefix of the
discovery document's own URL — the property conforming clients (`go-oidc`, and
therefore ArgoCD, oauth2-proxy, kube-oidc-proxy) verify. Tokens are RS256
JWTs minted by the project's signing key; discovery and JWKS are per project and
environment.

### Authorization request validation

`/oauth2/authorize` resolves the client **before** it persists anything. An
unknown or disabled `client_id`, or a `redirect_uri` the client has not
registered, is answered with a `400` error envelope and **no** redirect — the
user-agent is never sent to a URI the client did not register, and no
interaction record is created (RFC 6749 §4.1.2.1). Registered `redirect_uri`
values are matched exactly; there is no prefix or wildcard matching.

Once the client and its `redirect_uri` check out, a bad request parameter is
reported the other way round: a `302` back to the registered `redirect_uri`
carrying `error` (e.g. `unsupported_response_type`, `invalid_request`) and the
original `state`.

A client can be switched off with `disabled: true` instead of being deleted;
the authorization endpoint then treats it exactly like an unknown `client_id`.

### Pushed authorization requests (PAR)

A client can lodge its authorization request over the authenticated back channel
instead of putting it in the browser's query string:

```bash
# 1. push — client-authenticated, returns a single-use request_uri
curl -sX POST https://auth.example.com/oauth2/par \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d response_type=code -d client_id="$CLIENT_ID" \
  -d redirect_uri=https://app.example.com/cb -d scope=openid \
  -d code_challenge="$CHALLENGE" -d code_challenge_method=S256

# 2. redeem — only client_id and request_uri go through the user-agent
https://auth.example.com/oauth2/authorize?client_id=$CLIENT_ID&request_uri=urn:ietf:params:oauth:request_uri:…
```

The pushed request is validated **at push time**, exactly as the authorization
endpoint would validate it — client, `redirect_uri`, `response_type`, `scope`,
PKCE — so a request that could never be authorized is refused immediately rather
than after the user has been walked through a login screen. The `client_id` in
the body must be the authenticated client's own.

Redemption is single-use and bound to the client that pushed it; a replayed,
expired or foreign `request_uri` is `invalid_request_uri`. When `request_uri` is
present every other query parameter except `client_id` is ignored (RFC 9126 §4),
so nothing in the browser's URL can override what the client lodged.

Request objects (`request`, RFC 9101) are not supported and are refused rather
than silently ignored.

### Reusing an existing session (`prompt`, `max_age`)

An authorization request from a browser that already holds an IAM session and
has granted these scopes before is answered with a **code**, not a login screen.
That is the difference between single sign-on and signing on once per
application; `/oauth2/authorize` reads the session cookie for exactly this.

`prompt` is honored:

| Value | Effect |
| --- | --- |
| *(absent)* | reuse the session and a remembered grant when both are there |
| `none` | never show UI — answer with a code, or `login_required` / `consent_required` |
| `login`, `select_account` | re-authenticate even though the session is valid |
| `consent` | show the consent screen even though the scopes were granted |

`none` cannot be combined with the others: asking for no UI and for a login
screen at once is a contradiction, and is answered `invalid_request`. `prompt=none`
is what an SPA runs in a hidden iframe to renew silently — rendering a login page
there is the same as failing.

`max_age` bounds how old the relied-upon authentication may be. A session past it
is re-authenticated however valid it still is, and the resulting id_token's
`auth_time` says when that happened.

`response_mode=fragment` returns the code in the fragment instead of the query,
keeping it out of the `Referer` header and the redirect target's logs.

### The response names the issuer (RFC 9207)

Every authorization response — success and error alike — carries `iss`, the
provider that produced it. A client registered with more than one provider can
otherwise be steered into redeeming a perfectly honest code at the wrong one;
the code is genuine, the destination is not. Discovery advertises
`authorization_response_iss_parameter_supported: true`, and a client that checks
it closes that class of mix-up.

### Scopes a client may request

A client can be given a `scopes` allow-list. A request for anything outside it is
refused with `invalid_scope` rather than quietly trimmed — issuing a narrower
token than was asked for produces failures a long way from their cause. An empty
list means no restriction, so a project that has not thought about it is not
silently narrowed.

### Client authentication with a key (`private_key_jwt`)

A client can authenticate at the token endpoint by signing a short-lived
assertion with a key only it holds, instead of presenting a secret both sides
know (RFC 7523 §3). Nothing shareable ever travels, so there is no secret to leak
from our storage or from the client's configuration — which is why FAPI requires
it.

Publish the client's public keys as `jwks` (inline) or `jwks_uri` on the client,
then send `client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer`
and `client_assertion`. The assertion must name the client as both `iss` and
`sub`, be addressed to the token endpoint, expire within five minutes, and carry
a `jti` — which is **single-use**: replaying an assertion is how a captured one
becomes a second authentication.

`client_secret_jwt` is deliberately not supported. It needs the shared secret in
recoverable form to compute an HMAC, and we store only its sha256 — weakening
that for a method strictly weaker than `private_key_jwt` is a bad trade.

### Signed request objects (`request`)

The whole authorization request can arrive as a JWT the client signed (RFC 9101):

```
GET /oauth2/authorize?client_id=app_web&request=eyJhbGciOi...
```

Everything in a query string is modifiable by whatever sits between the client
and the browser; a signed object is not. Parameters inside the object **win** over
the query — that is the point — and `client_id` must appear in both and agree,
since we have to know whose key to verify against before trusting anything
inside.

The by-reference form of RFC 9101 (a `request_uri` pointing at a client-hosted
URL) is not accepted: here `request_uri` means a **pushed** request (RFC 9126),
which gives the same protection by lodging the request over the client's
authenticated back channel and never asks us to fetch from an address the browser
chose.

### PKCE

PKCE (RFC 7636) is enforced, and `S256` is the only method — `plain` is not
supported and is not advertised in `code_challenge_methods_supported`.

- **Public clients** (`spa`, `native`) **must** send `code_challenge` and
  `code_challenge_method=S256`. They hold no secret, so the authorization code
  is the only thing between an attacker who intercepts the redirect and a token.
  A request without a challenge is bounced back to the client's registered
  `redirect_uri` with `error=invalid_request`.
- **Confidential clients** (`web`, `machine`) authenticate with a client secret
  at the token endpoint and may omit PKCE — but when they send a challenge it is
  enforced identically.

The challenge is bound to the issued authorization code. The token endpoint
rejects the exchange with `invalid_grant` unless the presented `code_verifier`
hashes to it, and equally rejects a `code_verifier` presented for a code that
was issued without a challenge — otherwise an attacker holding a stolen code
could simply strip the challenge (RFC 7636 §4.6). A failed exchange does not
consume the code, so the legitimate client can still finish its flow.

Confidential clients are authenticated against the secrets issued for them under
`admin/apps/{app_id}/secrets`. A client may hold several at once, so a secret can
be rotated in before the old one is deleted; any issued secret authenticates.

### Verified against a real client

The provider is checked against `oauth2-proxy` and against the `go-oidc` library
it and ArgoCD are built on, not only against assertions of our own: discovery,
the authorization code exchange with `client_secret_basic`, id_token
verification against the published JWKS, and claim resolution through
`/oauth2/userinfo`. An assertion we write ourselves will happily agree with a
document no standards client accepts.

Two things such a client insists on that are easy to miss:

- `state` comes back on the authorization response unmodified (RFC 6749
  §4.1.2). Relying parties treat it as their CSRF defense and refuse a callback
  without it.
- `email_verified` must be true before most proxies will accept the login.
  oauth2-proxy rejects an unverified address unless started with
  `--insecure-oidc-allow-unverified-email`, so verify the address (or turn on
  verification at signup) before wiring one up.

### Logout

`GET /oauth2/logout` with an `id_token_hint` ends the IAM session that token
names — the session record, its refresh tokens and the browser's session cookie
— and notifies every other client holding a grant on it.

- `post_logout_redirect_uri` is followed only when the hint identifies the client
  that **registered** it, matched exactly against the client's
  `post_logout_redirect_uris`. Anything else lands on `/`; the parameter is
  attacker-controlled, and following it unchecked makes every logout link an open
  redirect. `state` is echoed back.
- without an `id_token_hint` nothing is ended and no redirect is honored: there
  is no way to know whose session, or whose redirect, is being asked for.

**Back-channel logout** is now actually sent. A client with a
`backchannel_logout_uri` receives a POST carrying a signed `logout_token` (aud =
the client, `sid` = the ended session, `sub` = the user, the
`backchannel-logout` event, and deliberately no `nonce`). Delivery runs through
the outbox, so a slow relying party cannot hold up a logout and a failed POST is
retried instead of lost, and it goes out through the same hardened client
webhooks use.

Any session ending triggers it, not only RP-initiated logout: an admin revoking
a session, or the user signing out of IAM, notifies the relying parties too.

### Claims in the id_token

The scopes a client is granted decide what the id_token says about the person:

| Scope | Claims |
| --- | --- |
| `openid` | `sub`, plus the standard `iss` / `aud` / `exp` / `at_hash` |
| `email` | `email`, `email_verified` |
| `profile` | `name`, `locale`, `phone_number`, `phone_number_verified`, `updated_at` |
| `groups` | `groups` — see below |

`email` matters out of proportion: oauth2-proxy, Grafana and most relying parties
identify the signed-in person by it and refuse a token without one. `/oauth2/userinfo`
returns the same claims for the same scopes, so a client that falls back to it
gets the same answer — and a client granted only `openid` gets a subject and
nothing else from either.

### Roles in the token (`groups`)

Relying parties (ArgoCD, Grafana, …) need to know *who* the user is inside your
organisation, not just that they authenticated. IAM assigns **roles** — plain
labels — to a user per project environment:

```bash
# the user ends up with exactly these roles in this environment
curl -sX PUT https://auth.example.com/v1/projects/prj_7Fk2/admin/users/usr_9/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: live" \
  -H "Content-Type: application/json" \
  -d '{"roles":["ops","platform:admin"]}'
```

A client that is granted the **`groups`** scope receives them as a `groups`
claim (array of strings) in both the access token and the id_token. The values
come only from what an admin assigned; a client can ask for the scope, but it
cannot ask for a role. A granted scope always produces the claim — an empty
array when the user has no roles — so "asked and has none" is distinguishable
from "did not ask". Without the scope the claim is absent entirely.

Roles are environment-scoped: the same person can be `ops` in `test` and
`viewer` in `live`. Allowed characters are letters, digits and `_ - . : /`.

This is deliberately separate from a token profile's `claims_template`, which is
a static per-client map and cannot express anything about the individual user.

### Interaction ownership

An interaction is created **unbound** by `/oauth2/authorize`, which is a public
endpoint — there is no user yet, that is the whole point. Its id then travels
through the user-agent, so it ends up in browser history, referrers and logs and
must not be treated as a capability on its own.

The first authenticated caller **claims** the interaction at
`POST /v1/oauth/interaction/{id}/login`: the interaction is bound to that
session and account. From then on:

- a different session cannot take it over, and cannot consent on it;
- consent must name the account that logged in;
- an interaction nobody has logged into cannot be consented at all;
- a credential without a browser session (an admin token, an API key, a client
  credential) cannot claim or consent on an interaction — it has no session to
  bind;
- interactions expire (10 minutes) and an expired one is `flow_expired`, not
  resumable.

`GET /v1/oauth/interaction/{id}` stays public: the login/consent screen needs the
requested scopes before the user has signed in.

### The hosted sign-in and consent screens

IAM ships the screens the authorization flow needs, so a project gets a working
provider without writing any UI:

| Path | Screen |
| --- | --- |
| `/oauth/interaction/{id}` | sign in, continue as the signed-in user, consent |
| `/oauth/device` | enter a device `user_code`, then approve or refuse it |

`/oauth2/authorize` redirects the browser to the interaction screen with nothing
but the interaction id; the page reads
`GET /v1/oauth/interaction/{id}` for the application name, the requested scopes,
the tenant and the project's locales, and renders in the project's language.

**Already signed in.** If the browser holds a valid IAM session the screen offers
it — "Continue as ada@example.com" — instead of asking for a password. That is
what makes this single sign-on: the session is established once, and the second
relying party only needs a consent decision (and not even that, if the user
ticked "remember" the first time). A browser session exists because the sign-in
flow is driven with `cookie_mode`, which returns the session as HttpOnly cookies
rather than in the response body.

The sign-in step is the same resumable flow engine, and literally the same step
components, as the console's `/flow` page — password, OTP, magic link, passkey,
MFA, consent documents and recovery all work here because they are not
reimplemented here.

Every state-changing call the screens make is cookie-mode with a CSRF token from
`GET /v1/csrf`; the pages never hold a token themselves.

### Building your own consent / device / interaction UI

The runtime exposes helper endpoints (and SDK methods, see
[`iam.oidc`](/sdk/typescript)) so you can render the login/consent screens
yourself:

- Device grant: `getDevice(userCode)`, `approveDevice`, `denyDevice`
- Interaction: `getInteraction`, `loginInteraction`, `consentInteraction`,
  `rejectInteraction`
- Grants: `listGrants`, `revokeGrant` (also admin-visible per user)

## Federation — consuming upstream IdPs

Under `/v1/projects/{id}/admin/sso/connections` you configure **SAML** and
**OIDC** connections. Runtime SSO endpoints then let users sign in through them:

- **SAML** — `/v1/sso/saml/{connection_id}/{metadata,login,acs,slo}`
- **OIDC** — `/v1/sso/oidc/{connection_id}/{start,callback}`

Connections support `test` and `rotate-certificate`, and are bound to verified
**domains** (admin `domains`).

## SCIM 2.0 provisioning

Upstream identity providers can provision users and groups via SCIM 2.0:

```
/v1/scim/v2/{connection_id}/Users
/v1/scim/v2/{connection_id}/Groups
```

Each connection issues its own **SCIM token** (`scimToken`) under the admin
`connections` surface; the token scopes provisioning to that one connection.

## Social login (OAuth in)

Distinct from enterprise SSO, **social login** lets end users sign in with
Google/GitHub/etc. Configure providers under
`/v1/projects/{id}/admin/oauth-providers` (client id + secret, encrypted at
rest), then use the runtime OAuth endpoints or the SDK — see the
[Social login guide](/guides/oauth-social).
