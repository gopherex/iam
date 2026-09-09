-- sqld:up
-- Member invitations: who created an invite. NULL keeps the admin/system
-- semantics (admin API, access-request approvals, the telegram bot) — those
-- are not subject to per-user invite caps. Non-null is a user id: the row
-- occupies one of that user's invite slots (pending-unexpired + accepted)
-- until revoked or expired, and forever once accepted.
ALTER TABLE "public"."iam_invites" ADD COLUMN "created_by" text;
CREATE INDEX "idx_iam_invites_creator" ON "public"."iam_invites" ("project_id", "environment", "created_by");

-- sqld:down
DROP INDEX "public"."idx_iam_invites_creator";
ALTER TABLE "public"."iam_invites" DROP COLUMN "created_by";
