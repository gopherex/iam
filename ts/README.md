# ts/ — TypeScript workspace

Yarn workspace holding TypeScript packages built around the IAM API.

- [`packages/sdk`](packages/sdk) — `@gopherex/iam-sdk`, the published client.
  Generated from [`../openapi/openapi.yaml`](../openapi/openapi.yaml) with
  [`@hey-api/openapi-ts`](https://heyapi.dev) (typed fetch client + types) and
  released to the **GitHub npm registry** (`https://npm.pkg.github.com`).

Workflow (run once `yarn install`):

- `yarn generate` — regenerate the SDK source (`packages/sdk/src/*.gen.ts`) from
  the spec (also `make generate-ts`). The generated source and `dist/` are not
  committed; they are reproduced from the spec.
- `yarn build` — compile each workspace package to `dist/` (tsc).

The generated SDK exposes a typed function per operation plus the request/response
types; configure the base URL and auth via the `@hey-api/client-fetch` client.
