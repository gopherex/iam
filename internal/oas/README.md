# internal/oas — generated (ogen)

Module-private ogen output, generated from `openapi/openapi.yaml` via
`make generate-go`. **Do not edit** and **do not import from outside this
module** — Go's `internal/` rule forbids it, and the public surface is
re-exported by [`pkg/api`](../../pkg/api).

All ogen features are enabled (see [`../../.ogen.yaml`](../../.ogen.yaml)):
client + server, request/response validation, request options & editors,
reentrant security, OpenTelemetry, fakers and example tests. All 280 operations
are generated (union responses use `oneOf` + a `result_type` discriminator; the
PAR form body is typed).
