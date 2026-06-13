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

## Errors & validation

Two layers, both rendering the shared `ErrorEnvelope` (`{ error: { code, message } }`):

- **Handler errors → `Service.NewError`** (ogen *convenient errors*). Every
  operation shares one `default` error response, so ogen generates
  `NewError(ctx, err) *oas.DefaultStatusCode`. Handlers just `return …, err`;
  `NewError` maps a `domain.Error` (stable `code` + HTTP status, see
  [`internal/domain/errors.go`](../../internal/domain/errors.go)) into the
  envelope; anything else becomes `500 internal_error`.
- **Generated-server errors → `ErrorHandler`**. ogen runs **code-generated
  schema validation** on decode (the spec's `minLength`/`maxLength`/`pattern`/
  `format`/`required`/`enum` constraints). Validation, parameter/body decode
  and security failures are raised *before* the handler, so they bypass
  `NewError`; response validation/encoding failures happen *after* the handler
  and also route through this hook. Wire `oas.WithErrorHandler(api.ErrorHandler)`
  to render them into the same envelope (`validation_failed` / `unauthorized` /
  `internal_error` with `details.stage=response_encode`).

Add constraints to the OpenAPI schemas to get more validation for free — no
handler code.

## Authentication (security layer)

ogen calls a `SecurityHandler` to authenticate before the handler runs.
`api.NewSecurityHandler(auth)` adapts an **`Authenticator`** port (one method per
scheme: `User`/`Admin`/`Master`/`Service`/`SCIM`/`Client`/`OAuth2`) — the adapter
verifies the credential and returns a `*domain.Principal`, which the handler
stores in the request context. Authenticated handlers read it via
`requirePrincipal(ctx)` (→ `domain.ErrUnauthorized` if absent) instead of
re-parsing tokens. A failed `Authenticator` call surfaces as a 401 through the
`ErrorHandler`.

Wire it all together:

```go
srv, _ := oas.NewServer(
    api.New(api.WithCoreAuth(coreAuth), /* … */),
    api.NewSecurityHandler(auth),
    oas.WithErrorHandler(api.ErrorHandler),
)
```

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
