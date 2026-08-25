-- Authoritative schema for the IAM Postgres store. sqld reads this to generate
-- gen/db (typed query funcs), gen/bob (bob query builders) and the bootstrap
-- migration.
--
-- Storage model (komeet/stroppy pattern): each aggregate is one table carrying
-- the queryable envelope columns (id, project_id, created_at, updated_at, plus
-- secondary lookup keys) and the full domain object in a `data jsonb` column.
-- IAM is project-scoped, so project_id is the partition key on every tenant
-- table. Adapters prefer the generated bob query builders; the sqld(c) typed
-- funcs are reserved for super-hot paths.

-- ============================================================
-- Identity core
-- ============================================================

CREATE TABLE iam_users (
  id            text PRIMARY KEY,
  project_id    text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  kind          text NOT NULL DEFAULT 'human',
  status        text NOT NULL DEFAULT 'active',
  primary_email text,
  primary_phone text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  data          jsonb NOT NULL
);
CREATE INDEX idx_iam_users_project ON iam_users (project_id);
CREATE UNIQUE INDEX uq_iam_users_email ON iam_users (project_id, environment, primary_email) WHERE primary_email IS NOT NULL;
CREATE UNIQUE INDEX uq_iam_users_phone ON iam_users (project_id, environment, primary_phone) WHERE primary_phone IS NOT NULL;

-- Role assignments. A role is a plain label owned by IAM; it is what the OIDC
-- `groups` scope projects into the `groups` claim, so a relying party (ArgoCD,
-- Grafana, ...) can map an IAM user onto its own permissions. Scoped to
-- project+environment: the same person can be an operator in test and a viewer
-- in live.
CREATE TABLE iam_user_roles (
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id     text NOT NULL,
  role        text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, environment, user_id, role)
);
CREATE INDEX idx_iam_user_roles_user ON iam_user_roles (project_id, environment, user_id);

