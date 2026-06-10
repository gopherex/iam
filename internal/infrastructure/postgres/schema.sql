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
  kind          text NOT NULL DEFAULT 'human',
  status        text NOT NULL DEFAULT 'active',
  primary_email text,
  primary_phone text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  data          jsonb NOT NULL
);
CREATE INDEX idx_iam_users_project ON iam_users (project_id);
CREATE UNIQUE INDEX uq_iam_users_email ON iam_users (project_id, primary_email) WHERE primary_email IS NOT NULL;
CREATE UNIQUE INDEX uq_iam_users_phone ON iam_users (project_id, primary_phone) WHERE primary_phone IS NOT NULL;

CREATE TABLE iam_credentials (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
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
  user_id             text NOT NULL,
  type                text NOT NULL,   -- password | oauth | saml | oidc | passkey
  provider            text,
  provider_account_id text,
  email               text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  data                jsonb NOT NULL
);
CREATE INDEX idx_iam_identities_user ON iam_identities (project_id, user_id);
CREATE UNIQUE INDEX uq_iam_identities_provider ON iam_identities (project_id, provider, provider_account_id)
  WHERE provider IS NOT NULL AND provider_account_id IS NOT NULL;

CREATE TABLE iam_sessions (
  id             text PRIMARY KEY,
  project_id     text NOT NULL,
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

CREATE TABLE iam_factors (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
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
  user_id      text NOT NULL,
  credential_id text NOT NULL,
  public_key   bytea,
  sign_count   bigint NOT NULL DEFAULT 0,
  created_at   timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz,
  data         jsonb NOT NULL
);
CREATE INDEX idx_iam_webauthn_user ON iam_webauthn_credentials (project_id, user_id);
CREATE UNIQUE INDEX uq_iam_webauthn_cred ON iam_webauthn_credentials (project_id, credential_id);

CREATE TABLE iam_recovery_codes (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  user_id    text NOT NULL,
  hash       text NOT NULL,
  used       boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_iam_recovery_user ON iam_recovery_codes (project_id, user_id);

CREATE TABLE iam_challenges (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
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
  user_id    text NOT NULL,
  doc_key    text NOT NULL,
  version    text NOT NULL,
  locale     text,
  accepted_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_iam_consents_user ON iam_consents (project_id, user_id);

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
  user_id    text NOT NULL,
  client_id  text NOT NULL,
  granted_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_oauth_grants_user ON iam_oauth_grants (project_id, user_id);

CREATE TABLE iam_interactions (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  client_id  text,
  session_id text,           -- bound session (anti-hijack)
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);

CREATE TABLE iam_auth_codes (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
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
  device_code  text NOT NULL,
  user_code    text NOT NULL,
  status       text NOT NULL DEFAULT 'pending',
  user_id      text,
  expires_at   timestamptz NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  data         jsonb NOT NULL
);
CREATE UNIQUE INDEX uq_iam_device_user_code ON iam_device_codes (project_id, user_code);
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
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  enabled    boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_webhooks_project ON iam_webhooks (project_id);

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
  user_id    text NOT NULL,
  type       text NOT NULL,
  at         timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_activity_user ON iam_activity (project_id, user_id, at);

CREATE TABLE iam_events (
  id          text PRIMARY KEY,
  project_id  text NOT NULL,
  environment text NOT NULL DEFAULT 'live',
  type        text NOT NULL,
  published   boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now(),
  data        jsonb NOT NULL
);
CREATE INDEX idx_iam_events_unpublished ON iam_events (created_at) WHERE published = false;
