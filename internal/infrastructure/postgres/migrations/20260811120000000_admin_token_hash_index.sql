-- sqld:up
-- Admin/operator token authentication looks up iam_admin_tokens by the sha256
-- of the presented token on every authenticated admin/mgmt request. Without an
-- index on hash this is a sequential scan over a table that only grows, so the
-- cost rises with every issued token. The existing (project_id) index cannot
-- serve an equality on hash.
CREATE INDEX "idx_iam_admin_tokens_hash" ON "public"."iam_admin_tokens" ("hash");

-- sqld:down
DROP INDEX "public"."idx_iam_admin_tokens_hash";
