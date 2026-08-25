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
