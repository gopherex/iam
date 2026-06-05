# internal/infrastructure/postgres — SQL store

Postgres persistence, replicating the stroppy-cloud / komeet pipeline:
**pgx v5 + pgtx (tx manager) + bob (query builder) + the sqld codegen
toolchain**. No ORM. Each record type is one table with envelope columns
(`id`, `project_id`, `created_at`, `updated_at`, secondary keys) plus the domain
object in a `data jsonb` column.

## Layout

| Path | Role |
| --- | --- |
| `schema.sql` | Authoritative schema — sqld reads it to generate code + the bootstrap migration. |
| `queries/*.sql` | Named queries (`-- name: X :exec\|:one\|:many\|:execrows`, `@param`). |
| `migrations/` | `go:embed`-ed SQL migrations + `migrations.go`. |
| `gen/db/` | sqld-gen-go output: `models.go`, `queries.go` (typed query funcs). |
| `gen/bob/models/` | sqld-gen-bob output: bob query builders per table. |
| `sqld.yaml` | Codegen + migration config. |
| `db.go` | Connection bundle: pool + tx manager + ctx-aware `TxDB` + bob pool. |
| `migrate.go` | `Migrate()` applies the embedded migrations at startup. |
| `bobexec.go` | Adapts the tx-aware executor to `bob.Executor`. |
| `helpers.go` | JSON marshal/unmarshal, pg error → `ErrNotFound`/`ErrConflict`, uuid. |
| `store.go` | `Store` façade; typed repos added per domain entity. |

Generated code (`gen/`) and migrations are committed. `bin/` (the sqld tools)
is git-ignored.

## Workflow

```sh
make tools             # install sqld, sqld-gen-go, sqld-gen-bob into ./bin
make db-gen            # regenerate gen/db + gen/bob from schema.sql + queries
make migrate-generate name=add_x   # incremental migration by schema diff
make migrate-clear     # rebuild the single bootstrap migration, then db-gen
make test-db           # DB-backed tests (auto-starts the compose postgres :5436)
```

## Transactions

Services open a transaction with `tx.Do*` (pgtx); repo calls inside it issue SQL
through `db.q()` / `db.Bobx()`, both bound to the ctx-aware `TxDB`, so they run
in that transaction. Outside one they auto-commit on the pool.

> The two seed tables (`iam_users`, `iam_sessions`) are representative; real IAM
> tables and repos are added as the domain is designed.
