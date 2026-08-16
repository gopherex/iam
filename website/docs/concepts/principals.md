---
id: principals
title: Principals & credentials
sidebar_label: Principals & credentials
---

# Principals & credentials

Every secured call authenticates as a **principal**. The principal's *kind*
decides which API surface it may touch. All credentials are HTTP bearer tokens
(`Authorization: Bearer <token>`).

| Principal | Credential | Surface | How you get it |
| --- | --- | --- | --- |
| **Operator** | `masterKey` | `/mgmt/v1/*` | Set at deploy time (`IAM_SERVICE_AUTH_ENCRYPTION_KEY` sibling `master_key`). |
| **Project admin** | `adminToken` | `/v1/projects/{id}/admin/*` | Minted by the operator: `POST /mgmt/v1/projects/{id}/admin-tokens`. |
| **End user** | user access token (bearer or `iam_session` cookie) | `/v1/*` runtime | From a sign-in / flow completion. |
| **Service account** | `serviceToken` (JWT) | `/v1/*` runtime, machine-to-machine | `POST /v1/service-accounts/tokens`; accounts managed under admin. |
| **API key** | opaque `iak_*` key | `/v1/*` runtime | Created under `/v1/projects/{id}/admin/api-keys`. |
| **SCIM** | `scimToken` | `/v1/scim/v2/{connection_id}/*` | Per-connection token under admin `connections`. |
| **OAuth client** | `clientSecretBasic` / `oauth2` | `/oauth2/*` provider | App-client credentials. |

## The master key

The operator master key is the root credential. **If it is unset, the operator
API is fully disabled** — every `/mgmt/v1/*` request is rejected. Set it only at
deploy time via `IAM_SERVICE_AUTH_MASTER_KEY`; treat it like a root password.

## Admin tokens

Project-admin tokens are minted by the operator, optionally scoped and
time-bounded:

```bash
curl -sX POST https://auth.example.com/mgmt/v1/projects/prj_7Fk2/admin-tokens \
  -H "Authorization: Bearer $MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"ci-admin","scopes":["project"],"expires_at":"2027-01-01T00:00:00Z"}'
# -> {"admin_token":"<shown once>","expires_at":"..."}
```

The token is **shown once** — store it in your secret manager.

## End-user tokens

A user access token is a signed **RS256 JWT** carrying:

```
iss, sub, sid, jti, pid, env, aud, aal, amr, typ=access
```

- `sub` — the user id (stable across sessions).
- `sid` — the session / device id.
- `aal` — authenticator assurance level (1 or 2).
- `amr` — the methods used to authenticate.
- `pid` / `env` — project and environment.

`sub` and `sid` are the stable keys a downstream product joins on. See
[Sessions & tokens](/concepts/sessions).

## Browser cookie mode

Browsers may use cookies instead of a bearer header: the session lives in the
`iam_session` cookie (refresh in `iam_refresh`, scoped to the refresh endpoint).
State-changing cookie requests require a CSRF token — see
[REST API → Authentication](/rest-api/authentication).