CREATE TABLE iam_credentials (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id    text NOT NULL,
  type       text NOT NULL,          -- password
  secret     text NOT NULL DEFAULT '', -- hash (argon2/bcrypt)
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_credentials_user ON iam_credentials (project_id, user_id);

CREATE TABLE iam_identities (
  id                  text PRIMARY KEY,
  project_id          text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id             text NOT NULL,
  type                text NOT NULL,   -- password | oauth | saml | oidc | passkey
  provider            text,
  provider_account_id text,
  email               text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  data                jsonb NOT NULL
);
CREATE INDEX idx_iam_identities_user ON iam_identities (project_id, user_id);
CREATE UNIQUE INDEX uq_iam_identities_provider ON iam_identities (project_id, environment, provider, provider_account_id)
  WHERE provider IS NOT NULL AND provider_account_id IS NOT NULL;

CREATE TABLE iam_sessions (
  id             text PRIMARY KEY,
  project_id     text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id        text NOT NULL,
  client_id      text,
  aal            integer NOT NULL DEFAULT 1,
  trusted        boolean NOT NULL DEFAULT false,
  expires_at     timestamptz,
  created_at     timestamptz NOT NULL DEFAULT now(),
  last_active_at timestamptz NOT NULL DEFAULT now(),
  data           jsonb NOT NULL
);
CREATE INDEX idx_iam_sessions_user ON iam_sessions (project_id, user_id);

CREATE TABLE iam_refresh_tokens (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id    text NOT NULL,
  session_id text NOT NULL,
  hash       text NOT NULL,
  revoked    boolean NOT NULL DEFAULT false,
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_refresh_session ON iam_refresh_tokens (project_id, session_id);
CREATE INDEX idx_iam_refresh_hash ON iam_refresh_tokens (hash);

-- ============================================================
-- MFA / passkeys / challenges
-- ============================================================

-- Revoked stateless tokens, by jti. Access tokens are signed JWTs verified
-- offline, so the only way to kill one before it expires is to name it here;
-- rows are swept once the token they name has expired anyway.
CREATE TABLE iam_revoked_tokens (
  jti         text PRIMARY KEY,
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  expires_at  timestamptz NOT NULL,
  revoked_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_iam_revoked_tokens_expiry ON iam_revoked_tokens (expires_at);

CREATE TABLE iam_factors (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id    text NOT NULL,
  type       text NOT NULL,   -- totp | sms | email | webauthn
  status     text NOT NULL DEFAULT 'pending',
  secret     text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_factors_user ON iam_factors (project_id, user_id);

CREATE TABLE iam_webauthn_credentials (
  id           text PRIMARY KEY,
  project_id   text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id      text NOT NULL,
  credential_id text NOT NULL,
  public_key   bytea,
  sign_count   bigint NOT NULL DEFAULT 0,
  created_at   timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz,
  data         jsonb NOT NULL
);
CREATE INDEX idx_iam_webauthn_user ON iam_webauthn_credentials (project_id, user_id);
CREATE UNIQUE INDEX uq_iam_webauthn_cred ON iam_webauthn_credentials (project_id, environment, credential_id);

CREATE TABLE iam_recovery_codes (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id    text NOT NULL,
  hash       text NOT NULL,
  used       boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_iam_recovery_user ON iam_recovery_codes (project_id, user_id);

CREATE TABLE iam_challenges (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  type       text NOT NULL,   -- otp | mfa | email | phone | passkey | consent | merge
  subject    text,            -- email/phone/user being challenged
  code_hash  text,
  expires_at timestamptz NOT NULL,
  consumed   boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_challenges_subject ON iam_challenges (project_id, subject);

CREATE TABLE iam_flows (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  token_hash  text NOT NULL UNIQUE,
  kind        text NOT NULL,
  status      text NOT NULL,
  step        text NOT NULL,
  user_id     text,
  expires_at  timestamptz NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX iam_flows_project_idx ON iam_flows (project_id);

CREATE TABLE iam_consents (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id    text NOT NULL,
  doc_key    text NOT NULL,
  version    text NOT NULL,
  locale     text,
  accepted_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_iam_consents_user ON iam_consents (project_id, user_id);

CREATE TABLE iam_invites (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  email       text,                              -- null = not bound to a specific address
  token_hash  text NOT NULL,
  status      text NOT NULL DEFAULT 'pending',   -- pending | accepted | revoked
  expires_at  timestamptz,
  accepted_at timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_invites_project ON iam_invites (project_id, status);
CREATE INDEX idx_iam_invites_hash ON iam_invites (token_hash);

-- ============================================================
-- Machine identity & app clients
-- ============================================================

CREATE TABLE iam_service_accounts (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  name       text NOT NULL,
  disabled   boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_service_accounts_project ON iam_service_accounts (project_id);

CREATE TABLE iam_api_keys (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  prefix      text NOT NULL,
  hash        text NOT NULL,
  disabled    boolean NOT NULL DEFAULT false,
  expires_at  timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_api_keys_project ON iam_api_keys (project_id);
CREATE UNIQUE INDEX uq_iam_api_keys_prefix ON iam_api_keys (prefix);

CREATE TABLE iam_app_clients (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  name        text NOT NULL,
  type        text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_app_clients_project ON iam_app_clients (project_id);
-- A client managing itself (RFC 7592) presents a registration access token; it
-- is matched by digest, which without this index is a scan over every client.
CREATE INDEX idx_iam_app_clients_registration_token
ON iam_app_clients ((data ->> 'RegistrationTokenHash'))
WHERE data ->> 'RegistrationTokenHash' IS NOT NULL;

CREATE TABLE iam_app_secrets (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  app_id     text NOT NULL,
  hash       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_app_secrets_app ON iam_app_secrets (project_id, app_id);

-- ============================================================
-- Federation (SSO / SCIM / domains)
-- ============================================================

CREATE TABLE iam_sso_connections (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  type        text NOT NULL,   -- saml | oidc
  status      text NOT NULL DEFAULT 'active',
  name        text NOT NULL,
  external_ref text,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_sso_connections_project ON iam_sso_connections (project_id);

CREATE TABLE iam_domains (
  id            text PRIMARY KEY,
  project_id    text NOT NULL,
  connection_id text,
  domain        text NOT NULL,
  status        text NOT NULL DEFAULT 'pending',
  verified_at   timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  data          jsonb NOT NULL
);
CREATE INDEX idx_iam_domains_project ON iam_domains (project_id);
CREATE UNIQUE INDEX uq_iam_domains_domain ON iam_domains (domain);

CREATE TABLE iam_scim_tokens (
  id            text PRIMARY KEY,
  project_id    text NOT NULL,
  connection_id text NOT NULL,
  hash          text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  data          jsonb NOT NULL
);
CREATE INDEX idx_iam_scim_tokens_conn ON iam_scim_tokens (project_id, connection_id);

CREATE TABLE iam_scim_resources (
  id            text PRIMARY KEY,
  project_id    text NOT NULL,
  connection_id text NOT NULL,
  resource_type text NOT NULL,   -- User | Group
  external_id   text,
  user_id       text,            -- linked IAM user, for User resources
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  data          jsonb NOT NULL
);
CREATE INDEX idx_iam_scim_resources_conn ON iam_scim_resources (project_id, connection_id, resource_type);

-- ============================================================
-- OAuth/OIDC provider
-- ============================================================

CREATE TABLE iam_oauth_grants (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id    text NOT NULL,
  client_id  text NOT NULL,
  granted_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_oauth_grants_user ON iam_oauth_grants (project_id, user_id);

CREATE TABLE iam_interactions (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  client_id  text,
  session_id text,           -- bound session (anti-hijack)
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);

CREATE TABLE iam_auth_codes (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  code_hash  text NOT NULL,
  client_id  text,
  user_id    text,
  expires_at timestamptz NOT NULL,
  consumed   boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_auth_codes_hash ON iam_auth_codes (code_hash);

CREATE TABLE iam_par_requests (
  id           text PRIMARY KEY,
  project_id   text NOT NULL,
  request_uri  text NOT NULL,
  client_id    text,
  expires_at   timestamptz NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  data         jsonb NOT NULL
);
CREATE UNIQUE INDEX uq_iam_par_request_uri ON iam_par_requests (request_uri);

CREATE TABLE iam_device_codes (
  id           text PRIMARY KEY,
  project_id   text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  device_code  text NOT NULL,
  user_code    text NOT NULL,
  status       text NOT NULL DEFAULT 'pending',
  user_id      text,
  expires_at   timestamptz NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  data         jsonb NOT NULL
);
CREATE UNIQUE INDEX uq_iam_device_user_code ON iam_device_codes (project_id, environment, user_code);
CREATE UNIQUE INDEX uq_iam_device_device_code ON iam_device_codes (device_code);

-- ============================================================
-- Keys & projects
-- ============================================================

CREATE TABLE iam_projects (
  id            text PRIMARY KEY,
  slug          text NOT NULL,
  name          text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  data          jsonb NOT NULL
);
CREATE UNIQUE INDEX uq_iam_projects_slug ON iam_projects (slug);

CREATE TABLE iam_environments (
  project_id text NOT NULL,
  name       text NOT NULL,
  issuer     text,
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL,
  PRIMARY KEY (project_id, name)
);

CREATE TABLE iam_signing_keys (
  kid         text PRIMARY KEY,
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  alg         text NOT NULL DEFAULT 'RS256',
  use         text NOT NULL DEFAULT 'sig',
  status      text NOT NULL DEFAULT 'active',
  private_pem text,
  created_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_signing_keys_env ON iam_signing_keys (project_id, environment);

CREATE TABLE iam_token_profiles (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  name       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_token_profiles_project ON iam_token_profiles (project_id);

CREATE TABLE iam_admin_tokens (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  hash       text NOT NULL,
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_admin_tokens_project ON iam_admin_tokens (project_id);

-- ============================================================
-- Configuration (per project/env JSON blobs)
-- ============================================================

CREATE TABLE iam_config (
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  key         text NOT NULL,   -- auth | password_policy | session_policy | mfa_policy | consent | retention | features | i18n | risk | rate_limits
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL,
  PRIMARY KEY (project_id, environment, key)
);

CREATE TABLE iam_providers (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  kind        text NOT NULL,   -- email | sms | oauth
  provider    text NOT NULL,
  enabled     boolean NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_providers_project ON iam_providers (project_id, kind);

CREATE TABLE iam_email_templates (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  key         text NOT NULL,
  locale      text NOT NULL DEFAULT 'en',
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_email_templates_project ON iam_email_templates (project_id);

-- ============================================================
-- Webhooks / hooks / jobs / audit / access requests / risk
-- ============================================================

CREATE TABLE iam_webhooks (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  idempotency_key text NOT NULL DEFAULT '',
  enabled     boolean NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_webhooks_project ON iam_webhooks (project_id);
CREATE INDEX idx_iam_webhooks_project_env ON iam_webhooks (project_id, environment);
CREATE UNIQUE INDEX idx_iam_webhooks_idempotency
  ON iam_webhooks (project_id, environment, idempotency_key)
  WHERE idempotency_key <> '';

CREATE TABLE iam_hooks (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  type       text NOT NULL,
  enabled    boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_hooks_project ON iam_hooks (project_id);

CREATE TABLE iam_jobs (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  type       text NOT NULL,
  status     text NOT NULL DEFAULT 'running',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_jobs_project ON iam_jobs (project_id);

CREATE TABLE iam_audit_logs (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  type       text NOT NULL,
  actor_id   text,
  target_id  text,
  at         timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_audit_project ON iam_audit_logs (project_id, at);

CREATE TABLE iam_access_requests (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  email      text NOT NULL,
  status     text NOT NULL DEFAULT 'pending',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_access_requests_project ON iam_access_requests (project_id, status);

CREATE TABLE iam_risk_rules (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  enabled    boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_risk_rules_project ON iam_risk_rules (project_id);

CREATE TABLE iam_blocks (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  subject    text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz,
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_blocks_project ON iam_blocks (project_id, subject);

-- ============================================================
-- Activity & events (events = future transactional outbox; no outbox logic yet)
-- ============================================================

CREATE TABLE iam_activity (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  user_id    text NOT NULL,
  type       text NOT NULL,
  at         timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_activity_user ON iam_activity (project_id, user_id, at);

CREATE TABLE iam_events (
  id           text PRIMARY KEY,
  project_id   text NOT NULL,
  environment  text NOT NULL DEFAULT 'live',
  aggregate_id text NOT NULL DEFAULT '',
  user_id      text NOT NULL DEFAULT '',
  type         text NOT NULL,
  published    boolean NOT NULL DEFAULT false,
  created_at   timestamptz NOT NULL DEFAULT now(),
  data         jsonb NOT NULL
);
CREATE INDEX idx_iam_events_unpublished ON iam_events (created_at) WHERE published = false;
CREATE INDEX idx_iam_events_project_created ON iam_events (project_id, environment, created_at DESC, id DESC);
CREATE INDEX idx_iam_events_user_created ON iam_events (project_id, environment, user_id, created_at DESC) WHERE user_id <> '';

CREATE TABLE iam_webhook_deliveries (
  id              text PRIMARY KEY,
  project_id      text NOT NULL,
  environment     text NOT NULL DEFAULT 'live',
  webhook_id      text NOT NULL,
  event_id        text NOT NULL,
  status          text NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'succeeded', 'failed')),
  attempt_count   integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  next_attempt_at timestamptz,
  last_attempt_at timestamptz,
  delivered_at    timestamptz,
  response_status integer,
  response_body   text,
  last_error      text,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  data            jsonb NOT NULL DEFAULT '{}'::jsonb,
  UNIQUE (webhook_id, event_id)
);
CREATE INDEX idx_iam_webhook_deliveries_project_created
  ON iam_webhook_deliveries (project_id, environment, created_at DESC, id DESC);
CREATE INDEX idx_iam_webhook_deliveries_retry
  ON iam_webhook_deliveries (status, next_attempt_at)
  WHERE status IN ('pending', 'failed');
