---
id: projects-environments
title: Projects & environments
sidebar_label: Projects & environments
---

# Projects & environments

## Projects

A **project** is the top-level tenant. One IAM instance hosts many projects; each
has its own users, app clients, config, signing keys, and connections. Projects
are created and managed by the **operator** through `/mgmt/v1/projects` (see
[Operator guide](/self-hosting/operator)).

Runtime calls never take a project id in the path — they resolve it from the
**`X-Client-Id`** header:

```bash
curl https://auth.example.com/v1/auth/session \
  -H "X-Client-Id: prj_7Fk2" -H "Authorization: Bearer eyJ..."
```

## Environments

Every runtime-data row is keyed on `(project_id, environment)`, giving
**Stripe-like test / live isolation**. A user, session, or flow in `test` is
completely separate from `live` — the same email can exist independently in each.

The environment is chosen per request with the **`X-Environment`** header.
Absent or empty means **`live`**.

```bash
# operate against the test environment
curl https://auth.example.com/v1/projects/prj_7Fk2/admin/users \
  -H "Authorization: Bearer <admin_token>" -H "X-Environment: test"
```

:::info Environments must exist
An environment is validated against the project before use — you cannot conjure
data (or signing keys) under an arbitrary environment name. Manage a project's
environments via `/mgmt/v1/projects/{id}/environments` (operator).
:::

### Environment binding of credentials

An authenticated credential is **bound to the environment its token was minted
in**. Presenting a `live` token with `X-Environment: test` is rejected — a token
cannot cross environments. This closes a whole class of test↔live data-leak bugs.

## What is *not* environment-scoped

- **Token TTLs** and **session policy** live in the per-project
  `session_policy` config doc (which itself is env-scoped), not in server config.
- **Test mode** (`/v1/test/*`: seed / reset / clock / captured messages) is
  available in **non-live environments only** — it can never touch live data.

## Config lifecycle (IaC)

Project configuration (auth methods, policies, providers, connections…) can be
managed imperatively via the admin API, or declaratively via the operator config
lifecycle: `config:export` → `config:plan` → `config:apply` under
`/mgmt/v1/projects/{id}`.
