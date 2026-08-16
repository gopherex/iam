# IAM — Authentication & Identity

A headless, multi-tenant **authentication** server: password + passwordless
sign-in, MFA (TOTP/SMS/email/WebAuthn), OAuth social login, resumable auth
flows, an OIDC provider, SAML/OIDC/SCIM federation, webhooks, and a full admin +
operator API. It ships as a single static binary (admin SPA embedded, Postgres
for storage) plus a first-class TypeScript SDK.

The HTTP contract is the single source of truth:
[`openapi/openapi.yaml`](openapi/openapi.yaml) (OpenAPI 3.1.0). The Go server and
the TypeScript SDK are generated from it.

## 📚 Documentation

**Full docs, guides, and API reference:
[https://gopherex.github.io/iam/](https://gopherex.github.io/iam/)**

Start there — it has real, copy-pasteable examples for everything below:

- [Quickstart](https://gopherex.github.io/iam/quickstart) — sign up a user in
  minutes
- [Core concepts](https://gopherex.github.io/iam/concepts/overview) — projects,
  environments, principals, sessions, MFA, webhooks, OIDC/federation
- [Guides](https://gopherex.github.io/iam/guides/sdk-quickstart) — SDK, auth
  flows, passwordless, OAuth, MFA, project config
- [REST API reference](https://gopherex.github.io/iam/rest-api/overview)
- [TypeScript SDK reference](https://gopherex.github.io/iam/sdk/typescript)
- [Self-hosting](https://gopherex.github.io/iam/self-hosting/docker) — Docker,
  configuration, operator, deployment, observability, migrations

## Run it locally

```sh
docker compose up
```

Brings up Postgres 17 + the IAM server (admin SPA embedded, migrations applied,
a `Root` project seeded). Open <http://localhost:8080> and sign in with the
master key `dev`.

> ⚠️ The compose defaults (master key `dev`, a committed AES key) are for
> localhost only. For anything shared, set a real
> `IAM_SERVICE_AUTH_ENCRYPTION_KEY` (`openssl rand -base64 32`) and
> `IAM_SERVICE_AUTH_MASTER_KEY`. See
> [Configuration](https://gopherex.github.io/iam/self-hosting/configuration).

## Use the SDK

```ts
import { createIamClient } from '@gopherex/iam-sdk';

const iam = createIamClient({ baseUrl: 'https://auth.example.com', clientId: 'app_web' });

await iam.flow.start({ kind: 'signup', email, password, name });
await iam.flow.verifyEmail({ code }); // → SIGNED_IN on completion
```

`@gopherex/iam-sdk` is published to the GitHub Packages registry. See the
[SDK quickstart](https://gopherex.github.io/iam/guides/sdk-quickstart).

## Repository layout

| Path | Purpose |
| --- | --- |
| [`Makefile`](Makefile) | Developer entry point — `make help` lists targets. |
| [`openapi/`](openapi) | The OpenAPI 3.1 spec (source of truth) + notes. |
| [`.ogen.yaml`](.ogen.yaml) | [ogen](https://ogen.dev) config — Go codegen from the spec. |
| [`pkg/`](pkg) | Public, importable Go: the API implementation (`pkg/api`) and the Go SDK (`pkg/sdk`). |
| [`internal/oas/`](internal/oas) | Module-private generated ogen code (wire types, client, server scaffolding). |
| [`internal/infrastructure/postgres/`](internal/infrastructure/postgres) | SQL store: pgx + pgtx + bob + the sqld codegen toolchain. |
| [`internal/`](internal) | Other module-private packages; not importable from outside. |
| [`cmd/iam/`](cmd/iam) | The Go server — serves the API and the embedded admin SPA. |
| [`ts/`](ts) | Yarn workspace; the TypeScript SDK, published to the GitHub npm registry. |
| [`web/`](web) | Admin panel SPA, served by the server. |
| [`website/`](website) | The documentation site (Docusaurus) → GitHub Pages. |
| [`deployments/`](deployments) | Production deployment artifacts (Dockerfile). |
| [`docker-compose.yml`](docker-compose.yml) | Local dev environment (full infra). |
| [`docs/rfc/`](docs/rfc) | Reference set of the standards IAM implements. |

## Development

```sh
make help        # list targets
make generate    # regenerate Go + TS from the spec
make dev         # bring up dev infra (docker compose)
make run         # run the server
make test        # unit tests
make test-db     # integration tests (testcontainers; needs Docker)
```

## License

See [LICENSE](LICENSE).
