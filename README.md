# IAM — Authentication & Identity

Headless authentication & identity server. The HTTP contract is the single
source of truth: [`openapi/openapi.yaml`](openapi/openapi.yaml) (OpenAPI 3.1.0).

## Layout

| Path | Purpose |
| --- | --- |
| [`Makefile`](Makefile) | Developer entry point — `make help` lists targets. |
| [`openapi/`](openapi) | The OpenAPI 3.1 spec (source of truth) + notes. |
| [`.ogen.yaml`](.ogen.yaml) | [ogen](https://ogen.dev) config — Go codegen from the spec. |
| [`pkg/`](pkg) | Public, importable Go: the API implementation (`pkg/api`) and the Go SDK (`pkg/sdk`). |
| [`internal/oas/`](internal/oas) | Module-private generated ogen code (wire types, client, server scaffolding). |
| [`internal/`](internal) | Other module-private packages; not importable from outside. |
| [`cmd/iam/`](cmd/iam) | The Go server — serves the API and the embedded admin SPA. |
| [`ts/`](ts) | Yarn workspace; the TypeScript SDK, published to the GitHub npm registry. |
| [`web/`](web) | Admin panel SPA, served by the server. |
| [`deployments/`](deployments) | Production deployment artifacts. |
| [`docker-compose.yml`](docker-compose.yml) | Local dev environment (full infra). |
| [`docs/rfc/`](docs/rfc) | Reference set of the standards IAM implements. |

## Quickstart

```sh
make help        # list targets
make generate    # regenerate Go + TS from the spec
make dev         # bring up dev infra
make run         # run the server
```

Stacks (HTTP runtime, storage, frontend, TS toolchain) are decided separately;
scaffolding here is intentionally stack-agnostic.
