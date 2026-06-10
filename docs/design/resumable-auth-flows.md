# Server-side resumable auth flows

Status: design / contract (authoritative). Owner: core. Implementation fans out
from this document — every task references a section here.

## 1. Goal

Move the auth state machine from the client to the server. The client holds only
an opaque `flow_token`; all state (current step, contact, active challenge,
attempts, consents, TTL) lives server-side. One resource (`flow`) with four
verbs: `create`, `get`, `submit`, `resend` (+ optional `abandon`). Every response
returns the full `FlowState`; the client renders it without computing anything.

This removes client-side `nextStep` branching, the separate "send verification"
call, and manual `challengeId` handling. Sign-up returns `verify_email` with an
already-issued challenge — one fewer round-trip.

## 2. Non-goals (this milestone = backend only)

- TS SDK stateful wrapper (`flow.start/submit/resend/onChange`, BroadcastChannel)
  and the web `/flow` page — separate later milestone. Backend is the source of
  truth; the SDK is generated from the openapi added here plus a hand wrapper.
- Cross-device deep-link / `iam_flow` httpOnly cookie — Phase 3 (the token rules
  here already allow it; the cookie binding is additive).

## 3. Principle

`kind` ∈ {`signup`, `signin`, `recovery`, `email_change`} selects the machine.
The server advances `step` on each `submit`/`resend` and returns `FlowState`. The
flow orchestrates EXISTING domain operations (Register, AuthenticatePassword,
StartEmailVerification/VerifyEmail, OTP, MFA Challenge/Verify, recovery codes,
CreateAccessRequest) — it does NOT reimplement them. Legacy `/v1/auth/*` stay.

## 4. Data model — `iam_flows`

New table (postgres; no Redis — stacks deferred). Envelope pattern like the rest
of the store: typed lookup columns + `data jsonb` for the rich state.

```
CREATE TABLE iam_flows (
  id          text PRIMARY KEY,         -- internal uuid
  project_id  text NOT NULL,            -- tenant boundary (X-Client-Id)
  token_hash  text NOT NULL UNIQUE,     -- sha256 of the opaque flow_token (never store the token)
  kind        text NOT NULL,            -- signup | signin | recovery | email_change
  status      text NOT NULL,            -- pending | completed | expired | aborted
  step        text NOT NULL,            -- see §6
  user_id     text,                     -- set once the user is created/resolved
  expires_at  timestamptz NOT NULL,     -- whole-flow TTL (default 30m)
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL            -- contact, active_challenge, collected, consents, attempts, etc.
);
CREATE INDEX iam_flows_project_idx ON iam_flows (project_id);
```

`data` (json) shape:
```
{
  "contact":   { "email": "...", "phone": "..." },      // raw, server-only
  "collected": { "name": "...", "has_password": true }, // NEVER raw password
  "active_challenge": { "challenge_id": "...", "channel": "email", "expires_at": ms,
                         "resend_at": ms, "attempts_left": 5 },
  "consents_required": [ { "key": "tos", "version": "3" } ],
  "registration_mode": "open"
}
```

The active second factor / email-verify code is NOT in `iam_flows` — it lives in
`iam_challenges` (existing), referenced by `active_challenge.challenge_id`. The
flow reuses the challenge machinery for codes, attempts, single-use.

## 5. Security (MUST — best practice; NIST 800-63B, RFC 6819/9700)

1. **flow_token**: ≥256-bit opaque random, URL-safe, prefixed `ftk_`. Store ONLY
   `sha256(token)` in `token_hash`; compare in constant time. The raw token is
   returned in the response body once per rotation and never persisted.
2. **Rotation**: issue a NEW token (new `token_hash`) on every privilege
   transition — at minimum after a successful factor (verify_email, mfa verify).
   Old token immediately invalid. Mitigates token capture / fixation.
3. **Tenant + TTL**: every lookup is scoped by `project_id` (X-Client-Id) AND
   `expires_at > now` AND `status = pending`. Expired/foreign/completed → 410 /
   not-found (never leak).
4. **Anti-enumeration**: `create` for `signin`/`recovery` MUST NOT reveal whether
   the email exists. Always return `pending` + `verify_*`/generic step with a
   masked contact; do the real lookup server-side. Timing-stable where feasible.
5. **No raw secrets in flow**: passwords are bcrypt-hashed by the existing domain
   op at the step that consumes them; never stored in `data`. `collected` records
   only `has_password: true`.
6. **Attempts / lockout**: `active_challenge.attempts_left` decrements on a wrong
   code; at 0 the challenge is dead and the step requires `resend`. A global
   per-flow failure cap aborts the flow (`status = aborted`). Wrong code returns
   `error` WITHOUT advancing or resetting (`status` stays `pending`).
7. **resend rate-limit**: `resend_at` gates re-issue; early resend → 429 with
   `resend_at`. Reuse the existing rate-limit middleware + per-challenge cooldown.
