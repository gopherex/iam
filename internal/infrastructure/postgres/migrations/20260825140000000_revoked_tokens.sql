-- sqld:up
-- Access tokens are stateless signed JWTs: a resource server verifies them
-- offline and never asks us, so RFC 7009 revocation had no effect on them at
-- all. Naming a revoked token's jti here gives the verification path something
-- to check, and the row is swept once the token would have expired regardless.
CREATE TABLE "public"."iam_revoked_tokens" (
  "jti"         text PRIMARY KEY,
  "project_id"  text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "expires_at"  timestamptz NOT NULL,
  "revoked_at"  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX "idx_iam_revoked_tokens_expiry" ON "public"."iam_revoked_tokens" ("expires_at");

-- sqld:down
DROP TABLE "public"."iam_revoked_tokens";
