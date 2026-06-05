CREATE TABLE "public"."iam_sessions" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "user_id" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE TABLE "public"."iam_users" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "primary_email" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE INDEX "idx_iam_sessions_project" ON "public"."iam_sessions" ("project_id");
CREATE INDEX "idx_iam_sessions_user" ON "public"."iam_sessions" ("user_id");
CREATE INDEX "idx_iam_users_project" ON "public"."iam_users" ("project_id");
CREATE UNIQUE INDEX "uq_iam_users_project_email" ON "public"."iam_users" ("project_id", "primary_email") WHERE (primary_email IS NOT NULL);

