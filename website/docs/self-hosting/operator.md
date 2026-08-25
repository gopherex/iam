---
id: operator
title: Operator & bootstrapping
sidebar_label: Operator
---

# Operator & bootstrapping

The **operator** plane (`/mgmt/v1/*`, secured by the `masterKey`) is how you
create projects, environments, and the project-admin tokens that everything else
uses. It is the top of the [principal hierarchy](/concepts/principals).

## The master key

Set `IAM_SERVICE_AUTH_MASTER_KEY` to a long random secret. It authenticates:

- the embedded **admin SPA** (open the server root and sign in with it), and
- the **operator API** (`Authorization: Bearer $MASTER_KEY`).

An empty master key disables the operator scheme entirely.

```bash
openssl rand -hex 32   # a good master key
```

## Bootstrap a project

```bash
export MASTER_KEY=...   # your IAM_SERVICE_AUTH_MASTER_KEY

# 1. create a project
curl -sX POST https://auth.example.com/mgmt/v1/projects \
  -H "Authorization: Bearer $MASTER_KEY" -H "Content-Type: application/json" \
  -d '{"name":"Acme","slug":"acme"}'
# -> 201 { "project": { "id": "prj_7Fk2", ... } }

# 2. mint a project-admin token (used for /v1/projects/{id}/admin/*)
curl -sX POST https://auth.example.com/mgmt/v1/projects/prj_7Fk2/admin-tokens \
  -H "Authorization: Bearer $MASTER_KEY" -H "Content-Type: application/json" \
  -d '{"name":"ci"}'
# -> { "admin_token": "<shown once>", "expires_at": ... }
```

The `Root` project can be auto-seeded in dev with `IAM_SERVICE_AUTH_SEED_ROOT=true`.

## Operator endpoints

| Method | Path |
| --- | --- |
| `GET/POST` | `/mgmt/v1/projects` |
| `GET/PATCH/DELETE` | `/mgmt/v1/projects/{id}` |
| `GET/PUT` | `/mgmt/v1/projects/{id}/features`, `/environments[/{env}]` |
| `GET/POST/DELETE` | `/mgmt/v1/projects/{id}/admin-tokens[/{id}]` |
| `POST` | `/mgmt/v1/projects/{id}/config:export \| config:plan \| config:apply` |

## Environments

Each project has isolated environments (`live`, `test`, `staging`). Create and
tune them via `/mgmt/v1/projects/{id}/environments`. Every runtime and admin call
targets one through the `X-Environment` header. See
[Projects & environments](/concepts/projects-environments).

## Config as code

Export a whole project's config, edit it, plan the diff, and apply:

```bash
curl -sX POST .../mgmt/v1/projects/prj_7Fk2/config:export -H "Authorization: Bearer $MASTER_KEY" > project.json
# edit project.json ...
curl -sX POST .../mgmt/v1/projects/prj_7Fk2/config:plan  -H "Authorization: Bearer $MASTER_KEY" -d @project.json
curl -sX POST .../mgmt/v1/projects/prj_7Fk2/config:apply -H "Authorization: Bearer $MASTER_KEY" -d @project.json
```

Once you have an `adminToken`, configure the project itself via the
[admin API](/guides/admin-config).
