# OpenAPI — single source of truth

`openapi.yaml` is the **authoritative HTTP contract** for the IAM API
(OpenAPI 3.1.0). All endpoints, request/response schemas, error codes, security
schemes, and parameters are defined here.

- Validate: `python -m openapi_spec_validator openapi.yaml` (or any 3.1 validator).
- Human-readable API docs are generated from this spec when a docs build is set
  up (e.g. `docusaurus-plugin-openapi-docs` or `redocly`). There is no
  hand-written narrative reference; this file is the only authoritative source.

## Tag groups & feature modules

Tags are organized two ways:

- **`x-tagGroups`** — eight logical macro-sections for doc rendering
  (Redoc/Scalar): Platform, Authentication, Account, Machine Identity,
  Federation, OIDC Provider, Administration, Operator.
- **`x-features`** — capability **modules**: cohesive route bundles that ship
  and enable as a unit. Each tag carries `x-feature` (its module) and, where a
  route bundle is meaningless on its own, `x-depends-on` (the tags it requires).
  `x-features.<module>.requires` lists cross-module dependencies.

Hard couplings encoded:

| Module | Tags | Requires |
| --- | --- | --- |
| `core-auth` | Core Auth, Verification, Password, Tokens | — |
| `mfa` | MFA | `core-auth` |
| `account` | Users, Identities, Sessions | — (Identities/Sessions need a user) |
| `federation` | Enterprise SSO, SCIM, Domains | — (SCIM/Domains need an SSO connection) |
| `oidc-provider` | OIDC Provider, OIDC Interaction, Device Flow | — (Interaction/Device only drive the provider) |
| `machine-identity` | Service Accounts, API Keys | — |
| `admin` | Admin Users/Apps/Config/Webhooks/Keys/Jobs/Risk, Test Mode | — |
| `passwordless`, `oauth-social`, `webauthn`, `platform`, `operator` | (one tag each) | — |

`x-depends-on` is intra-module (e.g. `SCIM` → `Enterprise SSO`); `requires` is
inter-module (e.g. `mfa` → `core-auth`).

## Filter planes

The API is sliceable along **9 orthogonal axes**, each queryable from a spec
field with **no URL heuristics** (`x-filter-planes` enumerates them):

| # | Plane | Kind | Source field | Values |
| --- | --- | --- | --- | --- |
| 1 | tag | taxonomy | `operation.tags` | 29 |
| 2 | tag-group | taxonomy | `x-tagGroups` (tag→group) | 8 |
| 3 | feature | taxonomy | `tag.x-feature` / `x-features` | 12 |
| 4 | audience | access | `operation.x-audience` | 5 (`runtime`, `project-admin`, `operator`, `scim`, `oidc-provider`) |
| 5 | credential | access | `operation.security` (`[]` = public) | 8 |
| 6 | method | operation | HTTP verb | 5 |
| 7 | project | scope | `operation.x-scopes` ∋ `project` | 275 ops |
| 8 | environment | scope | `operation.x-scopes` ∋ `environment` | 260 ops |
| 9 | content-type | protocol | request/response media-type | 5 |

`x-audience` is the caller plane made explicit (no prefix parsing).
`x-scopes` lists the **data-partition keys** an op is bound to —
`[project, environment]` for runtime/admin/scim/oidc, `[project]` for operator
calls on a specific project, `[]` for global health probes. Project-admin ops
additionally accept the `X-Environment` header to pick the environment.

## Security schemes

| Scheme | Credential | Used by |
| --- | --- | --- |
| `bearerAuth` | user access token | runtime user endpoints (default) |
| `serviceToken` | service-account token / API key | server-to-server |
| `adminToken` | project-admin token | `/v1/projects/{id}/admin/...` |
| `masterKey` | operator master key | `/mgmt/...` |
| `scimToken` | per-connection SCIM token | `/v1/scim/v2/...` |
| `clientSecretBasic` | `client_secret_basic` (HTTP Basic) | `/oauth2/token`, `/revoke`, `/introspect`, `/par`, device auth |
| `oauth2` | OAuth flows (authorizationCode, clientCredentials) | `/oauth2/userinfo` (alt: `bearerAuth`) |

`X-Client-Id` (resolves project + environment on runtime calls) and
`X-CSRF-Token` (cookie-mode state-changing requests) are modeled as request
**parameters/headers**, not security schemes.

## Notes / known trade-offs

- Config, provider, consent, retention, risk-rule, and the auth request bodies
  are **typed schemas**. Remaining `additionalProperties: true` are intentional:
  SCIM resource extensions, standard OIDC payloads (discovery/jwks/userinfo/
  token/introspect), per-vendor provider `config` sub-objects, and freeform
  `metadata`.
- Resource-create POSTs accept an optional `Idempotency-Key` header and return
  `201` (auth-flow POSTs that mint a session — sign-up, refresh, MFA — return
  `200`, since they are actions, not resource creation).
- Cursor-paginated list responses compose `data[]` with the shared `PageMeta`
  schema. Small bounded per-user collections (`sessions`, `identities`,
  `mfa/factors`, `webauthn/credentials`) return `data[]` without a cursor.
