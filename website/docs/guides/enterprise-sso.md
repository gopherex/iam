---
id: enterprise-sso
title: Enterprise SSO & SCIM
sidebar_label: Enterprise SSO
---

# Enterprise SSO & SCIM

Enterprise SSO is IAM **consuming** an upstream identity provider: a customer's
Okta, Entra ID or Google Workspace authenticates their staff, and IAM turns that
into an IAM session. It is the opposite direction from
[IAM as an OIDC provider](/concepts/oidc-federation), and the two coexist — you
can be an IdP to your apps and an SP to your customers' IdPs at the same time.

A **connection** is one customer's IdP. Its **domains** are the email domains
that route to it. **SCIM** lets that IdP push user and group changes into IAM.

## Connections

```bash
# create — type and name only; the IdP details come next
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/sso/connections \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"type":"saml","name":"Acme Okta","domains":["acme.com"]}'
# -> { "connection": { "id": "con_1", "type": "saml", "status": "active" } }

curl -s .../admin/sso/connections           -H "Authorization: Bearer $ADMIN_TOKEN"
curl -s .../admin/sso/connections/con_1     -H "Authorization: Bearer $ADMIN_TOKEN"
curl -sX DELETE .../admin/sso/connections/con_1 -H "Authorization: Bearer $ADMIN_TOKEN"
```

The `config` and `attribute_mapping` fields of the create request are accepted by
the schema but **ignored** — configuration is applied with `PATCH`, which takes
a flat patch rather than a nested object:

```bash
# SAML: point at the IdP's metadata
curl -sX PATCH .../admin/sso/connections/con_1 -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"saml_metadata_url":"https://acme.okta.com/app/xxx/sso/saml/metadata"}'

# OIDC: the upstream client credentials
curl -sX PATCH .../admin/sso/connections/con_2 -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"oidc_issuer":"https://acme.okta.com","oidc_client_id":"…","oidc_client_secret":"…",
       "oidc_scopes":["openid","email","profile"]}'
```

Accepted patch keys: `name`, `display_name`, `enabled`, `domain`, `domains`,
`metadata`, `saml_metadata_url`, `saml_metadata_xml`, `oidc_issuer`,
`oidc_discovery_url`, `oidc_client_id`, `oidc_client_secret`, `oidc_scopes`,
`oidc_response_mode`. Anything else in the patch is dropped, not rejected.

:::caution The configuration surface is incomplete today
The connection store, the SAML and OIDC runtime flows, domains and SCIM all
exist, but some fields a working connection needs have **no API path**:

- **SAML** — the SP `entity_id` and `acs_url` that the metadata document
  advertises and that assertion validation matches against. Without them the
  rendered SP metadata is incomplete and an assertion cannot be consumed.
- **OIDC** — the provider's `auth_url`, `token_url` and `jwks_url`. Setting
  `oidc_issuer` or `oidc_discovery_url` does not populate them, because
  discovery is not fetched.

Until those are settable, treat enterprise SSO as configurable only by writing
the connection's config directly in the database. The rest of this page
describes the surface as it stands.
:::

`POST .../connections/{id}/test` returns a URL that starts the flow, for checking
a connection before handing it to a customer. `POST .../connections/{id}/rotate-certificate`
issues a new SP signing certificate — the customer must re-import the metadata
afterwards.

## Domains

Domains are how an email address finds its connection.

```bash
curl -sX POST .../admin/domains -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"domain":"acme.com","connection_id":"con_1"}'

curl -sX POST .../admin/domains/dom_1/verify -H "Authorization: Bearer $ADMIN_TOKEN"
```

A domain starts `pending` and becomes `verified`.

:::note Verification is a state flip, not a DNS check
`/verify` marks the domain verified; IAM does not resolve a DNS challenge. Prove
ownership out of band before calling it, or anyone who can add a domain can claim
one.
:::

## Signing in through a connection

```
GET  /v1/sso/connections/resolve?email=ada@acme.com          → which connection to use
GET  /v1/sso/saml/{connection_id}/metadata                   → SP metadata XML for the IdP
GET  /v1/sso/saml/{connection_id}/login?redirect_to&state    → 302 to the IdP
GET  /v1/sso/oidc/{connection_id}/start?redirect_to&state&login_hint
```

`/metadata` is the document you hand the customer's IdP administrator; it is
rendered from the connection's stored SP config (entity id, ACS URL, signing
certificate when one exists).

The IdP returns to `POST /v1/sso/saml/{connection_id}/acs` or
`GET /v1/sso/oidc/{connection_id}/callback`. IAM provisions or matches the user,
mints a session, and redirects to `redirect_to` with a one-time code, which the
app exchanges:

```bash
curl -sX POST https://auth.example.com/v1/sso/exchange \
  -H "X-Client-Id: prj_7Fk2" -H "Content-Type: application/json" \
  -d '{"code":"…"}'
# -> the normal session response
```

The typical login screen: ask for the email, call `resolve`, and either send the
browser to the connection's `start`/`login` URL or fall back to your normal
sign-in form.

`POST /v1/sso/saml/{connection_id}/slo` is the SAML single-logout endpoint the
IdP posts to.

## SCIM 2.0 provisioning

Each connection issues its own SCIM credential, so a token can only provision
into the connection it belongs to.

```bash
curl -sX POST .../admin/sso/connections/con_1/scim/tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"okta-provisioning","expires_at":"2027-01-01T00:00:00Z"}'
# -> { "token": { "id": "…", "name": "okta-provisioning" }, "secret": "<shown once>" }
```

Give the IdP the base URL and that secret as its bearer token:

```
https://auth.example.com/v1/scim/v2/con_1
```

| Resource | Operations |
| --- | --- |
| `/Users` | `GET` (list, filter), `POST` |
| `/Users/{id}` | `GET`, `PUT`, `PATCH`, `DELETE` |
| `/Groups` | `GET`, `POST` |
| `/Groups/{id}` | `GET`, `PUT`, `PATCH`, `DELETE` |

Deprovisioning through SCIM (`DELETE /Users/{id}`, or a `PATCH` setting
`active: false`) is what makes offboarding work: the person loses access to
every app that authenticates through IAM at once.

Revoke a token with `DELETE .../scim/tokens/{token_id}`; list them with `GET`.

## Not the same thing as social login

Social login (Google, GitHub for consumers) is a different feature with a
different configuration surface — see [Social login](/guides/oauth-social).
Enterprise SSO is per-customer and domain-routed; social login is per-project and
offered to everyone.
