---
id: machine-identity
title: Machine identity
sidebar_label: Machine identity
---

# Machine identity

Three ways a non-human caller authenticates against IAM, for three different
jobs:

| Credential | Use it for | Surface |
| --- | --- | --- |
| **API key** | a backend or CI job calling the runtime API | `/v1/*` |
| **Project-admin token** | configuring a project | `/v1/projects/{id}/admin/*` |
| **OAuth client** | an application in an OIDC flow | `/oauth2/*` |

A project-admin token is minted by the operator — see
[Operator](/self-hosting/operator). OAuth clients are covered in
[OIDC provider](/concepts/oidc-federation). This page is about the first one.

## API keys

An API key is an opaque secret, shown once, presented as a bearer token.

```bash
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/api-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"ci-runner","scopes":["users:read"],"expires_at":"2027-01-01T00:00:00Z"}'
# -> { "api_key": { "id": "key_3", "prefix": "iak_7Fk2", ... }, "secret": "<shown once>" }
```

Use it directly:

```bash
curl -s https://auth.example.com/v1/users/me \
  -H "Authorization: Bearer $API_KEY_SECRET" -H "X-Client-Id: prj_7Fk2"
```

Only the sha256 of the key is stored; the `prefix` is what you show in a UI or
match in a log, since the secret itself is unrecoverable.

| Operation | Endpoint |
| --- | --- |
| List / create | `GET/POST admin/api-keys` |
| Disable or rename | `PATCH admin/api-keys/{key_id}` (`disabled`, `name`) |
| Rotate | `POST admin/api-keys/{key_id}/rotate` — returns a new secret, invalidates the old one |
| Delete | `DELETE admin/api-keys/{key_id}` |

A key is refused once `disabled` is set or `expires_at` has passed. Rotation is
the safe way to replace one: issue the new secret, deploy it, then delete the
old key.

## Service accounts

A service account is a named machine principal that can hold scopes and mint a
short-lived JWT:

```bash
curl -sX POST .../admin/service-accounts -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"billing-sync","scopes":["users:read","users:write"]}'
```

`POST /v1/service-accounts/tokens` mints an access token for the **calling**
service account: a one-hour RS256 JWT carrying `typ: service`, the account id as
`sub`, and the account's scopes. The `scopes` and `ttl_seconds` in the request
body are not yet honoured — the account's own scopes and a fixed one-hour
lifetime are used.

:::caution A service-account secret is not yet a credential
`POST admin/service-accounts/{sa_id}/secrets` issues and stores a secret, but no
authentication path verifies it — there is no client-credentials grant for
service accounts today, and the token-minting endpoint requires a service token
you already hold. Until that lands, use an **API key** for machine access.
:::

## Scopes

Scopes are plain strings carried in the credential and checked by whatever reads
it. The [Go SDK](/sdk/go) exposes them as `principal.HasScope("users:read")`.
They are not a substitute for authorization: IAM does authentication, and a
permissions model belongs in your own authorization layer.

## Verifying these tokens in your services

A resource server behind IAM verifies whatever bearer it is handed — user token,
service token or API key — with the [Go SDK](/sdk/go) or by calling
`POST /v1/tokens/verify`. Local JWT verification does not see a revocation until
the token expires; the SDK page has the table.
