---
id: admin
title: Admin & operator API
sidebar_label: Admin & operator
---

# Admin & operator API

Two management planes: **project admin** (`adminToken`) and **operator**
(`masterKey`). See [Operator guide](/self-hosting/operator) for bootstrapping.

## Project-admin — `/v1/projects/{project_id}/admin/*`

Secured with `adminToken`; accepts `X-Environment`.

| Area | Endpoints |
| --- | --- |
| **Users** | `GET/POST /users`, `GET/PATCH/DELETE /users/{id}`, `/ban`, `/unban`, `/password`, `/verify-email`, `/verify-phone`, `/mfa/reset`, `/sessions[/revoke\|/{id}]`, `/identities[/{id}]`, `/grants[/{id}]`, `/impersonate`, `/anonymize`, `/export` |
| **App clients** | `GET/POST /apps`, `GET/PATCH/DELETE /apps/{id}`, `/apps/{id}/secrets[/{id}]` |
| **Service accounts** | CRUD + `/secrets` (runtime minting: `POST /v1/service-accounts/tokens`) |
| **API keys** | CRUD + `/rotate` |
| **Connections (SSO)** | SAML/OIDC connection CRUD, `/test`, `/rotate-certificate`, `/scim/tokens`; verified `domains` |
| **Config** | `GET/PUT config/{auth, password-policy, session-policy, mfa-policy, rate-limits}`, `features`, `consents`, `retention-policy`, `i18n/{locale}` |
| **Signing keys** | `jwks`, `jwks/rotate`, `jwks/{key_id}/activate`, `DELETE jwks/{key_id}`; `token-profiles[/{id}][/preview]` |
| **Invites** | `GET/POST /invites`, `POST /invites/{id}/revoke` (token shown once) |
| **Webhooks & hooks** | `webhooks[/{id}]`, `/rotate-secret`, `/test`; `webhook-deliveries[/{id}/retry]`; `events[/{id}/replay]`; `hooks[/{id}][/test]` |
| **Audit / jobs / import / export** | `audit-logs[/{id}]`, `audit/export`; `jobs[/{id}][/cancel]`; `import/users`, `import/password-hashes/verify`; `exports/{job_id}` |
| **Risk & rate limits** | `risk/events`, `risk/rules[/{id}]`, `rate-limit/blocks[/{id}]` |
| **Providers** | `oauth-providers[/{id}]`, `email-providers[/{id}]`, `sms-providers[/{id}][/send-test]`, `email-templates[/{id}][/preview\|/send-test]` |
| **Access requests** | `access-requests[/{id}/approve\|/deny]` |

### Examples

```bash
# List users in the test environment (keyset paginated)
curl "https://auth.example.com/v1/projects/prj_7Fk2/admin/users?limit=50" \
  -H "Authorization: Bearer <admin_token>" -H "X-Environment: test"

# Set the session policy
curl -sX PUT https://auth.example.com/v1/projects/prj_7Fk2/admin/config/session-policy \
  -H "Authorization: Bearer <admin_token>" -H "Content-Type: application/json" \
  -d '{"access_ttl":900,"refresh_ttl":2592000,"idle_timeout":0,"reuse_detection":true}'

# Import users (async job; supply pre-hashed bcrypt passwords)
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/import/users \
  -H "Authorization: Bearer <admin_token>" -H "Content-Type: application/json" \
  -d '{"users":[{"email":"ada@example.com","password_hash":"$2b$12$..."}],"password_hash_format":"bcrypt"}'
# -> { "job_id": "job_...", "status": "running" }  ; poll GET .../admin/jobs/{job_id}
```

## Operator — `/mgmt/v1/*`

Secured with `masterKey`.

| Method | Path |
| --- | --- |
| `GET/POST` | `/mgmt/v1/projects` |
| `GET/PATCH/DELETE` | `/mgmt/v1/projects/{id}` |
| `GET/PUT` | `/mgmt/v1/projects/{id}/features`, `/environments[/{env}]` |
| `GET/POST/DELETE` | `/mgmt/v1/projects/{id}/admin-tokens[/{id}]` |
| `POST` | `/mgmt/v1/projects/{id}/config:export \| config:plan \| config:apply` |

```bash
curl -sX POST https://auth.example.com/mgmt/v1/projects \
  -H "Authorization: Bearer $MASTER_KEY" -H "Content-Type: application/json" \
  -d '{"name":"Acme","slug":"acme"}'
# -> 201 { "project": { "id": "prj_...", ... } }
```

## Test mode — `/v1/test/*`

Secured with `adminToken`; **non-live environments only** (a live `X-Environment`
is refused). For deterministic tests:

| Method | Path |
| --- | --- |
| `GET` | `/v1/test/messages?channel&to` — captured email/SMS inbox |
| `POST` | `/v1/test/clock` — `{advance_seconds?, reset?}` |
| `POST` | `/v1/test/reset` — wipe the environment's data |
| `POST` | `/v1/test/seed` — create fixtures |
