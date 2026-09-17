# Account security implementation

## Contract

IAM owns detection, incidents, devices, notification delivery, protection and
recovery. Applications render server-owned flows through the existing SDK.
An incident review does not implicitly authenticate a user or trust a device.
Recovery can be escalated to an audited support decision. Reporting an incident
revokes its affected session by default; the user may explicitly select all
sessions and device trust. That choice persists through credential recovery.
All data and actions are scoped by project, environment and account.

## Implementation checklist

- [x] Domain, OpenAPI, persistence and additive migration
- [x] Device recognition independent of sessions and explicit trust
- [x] Risk coverage across authentication methods and sensitive changes
- [x] Atomic protection, scope selection and legacy endpoint enforcement
- [x] Review/recovery flows, proof, expiry, replay and concurrent operations
- [x] Application continuation URLs and restricted capabilities
- [x] Support cases, decisions and one-time recovery grants
- [x] Durable notifications, channel isolation and delivery observability
- [x] Existing TypeScript SDK extension and behavior tests
- [x] Admin interfaces and runnable headless integration example
- [x] Compatibility, policy, retention and current-state documentation
- [x] Integration, browser, generation, lint and build verification

## Acceptance

Exercise both single-session and all-session recovery; an unrelated session
survives the former even after password replacement. No ordinary session is
issued by security recovery. A continuation can only address its incident and
cannot prove ownership by itself. Pre-existing trusted proof or an audited
support grant is needed to recover access. A newly supplied support contact is
not ownership proof. All read-only page loads have no protective side effects.
Support cases survive expired browser flows. Public events never carry secrets.
Detection works without a geographic database; unknown context is explicit.
Local JWT validation retains its documented expiry window; remote validation
must enforce revocation immediately.

## Integration and operation

The current contract, configuration defaults, SDK examples, support permissions,
notification semantics and rollout sequence are documented in
[Account security and recovery](../website/docs/guides/account-security.md).
The policy defaults to disabled; enabling it is a per-project/environment action.

## Validation

- `GOWORK=off make lint`: Go vet and strict lint for changes.
- `GOWORK=off make lint-web`: TypeScript SDK and application type checks.
- `GOWORK=off go test -tags=integration ./... -count=1 -p 1`: complete Go suite,
  including existing authentication compatibility tests and real Postgres.
- `GOWORK=off go test -race -tags=integration ./internal/infrastructure/postgres -run '^TestSecurity' -count=1`:
  flow replay, concurrent protection, proof, scope and support transitions.
- `yarn --cwd ts workspace @gopherex/iam-sdk test`: SDK capability isolation,
  rotation, storage, explicit sign-in retry and session clearing.
- `GOWORK=off make generate` and `GOWORK=off make build`: contract/store generation
  and the server with the embedded application.
- `yarn --cwd website build`: documentation links and production build.

Browser integration tests live in `integration_security_browser_test.go` and
`web/test/security-browser.mjs`. They start a local HTTP server against the test
Postgres, drive the built reference application, and use Chromium's virtual
authenticator for a cryptographically verified passkey ceremony. They exercise
selective/all-session protection, reload of the restricted flow and no automatic
login after password/passkey recovery.

Install Playwright and Chromium outside the repository, then run:

```sh
npm install --prefix /tmp/iam-security-browser playwright
node /tmp/iam-security-browser/node_modules/playwright/cli.js install chromium
IAM_PLAYWRIGHT_MODULE=/tmp/iam-security-browser/node_modules/playwright/index.mjs \
  GOWORK=off go test -tags=integration ./internal/infrastructure/postgres \
  -run '^TestSecurityBrowser$' -count=1
```

Build `web/dist` first. Without `IAM_PLAYWRIGHT_MODULE`, only the browser tests are
skipped; the other integration tests still run. Notification transport is not sent
to external recipients by these tests: the local harness reads queued verification
codes and validates the durable flow behavior.
