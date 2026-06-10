CREATE TABLE "public"."iam_access_requests" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "email" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending'::text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_activity" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "type" text NOT NULL,
  "at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_admin_tokens" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "hash" text NOT NULL,
  "expires_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_api_keys" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "prefix" text NOT NULL,
  "hash" text NOT NULL,
  "disabled" bool NOT NULL DEFAULT false,
  "expires_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_app_clients" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "name" text NOT NULL,
  "type" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_app_secrets" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "app_id" text NOT NULL,
  "hash" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_audit_logs" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "type" text NOT NULL,
  "actor_id" text,
  "target_id" text,
  "at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_auth_codes" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "code_hash" text NOT NULL,
  "client_id" text,
  "user_id" text,
  "expires_at" timestamptz NOT NULL,
  "consumed" bool NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_blocks" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "subject" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "expires_at" timestamptz,
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_challenges" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "type" text NOT NULL,
  "subject" text,
  "code_hash" text,
  "expires_at" timestamptz NOT NULL,
  "consumed" bool NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_config" (
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "key" text NOT NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("project_id", "environment", "key")
);
CREATE TABLE "public"."iam_consents" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "doc_key" text NOT NULL,
  "version" text NOT NULL,
  "locale" text,
  "accepted_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_credentials" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "type" text NOT NULL,
  "secret" text NOT NULL DEFAULT ''::text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_device_codes" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "device_code" text NOT NULL,
  "user_code" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending'::text,
  "user_id" text,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_domains" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "connection_id" text,
  "domain" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending'::text,
  "verified_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_email_templates" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "key" text NOT NULL,
  "locale" text NOT NULL DEFAULT 'en'::text,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_environments" (
  "project_id" text NOT NULL,
  "name" text NOT NULL,
  "issuer" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("project_id", "name")
);
CREATE TABLE "public"."iam_events" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "type" text NOT NULL,
  "published" bool NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_factors" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "type" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending'::text,
  "secret" text NOT NULL DEFAULT ''::text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_flows" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "token_hash" text NOT NULL,
  "kind" text NOT NULL,
  "status" text NOT NULL,
  "step" text NOT NULL,
  "user_id" text,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_hooks" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "type" text NOT NULL,
  "enabled" bool NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_identities" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "type" text NOT NULL,
  "provider" text,
  "provider_account_id" text,
  "email" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_interactions" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "client_id" text,
  "session_id" text,
  "expires_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_invites" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "email" text,
  "token_hash" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending'::text,
  "expires_at" timestamptz,
  "accepted_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_jobs" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "type" text NOT NULL,
  "status" text NOT NULL DEFAULT 'running'::text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_oauth_grants" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "client_id" text NOT NULL,
  "granted_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_par_requests" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "request_uri" text NOT NULL,
  "client_id" text,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_projects" (
  "id" text NOT NULL,
  "slug" text NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_providers" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "kind" text NOT NULL,
  "provider" text NOT NULL,
  "enabled" bool NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_recovery_codes" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "hash" text NOT NULL,
  "used" bool NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_refresh_tokens" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "session_id" text NOT NULL,
  "hash" text NOT NULL,
  "revoked" bool NOT NULL DEFAULT false,
  "expires_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_risk_rules" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "enabled" bool NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_scim_resources" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "connection_id" text NOT NULL,
  "resource_type" text NOT NULL,
  "external_id" text,
  "user_id" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_scim_tokens" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "connection_id" text NOT NULL,
  "hash" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_service_accounts" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "name" text NOT NULL,
  "disabled" bool NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_sessions" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "client_id" text,
  "aal" int4 NOT NULL DEFAULT 1,
  "trusted" bool NOT NULL DEFAULT false,
  "expires_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "last_active_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_signing_keys" (
  "kid" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "alg" text NOT NULL DEFAULT 'RS256'::text,
  "use" text NOT NULL DEFAULT 'sig'::text,
  "status" text NOT NULL DEFAULT 'active'::text,
  "private_pem" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("kid")
);
CREATE TABLE "public"."iam_sso_connections" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "type" text NOT NULL,
  "status" text NOT NULL DEFAULT 'active'::text,
  "name" text NOT NULL,
  "external_ref" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_token_profiles" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_users" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "kind" text NOT NULL DEFAULT 'human'::text,
  "status" text NOT NULL DEFAULT 'active'::text,
  "primary_email" text,
  "primary_phone" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_webauthn_credentials" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id" text NOT NULL,
  "credential_id" text NOT NULL,
  "public_key" bytea,
  "sign_count" int8 NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "last_used_at" timestamptz,
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_webhooks" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "enabled" bool NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
ALTER TABLE "public"."iam_flows" ADD CONSTRAINT "iam_flows_token_hash_key" UNIQUE ("token_hash");
CREATE INDEX "idx_iam_access_requests_project" ON "public"."iam_access_requests" ("project_id", "status");
CREATE INDEX "idx_iam_activity_user" ON "public"."iam_activity" ("project_id", "user_id", "at");
CREATE INDEX "idx_iam_admin_tokens_project" ON "public"."iam_admin_tokens" ("project_id");
CREATE INDEX "idx_iam_api_keys_project" ON "public"."iam_api_keys" ("project_id");
CREATE UNIQUE INDEX "uq_iam_api_keys_prefix" ON "public"."iam_api_keys" ("prefix");
CREATE INDEX "idx_iam_app_clients_project" ON "public"."iam_app_clients" ("project_id");
CREATE INDEX "idx_iam_app_secrets_app" ON "public"."iam_app_secrets" ("project_id", "app_id");
CREATE INDEX "idx_iam_audit_project" ON "public"."iam_audit_logs" ("project_id", "at");
CREATE INDEX "idx_iam_auth_codes_hash" ON "public"."iam_auth_codes" ("code_hash");
CREATE INDEX "idx_iam_blocks_project" ON "public"."iam_blocks" ("project_id", "subject");
CREATE INDEX "idx_iam_challenges_subject" ON "public"."iam_challenges" ("project_id", "subject");
CREATE INDEX "idx_iam_consents_user" ON "public"."iam_consents" ("project_id", "user_id");
CREATE INDEX "idx_iam_credentials_user" ON "public"."iam_credentials" ("project_id", "user_id");
CREATE UNIQUE INDEX "uq_iam_device_device_code" ON "public"."iam_device_codes" ("device_code");
CREATE UNIQUE INDEX "uq_iam_device_user_code" ON "public"."iam_device_codes" ("project_id", "environment", "user_code");
CREATE INDEX "idx_iam_domains_project" ON "public"."iam_domains" ("project_id");
CREATE UNIQUE INDEX "uq_iam_domains_domain" ON "public"."iam_domains" ("domain");
CREATE INDEX "idx_iam_email_templates_project" ON "public"."iam_email_templates" ("project_id");
CREATE INDEX "idx_iam_events_unpublished" ON "public"."iam_events" ("created_at") WHERE (published = false);
CREATE INDEX "idx_iam_factors_user" ON "public"."iam_factors" ("project_id", "user_id");
CREATE INDEX "iam_flows_project_idx" ON "public"."iam_flows" ("project_id");
CREATE INDEX "idx_iam_hooks_project" ON "public"."iam_hooks" ("project_id");
CREATE INDEX "idx_iam_identities_user" ON "public"."iam_identities" ("project_id", "user_id");
CREATE UNIQUE INDEX "uq_iam_identities_provider" ON "public"."iam_identities" ("project_id", "environment", "provider", "provider_account_id") WHERE ((provider IS NOT NULL) AND (provider_account_id IS NOT NULL));
CREATE INDEX "idx_iam_invites_hash" ON "public"."iam_invites" ("token_hash");
CREATE INDEX "idx_iam_invites_project" ON "public"."iam_invites" ("project_id", "status");
CREATE INDEX "idx_iam_jobs_project" ON "public"."iam_jobs" ("project_id");
CREATE INDEX "idx_iam_oauth_grants_user" ON "public"."iam_oauth_grants" ("project_id", "user_id");
CREATE UNIQUE INDEX "uq_iam_par_request_uri" ON "public"."iam_par_requests" ("request_uri");
CREATE UNIQUE INDEX "uq_iam_projects_slug" ON "public"."iam_projects" ("slug");
CREATE INDEX "idx_iam_providers_project" ON "public"."iam_providers" ("project_id", "kind");
CREATE INDEX "idx_iam_recovery_user" ON "public"."iam_recovery_codes" ("project_id", "user_id");
CREATE INDEX "idx_iam_refresh_hash" ON "public"."iam_refresh_tokens" ("hash");
CREATE INDEX "idx_iam_refresh_session" ON "public"."iam_refresh_tokens" ("project_id", "session_id");
CREATE INDEX "idx_iam_risk_rules_project" ON "public"."iam_risk_rules" ("project_id");
CREATE INDEX "idx_iam_scim_resources_conn" ON "public"."iam_scim_resources" ("project_id", "connection_id", "resource_type");
CREATE INDEX "idx_iam_scim_tokens_conn" ON "public"."iam_scim_tokens" ("project_id", "connection_id");
CREATE INDEX "idx_iam_service_accounts_project" ON "public"."iam_service_accounts" ("project_id");
CREATE INDEX "idx_iam_sessions_user" ON "public"."iam_sessions" ("project_id", "user_id");
CREATE INDEX "idx_iam_signing_keys_env" ON "public"."iam_signing_keys" ("project_id", "environment");
CREATE INDEX "idx_iam_sso_connections_project" ON "public"."iam_sso_connections" ("project_id");
CREATE INDEX "idx_iam_token_profiles_project" ON "public"."iam_token_profiles" ("project_id");
CREATE INDEX "idx_iam_users_project" ON "public"."iam_users" ("project_id");
CREATE UNIQUE INDEX "uq_iam_users_email" ON "public"."iam_users" ("project_id", "environment", "primary_email") WHERE (primary_email IS NOT NULL);
CREATE UNIQUE INDEX "uq_iam_users_phone" ON "public"."iam_users" ("project_id", "environment", "primary_phone") WHERE (primary_phone IS NOT NULL);
CREATE INDEX "idx_iam_webauthn_user" ON "public"."iam_webauthn_credentials" ("project_id", "user_id");
CREATE UNIQUE INDEX "uq_iam_webauthn_cred" ON "public"."iam_webauthn_credentials" ("project_id", "environment", "credential_id");
CREATE INDEX "idx_iam_webhooks_project" ON "public"."iam_webhooks" ("project_id");

