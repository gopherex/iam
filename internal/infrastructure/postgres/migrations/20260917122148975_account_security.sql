-- sqld:up
CREATE TABLE "public"."iam_security_attempts" (
  "project_id" text NOT NULL,
  "environment" text NOT NULL,
  "subject_hash" text NOT NULL,
  "kind" text NOT NULL,
  "window_start" timestamptz NOT NULL,
  "attempts" int4 NOT NULL,
  PRIMARY KEY ("project_id", "environment", "subject_hash", "kind")
);
CREATE TABLE "public"."iam_security_case_decisions" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL,
  "case_id" text NOT NULL,
  "actor_id" text NOT NULL,
  "action" text NOT NULL,
  "evidence" text NOT NULL,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_security_cases" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL,
  "user_id" text NOT NULL DEFAULT ''::text,
  "status" text NOT NULL,
  "created_at" timestamptz NOT NULL,
  "data" jsonb NOT NULL,
  "private_data" text NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_security_continuations" (
  "token_hash" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL,
  "user_id" text NOT NULL,
  "incident_id" text NOT NULL DEFAULT ''::text,
  "case_id" text NOT NULL DEFAULT ''::text,
  "purpose" text NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "consumed" bool NOT NULL DEFAULT false,
  PRIMARY KEY ("token_hash")
);
CREATE TABLE "public"."iam_security_deliveries" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL,
  "user_id" text NOT NULL,
  "dedup_key" text NOT NULL,
  "status" text NOT NULL,
  "created_at" timestamptz NOT NULL,
  "data" jsonb NOT NULL,
  "private_data" text NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_security_devices" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL,
  "user_id" text NOT NULL,
  "token_hash" text NOT NULL,
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_security_incidents" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL,
  "user_id" text NOT NULL,
  "status" text NOT NULL,
  "created_at" timestamptz NOT NULL,
  "data" jsonb NOT NULL,
  "private_data" text NOT NULL DEFAULT ''::text,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_security_policies" (
  "project_id" text NOT NULL,
  "environment" text NOT NULL,
  "data" jsonb NOT NULL,
  PRIMARY KEY ("project_id", "environment")
);
ALTER TABLE "public"."iam_security_deliveries" ADD CONSTRAINT "iam_security_deliveries_dedup_key_key" UNIQUE ("dedup_key");
ALTER TABLE "public"."iam_security_devices" ADD CONSTRAINT "iam_security_devices_token_hash_key" UNIQUE ("token_hash");
CREATE INDEX "idx_security_cases_queue" ON "public"."iam_security_cases" ("project_id", "environment", "id");
CREATE INDEX "idx_security_deliveries_queue" ON "public"."iam_security_deliveries" ("project_id", "environment", "id");
CREATE INDEX "idx_security_devices_user" ON "public"."iam_security_devices" ("project_id", "environment", "user_id", "id");
CREATE INDEX "idx_security_incidents_retention" ON "public"."iam_security_incidents" ("project_id", "environment", "created_at");
CREATE INDEX "idx_security_incidents_user" ON "public"."iam_security_incidents" ("project_id", "environment", "user_id", "id");

CREATE TABLE iam_security_guards (flow_id text NOT NULL, project_id text NOT NULL, environment text NOT NULL, user_id text NOT NULL, created_at timestamptz NOT NULL, PRIMARY KEY(project_id,environment,user_id));

-- sqld:down
DROP TABLE iam_security_guards;
DROP INDEX "public"."idx_security_incidents_user";
DROP INDEX "public"."idx_security_incidents_retention";
DROP INDEX "public"."idx_security_devices_user";
DROP INDEX "public"."idx_security_deliveries_queue";
DROP INDEX "public"."idx_security_cases_queue";
ALTER TABLE "public"."iam_security_devices" DROP CONSTRAINT "iam_security_devices_token_hash_key";
ALTER TABLE "public"."iam_security_deliveries" DROP CONSTRAINT "iam_security_deliveries_dedup_key_key";
DROP TABLE "public"."iam_security_policies";
DROP TABLE "public"."iam_security_incidents";
DROP TABLE "public"."iam_security_devices";
DROP TABLE "public"."iam_security_deliveries";
DROP TABLE "public"."iam_security_continuations";
DROP TABLE "public"."iam_security_cases";
DROP TABLE "public"."iam_security_case_decisions";
DROP TABLE "public"."iam_security_attempts";