8. **Completion**: `session` is returned exactly once, when `status` becomes
   `completed`; the flow row is then terminal. A completed/aborted flow_token
   grants nothing.
9. **MFA**: `signin` completion MUST honor the MFA step-up built in
   `feat(auth): MFA step-up` — when the account has an active factor the flow
   enters `mfa_required` and only completes after the second factor.
10. **Masking**: contact shown to the client is masked (`a***@b.ru`) so the
    client never needs to store the raw email.

## 6. Steps / next_actions

```
collect_credentials | verify_email | verify_phone | set_password |
mfa_required | step_up | accept_consents | request_access |
awaiting_approval | completed | blocked
```
- `completed` → response carries `session`.
- `request_access` / `awaiting_approval` → `registration.mode = request_access`.
- `blocked` (+ `error.code`) → `closed` / `invite_only`.
`next_actions` is the machine-readable subset of allowed `submit` actions for the
current step (always includes `resend` when a challenge is active).

## 7. State machines (per kind)

- **signup**: `collect_credentials` → (mode gate) → `verify_email` (challenge
  auto-issued) → [`accept_consents`] → `completed`. Modes: `open` → continue;
  `request_access` → `request_access` → `awaiting_approval`; `invite_only`/`closed`
  → `blocked`.
- **signin**: `collect_credentials` → (password verify) → [`mfa_required`] →
  `completed`. Unknown user → still `verify`/generic to avoid enumeration, fails
  at submit with a neutral `error`.
- **recovery**: `collect_credentials`(email) → `verify_email`(OTP) → `set_password`
  → `completed`. Orchestrates password/forgot + reset.
- **email_change**: requires an authenticated principal at `create`; `verify_email`
  (new address) → `completed`. Orchestrates email/change/start+verify.

## 8. API contract (openapi-first → ogen regen)

All under tag `Core Auth`, `x-ogen-operation-group: CoreAuth`. `X-Client-Id`
required on every call. `security: []` (public) — the flow_token is the
credential, except `email_change` create which also accepts a bearer.

- `POST /v1/auth/flows` — body `{ kind, email?, password?, name?, captcha_token? }`
  → `200 FlowState`.
- `GET  /v1/auth/flows/current` — by `iam_flow` cookie (Phase 3) → `200 | 404`.
- `GET  /v1/auth/flows/{flow_token}` — explicit resume → `200 FlowState | 410`.
- `POST /v1/auth/flows/{flow_token}/submit` — body `{ action, payload }` →
  `200 FlowState` (advanced; `error` set + `pending` on bad input).
- `POST /v1/auth/flows/{flow_token}/resend` → `200 FlowState | 429`.
- `DELETE /v1/auth/flows/{flow_token}` → `204`.

`FlowState` schema (response): `flow_token, kind, status, step, next_actions[],
contact{email_masked?, phone_masked?}, challenge{channel, expires_at, resend_at,
attempts_left}?, consents_required[]?, expires_at, error{code,message}?, session?`.
`session` reuses the existing `SessionTokens`/`AuthResult` shape. `error.code` is
a stable machine code (client branches on code, not HTTP status).

## 9. Architecture

- `domain`: `Flow`, `FlowState`, `FlowKind`, `FlowStep`, command types, errors.
- Port `CoreAuthFlows` (pkg/api): `Create/Get/Submit/Resend/Abandon`.
- `pgCoreAuthFlows` (postgres): the engine — flow store CRUD (hashed token,
  rotation, TTL, attempts) + per-kind `advance` functions that call the existing
  coreauth/mfa adapters. Reuses `iam_challenges` for codes.
- `CoreAuthFlowService` (pkg/api): thin handlers mapping HTTP ↔ port, building
  `FlowState`.
- Wire into `buildHandler` + the e2e harness.

## 10. Task decomposition (each → its own implementation unit)

Foundation (single-authored, owned by core; must land first):
- F1 contract: this doc + openapi schemas/endpoints + `make generate-go`.
- F2 store: `iam_flows` migration + schema.sql + `db-gen`; flow store CRUD with
  hashed token, rotation, TTL, attempts; domain types + port skeleton; FlowState
  mapper; handlers + wiring. The engine spine, no per-kind logic yet.

Parallelizable on top of F2 (one unit each):
- T-signup: signup `advance` + registration-mode gate + consents + e2e.
- T-signin: signin `advance` + MFA step-up integration + e2e.
- T-recovery: recovery `advance` (forgot/reset orchestration) + e2e.
- T-emailchange: email_change `advance` (authenticated) + e2e.
- T-harden: anti-enumeration timing, attempts/lockout, resend rate-limit, token
  rotation tests (adversarial e2e across kinds).

Out of scope here (later milestone): TS SDK wrapper, web `/flow` page, deep-link,
`iam_flow` cookie.

## 11. Definition of done (backend milestone)

`go build ./...` + `go vet` clean; `make test-db` green including new
`integration_e2e_flow_*_test.go`; openapi validates; ogen regenerated; no legacy
endpoint regressions; security items in §5 covered by tests.
