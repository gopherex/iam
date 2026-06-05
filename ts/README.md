# ts/ — TypeScript workspace

Yarn workspace holding TypeScript packages built around the IAM API.

- [`packages/sdk`](packages/sdk) — `@gopherex/iam-sdk`, the published client.
  Generated from [`../openapi/openapi.yaml`](../openapi/openapi.yaml) and
  released to the **GitHub npm registry** (`https://npm.pkg.github.com`).

Generation tool and TS stack are decided later. `yarn generate` regenerates the
SDK from the spec; `yarn build` builds all workspace packages.
