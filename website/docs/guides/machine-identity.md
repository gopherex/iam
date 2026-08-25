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

A service account is an OAuth client of the client-credentials kind (RFC 6749
§4.4): its id **is** the client id, its secrets are client secrets, and it gets a
token from the standard token endpoint — so every OAuth library already knows how
to talk to it.

```bash
# create it and issue a secret
curl -sX POST .../admin/service-accounts -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"billing-sync","scopes":["users:read","users:write"]}'
# -> { "service_account": { "id": "sa_…" } }

curl -sX POST .../admin/service-accounts/sa_…/secrets -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"name":"prod"}'
# -> { "secret": "<shown once>" }
```

Then, from the service itself:

```bash
curl -sX POST https://auth.example.com/oauth2/token \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d grant_type=client_credentials
# -> { "access_token": "eyJ…", "token_type": "Bearer", "expires_in": 3600,
#      "scope": "users:read users:write" }
```

`client_secret_post` works too — send `client_id` and `client_secret` in the
form body instead of the Authorization header. `client_credentials` is
advertised in `grant_types_supported`.

The token is a one-hour RS256 JWT carrying `typ: service`, the account id as
`sub`, and the account's scopes. There is no refresh token and no id_token:
there is no user to represent, and a client that can authenticate can simply ask
again.

### Narrowing a token

```bash
curl -sX POST https://auth.example.com/oauth2/token -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d grant_type=client_credentials -d scope="users:read"
```

`scope` can only ask for **less** than the account already holds. A scope outside
the grant is refused with `invalid_scope` rather than quietly trimmed, so a
misconfigured caller fails loudly instead of holding a token that does less than
it thinks.

An account may hold several secrets at once, so one can be rotated in before the
other is deleted. Setting `disabled` on the account stops it minting immediately
— existing tokens still run out their hour.

`POST /v1/service-accounts/tokens` is the older path to the same thing; it mints
for the calling service account and ignores the request's `scopes` and
`ttl_seconds`. Prefer `client_credentials`.

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
