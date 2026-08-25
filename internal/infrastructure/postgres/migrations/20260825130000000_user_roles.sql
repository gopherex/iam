-- sqld:up
-- Role assignments per project environment. A role is a plain label owned by
-- IAM; the OIDC `groups` scope projects a user's roles into the `groups` claim
-- so a relying party (ArgoCD, Grafana, ...) can map an IAM user onto its own
-- permissions. Values are never taken from the request — only what an admin
-- assigned here can appear in a token.
CREATE TABLE "public"."iam_user_roles" (
  "project_id"  text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "user_id"     text NOT NULL,
  "role"        text NOT NULL,
  "created_at"  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("project_id", "environment", "user_id", "role")
);
CREATE INDEX "idx_iam_user_roles_user" ON "public"."iam_user_roles" ("project_id", "environment", "user_id");

-- sqld:down
DROP TABLE "public"."iam_user_roles";
