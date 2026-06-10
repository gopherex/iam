-- sqld:up
CREATE TABLE "public"."iam_flows" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
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
ALTER TABLE "public"."iam_flows" ADD CONSTRAINT "iam_flows_token_hash_key" UNIQUE ("token_hash");
CREATE INDEX "iam_flows_project_idx" ON "public"."iam_flows" ("project_id");

-- sqld:down
DROP INDEX "public"."iam_flows_project_idx";
ALTER TABLE "public"."iam_flows" DROP CONSTRAINT "iam_flows_token_hash_key";
DROP TABLE "public"."iam_flows";
