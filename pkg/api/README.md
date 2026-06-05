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
