# Design: Multi-environment isolation, registration enforcement, invites, password strategy

Status: proposed (2026-06-10). Covers three coupled gaps surfaced from the admin UI:

1. **A2 — full environment isolation** (Stripe-like test/live/staging separation).
2. **B — registration mode enforced end-to-end**, including an invite system (email link + manual token).
3. **C2 — password ordering as a registration config option**.

All three are server-side (IAM). The admin UI already renders the needed states
(`blocked`, `request_access`, `set_password`); they were never wired on the server.

---

## Current state (verified)

- Public/auth/flow endpoints take the project from `X-Client-Id`; **no environment** is
  sent or resolved. Runtime is hardcoded to `live` (`authDefaultEnv`, captcha/public-config
  read `env=live`).
- `X-Environment` exists only on `/admin/*` — a config-authoring dimension. Config saved
  under a non-live env is inert at runtime (footgun).
- Signing keys, token `env` claim, issuer-per-env, and the `Environment` domain type already
  anticipate per-env; **data tables (users/sessions/challenges/flows/identities/grants/mfa)
  are project-scoped only** — no environment column.
- The flow engine never reads the `auth` config doc → `registration.mode` is not enforced;
  public config exposes `methods` but not `registration`.
- Signup flow (`advanceSignupCreate`) registers with a password immediately, then verifies
  email. Step order is fixed in code. `set_password` step already exists.

---

## A2 — Full environment isolation

**Model:** `environment` becomes a first-class scope on every runtime-data row, alongside
`project_id`. A user/session/flow in `test` is fully separate from `live`. Default for all
existing data: `live`.

### Schema migration
Add `environment text NOT NULL DEFAULT 'live'` and rewrite uniqueness to include it:
- `iam_users` — unique email/phone now per `(project_id, environment)` (same email may exist
  in test and live).
- `iam_sessions`, `iam_challenges`, `iam_flows`, `iam_identities`, `iam_grants`,
  `iam_mfa_*` / recovery / backup codes, `iam_invites` (new).
- Management tables stay project-level (app_clients, signing_keys already env-tagged,
  providers, configs already per-env, admin_tokens, api_keys). API keys / service accounts:
  keep project-level but stamp env on minted tokens.
- Backfill: existing rows → `live`. Indexes recreated with `environment` prefix where they
  scope tenancy.

### Request resolution
- Public ops gain an optional `X-Environment` header (default `live`), mirroring admin.
- SDK `createIamClient({ environment })` sends `X-Environment` on every call; `headers()`
  returns `{ 'X-Client-Id', 'X-Environment' }`.
- Every public/flow handler threads `(ProjectID, Environment)` into its cmd.
- Authenticator: mint stamps the request env into the token `env` claim and the session row;
  verify already reads `env` for key selection — also assert the principal's env matches the
  resource env on env-scoped ops. Remove the `authDefaultEnv` hardcode (default only when no
  signal).

### Blast radius
Every `internal/infrastructure/postgres` adapter method that filters by `project_id` also
filters by `environment`; every public handler passes env; openapi adds `X-Environment` to
all public ops; ogen + SDK regen. This is the largest piece — phased below.

---

## B — Registration modes + invites

`auth` config `registration.mode` ∈ `open | closed | request_access | invite_only`, read by
the signup flow at create (project+env), and exposed in `/v1/config/public`.

Flow signup branch (machine steps already exist):
- `open` → normal.
- `closed` → `FlowStepBlocked` (status pending→blocked, reason `registration_closed`).
- `request_access` → `FlowStepRequestAccess` (existing access-request path).
- `invite_only` → require a valid invite; no token → `blocked` (reason `invite_required`).

### Invites (both delivery modes)
New `iam_invites`: `id, project_id, environment, email (nullable), token_hash, status
(pending|accepted|revoked|expired), role/metadata jsonb, expires_at, created_by, accepted_at`.
Token: opaque `inv_…`, sha256 at rest (same pattern as flow/challenge tokens).

- **Admin API** (`/v1/projects/{id}/admin/invites`): create (optional `email` → also send
  the invite email; always returns the raw token once for manual sharing), list, revoke.
- **Email**: new built-in template `invite` (en+ru) with `{{.invite_url}}` =
  `<app_base_url>/continue?...` or a dedicated `/invite?token=` link (reuse app_base_url +
  per-flow redirect rules from the i18n work).
- **Redeem in flow**: `flow.start({ kind: 'signup', inviteToken })` (or a redeem step). Server
  validates token (project+env, pending, unexpired); on success the signup proceeds and the
  invite is marked accepted in the same tx; mismatched `email` (if the invite is email-bound)
  is rejected.

Public config exposes `registration.mode` so the app can pre-block `/register`.

---

## C2 — Password strategy

`auth` config `registration.password_strategy` ∈ `password_first` (default, current) |
`after_verify`. Exposed in public config so the app shows/hides the password field.

Signup flow branch:
- `password_first` → unchanged (collect email+password → verify → done).
- `after_verify` → collect email(+name), **no password** → `verify_email` → `set_password`
  step → create the account on set-password. Requires deferring account creation (register a
  shell/unverified account without a password, or create on set-password); `set_password`
  submit handler finalizes. Anti-enumeration preserved.

---

## Sequencing (phases, each independently shippable + green CI)

1. **B + C2 (no env dependency beyond reading live config)** — fastest user-visible win:
   flow reads auth config, enforces mode, password strategy option, public config exposes
   `registration`. Invites (table + admin CRUD + email + redeem, both delivery modes).
2. **A2 schema** — migration: add `environment` to runtime tables, backfill `live`, rebuild
   unique indexes. No behavior change yet (everything still resolves `live`).
3. **A2 wiring** — `X-Environment` on public ops + SDK; thread env through handlers/adapters;
   per-env signing/mint/verify; remove `authDefaultEnv` hardcode. Admin panel env selector
   becomes meaningful (and B/C2 config now per-env).
4. **Admin/web polish** — env switcher in the client/test tooling; invites UI; registration
   mode + password strategy controls already exist (Configuration → Auth) and now take effect.

Risk: Phase 3 is the high-risk, wide-touch change; gated behind Phase 2's migration. Phases
1–2 deliver enforcement + invites without the big refactor.

## Open questions
- Phase 3: does `X-Environment` default to `live` when absent (back-comp) — yes.
- Invite roles/metadata: minimal now (email binding + expiry); RBAC roles deferred to AuthZ.
- Do API keys / service accounts become env-scoped credentials? Proposal: stamp env on issue,
  keep the row project-level; revisit if test/live key separation is required.
