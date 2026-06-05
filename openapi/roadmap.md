# IAM API — Release Roadmap

> Source of truth for per-operation versioning is the `x-since` field in
> `openapi.yaml` (filter plane `since`). This file describes **what mechanics
> ship in each version**, not individual endpoints.
>
> - **Minor** (`1.x`) = additive auth method or operational surface; no breaking change.
> - **Major** (`2.0`, `3.0`) = new product role/surface.
>
> Slicing axis is **feature**: each release ships a runtime capability plus the
> admin/operator surface needed to configure and run it. Cross-cutting admin
> tags are split op-by-op so each operation lands with the feature it configures.

## v1.0.0 — Foundation + password auth

First runnable product. A single project/env can do full password-based auth.

- **Operator**: project & environment provisioning, admin-token minting, config plan/apply (IaC).
- **Core auth**: sign-up/sign-in (password), refresh, token exchange, sign-out, session, step-up, guest, request-access.
- **Verification**: email & phone verification and change.
- **Password**: forgot/reset/change/verify/check.
- **Tokens**: runtime introspection, verification, revocation.
- **Account**: current-user profile, consents, activity, GDPR export; linked identities & merge; session/device management.
- **Admin baseline**: users CRUD + lifecycle (ban, anonymize, impersonate, password set), app clients, signing keys & token profiles, auth/password/session policy, feature toggles, email/SMS providers & templates, i18n, consents, access-request review.

## v1.1.0 — Passwordless + MFA

- **Passwordless**: OTP and magic-link login/signup.
- **MFA**: TOTP/SMS/email/WebAuthn factors, challenge/verify, recovery codes, MFA policy, admin MFA reset.

## v1.2.0 — Social + WebAuthn

- **OAuth social**: social login as an OAuth client, identity linking, provider config.
- **WebAuthn**: passkey login & registration, credential management.

## v1.3.0 — Machine identity

- **Service accounts**: machine identities, secrets, short-lived token minting.
- **API keys**: static scoped credentials with rotation.

## v1.4.0 — Operational hardening

- **Webhooks & hooks**: event delivery, deliveries, replay, blocking hooks.
- **Jobs**: background jobs, bulk user import, audit logs & export, retention policy.
- **Risk**: risk rules & events, rate limits, blocks.
- **Test mode**: test clock, seed, reset, captured-message inbox.

## v2.0.0 — IAM as OIDC provider

New product role: IAM becomes an OAuth2/OIDC identity provider.

- **OIDC provider**: authorize, token, userinfo, introspect, revoke, PAR, logout, discovery & JWKS.
- **OIDC interaction**: headless login/consent, grant management (incl. admin view of user grants).
- **Device flow**: device authorization grant.

## v3.0.0 — Enterprise federation

New product surface: enterprise SSO and provisioning.

- **Enterprise SSO**: SAML/OIDC connections and runtime.
- **SCIM**: SCIM 2.0 user/group provisioning and connection tokens.
- **Domains**: verified-domain management.

## Notable op-by-op admin placements

Cross-cutting admin operations follow the feature they configure, not their tag:

- `users/{id}/grants` → **v2.0.0** — OAuth consent grants are meaningless before the provider exists.
- `config/rate-limits`, `risk/*`, `rate-limit/blocks` → **v1.4.0** — risk engine ships with hardening.
- admin `access-requests` → **v1.0.0** — mirrors runtime request-access mode.
- `config/retention-policy` → **v1.4.0** — pairs with GDPR export/anonymize jobs.
- `config/mfa-policy`, `users/{id}/mfa/reset` → **v1.1.0** — ride with MFA.
- `oauth-providers` config → **v1.2.0** — rides with social login.
