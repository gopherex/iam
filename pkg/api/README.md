# pkg/api — IAM API implementation

Hand-written, importable. This is **our** code: the handler implementation that
satisfies the generated server interface and the public façade callers import.

- `Handler` — the server interface (re-exported from `internal/oas`), so
  importers depend only on `pkg/api`, never on internal packages.
- `Service` — implements `Handler`; methods added as the runtime stack is wired.

The generated ogen code (wire types, client, server scaffolding, validators,
fakers) is **module-private** under [`../../internal/oas`](../../internal/oas)
and regenerated with `make generate-go`. Do not edit generated code; put logic
here.

## Dependencies & transaction boundary

Each `XxxService` is **pure orchestration**: it holds aggregate-port interfaces
in an `XxxDeps` (constructed via `NewXxxService(deps)`) and nothing else. A
service method maps `oas → domain`, calls one or more aggregate-port methods,
maps `domain → oas`. **Services never open a transaction.**

- **Ports are consumer-defined, next to the service** (`coreAuthAccounts`,
  `federationConnections`, …) — each declares only the slice of an aggregate it
  uses (interface segregation).
- **The aggregate is the transaction boundary.** Each port method is one atomic
  business operation; the persistence adapter (`internal/infrastructure/postgres`)
  opens the `pgtx` transaction **inside** the method. So a custom implementation
  injected via `WithCoreAuth(...)` owns its own persistence and never inherits
  `pgtx` from the service contract.
- Domain types live in [`../../internal/domain`](../../internal/domain); `oas`
  types stay at the wire edge.
- Cross-aggregate consistency (events/outbox) is out of scope here — one call
  mutates one aggregate.
