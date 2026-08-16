---
id: authentication
title: Authenticating requests
sidebar_label: Authentication
---

# Authenticating requests

## Headers

| Header | Required | Purpose |
| --- | --- | --- |
| `X-Client-Id` | yes (runtime calls) | Resolves the **project** + app client. |
| `X-Environment` | optional | Selects the environment; default `live`. |
| `Authorization: Bearer <token>` | yes (secured calls) | User token, service/API key, admin token, master key, or SCIM token — depends on the namespace. |
| `X-CSRF-Token` | cookie-mode state changes | CSRF token from `GET /v1/csrf`. |
| `X-Device-Name` / `X-Device-Fingerprint` | optional (sign-in) | Persisted with the new session; fingerprint aids refresh-theft detection. |
| `Idempotency-Key` | optional | Safe retry of create POSTs. |

## Security schemes

All credentials are HTTP bearer tokens; the *kind* determines the surface:

| Scheme | Principal | Used on |
| --- | --- | --- |
| `bearerAuth` | end-user access token (default) | `/v1/*` runtime |
| `serviceToken` | service account | `/v1/*` machine-to-machine |
| `adminToken` | project admin | `/v1/projects/{id}/admin/*` |
| `masterKey` | operator | `/mgmt/v1/*` |
| `scimToken` | SCIM connection | `/v1/scim/v2/{connection_id}/*` |
| `clientSecretBasic` / `oauth2` | OAuth client | `/oauth2/*` |

## Two request modes

### Bearer mode (default; APIs, mobile, SPAs)

Send the access token in the `Authorization` header. Immune to CSRF because the
credential is not ambiently attached.

```bash
curl https://auth.example.com/v1/auth/session \
  -H "X-Client-Id: prj_7Fk2" \
  -H "X-Environment: live" \
  -H "Authorization: Bearer eyJ..."
```

### Cookie mode (browsers)

The session can instead live in the `iam_session` cookie (`iam_refresh` scoped to
the refresh endpoint). The server promotes the cookie into a bearer credential
internally. **Every state-changing cookie request must carry a valid
`X-CSRF-Token`** (bound to the same `X-Client-Id`) or it is rejected `403
invalid_csrf`. Safe methods, bearer requests, and non-cookie requests are exempt.

```bash
# 1. get a CSRF token
curl https://auth.example.com/v1/csrf -H "X-Client-Id: app_web"
# -> { "csrf_token": "csrf_..." }

# 2. state-changing cookie request
curl -X POST https://auth.example.com/v1/auth/sign-out \
  -H "X-Client-Id: app_web" \
  -H "X-CSRF-Token: csrf_..." \
  --cookie "iam_session=..."
```

## Environment binding

An authenticated credential is bound to the environment it was minted in. A
`live` token presented with `X-Environment: test` is rejected — tokens cannot
cross environments.
