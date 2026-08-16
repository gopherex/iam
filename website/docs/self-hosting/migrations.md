---
id: migrations
title: Database & migrations
sidebar_label: Migrations
---

# Database & migrations

IAM stores everything in Postgres (17 in the dev stack). Migrations are
**embedded in the binary and applied automatically on startup** — there is no
separate migration command or job to run in production.

## In production

Nothing to do. When a replica boots, it applies any pending migrations before
its readiness probe goes green, so a rolling update is safe: new replicas gate
themselves on migration completion while old ones keep serving. Data model
changes are additive-first, so old and new server versions coexist during a
roll.

Just make sure the database user in `IAM_INFRA_POSTGRES_*` can create/alter
tables.

## The data model

The schema is generated and managed with **sqld** (the `gopherex/sqld`
toolchain). Application storage uses an envelope + JSONB pattern under
`internal/infrastructure/postgres`, with typed access generated from
`schema.sql` and `queries/*.sql`.

## Working on the schema (contributors)

You only need this when changing the schema — not to deploy.

```bash
# install the sqld generators into ./bin
make tools

# generate a migration from a schema diff
make migrate-generate name=add_widgets

# regenerate gen/db + gen/bob from schema.sql + queries/*.sql
make db-gen

# regenerate everything (API + DB store)
make generate
```

`make migrate-clear` collapses migrations back to a single bootstrap migration
derived from `schema.sql`, then regenerates code — useful pre-release to keep the
migration set tidy.

## Local database

The dev stack runs Postgres 17 on host port **5436** (container `5432`):

```bash
docker compose up   # brings up Postgres + IAM, applies migrations, seeds Root
```

Integration tests spin up a throwaway Postgres via testcontainers (needs
Docker):

```bash
make test-db
```

## Backups

Standard Postgres backup practices apply (`pg_dump` / continuous archiving).
Because reversible secrets are encrypted at rest with
`IAM_SERVICE_AUTH_ENCRYPTION_KEY`, **a backup is useless without that key** —
store the key alongside your backup policy (in a separate secret store), and
never rotate it without a re-encryption plan.
