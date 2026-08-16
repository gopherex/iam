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

The canonical **issuer** is `/p/{project_id}/e/{environment}`. Tokens are RS256
JWTs minted by the project's signing key; discovery and JWKS are per project and
environment.

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
