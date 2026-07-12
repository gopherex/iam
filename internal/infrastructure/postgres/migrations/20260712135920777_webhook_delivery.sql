-- sqld:up
CREATE TABLE "public"."iam_webhook_deliveries" (
  "id" text NOT NULL,
  "project_id" text NOT NULL,
  "environment" text NOT NULL DEFAULT 'live'::text,
  "webhook_id" text NOT NULL,
  "event_id" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending'::text,
  "attempt_count" int4 NOT NULL DEFAULT 0,
  "next_attempt_at" timestamptz,
  "last_attempt_at" timestamptz,
  "delivered_at" timestamptz,
  "response_status" int4,
  "response_body" text,
  "last_error" text,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "data" jsonb NOT NULL DEFAULT '{}'::jsonb,
  PRIMARY KEY ("id")
);
ALTER TABLE "public"."iam_events" ADD COLUMN "aggregate_id" text NOT NULL DEFAULT ''::text;
ALTER TABLE "public"."iam_events" ADD COLUMN "user_id" text NOT NULL DEFAULT ''::text;
ALTER TABLE "public"."iam_webhooks" ADD COLUMN "environment" text NOT NULL DEFAULT 'live'::text;
ALTER TABLE "public"."iam_webhooks" ADD COLUMN "idempotency_key" text NOT NULL DEFAULT ''::text;
ALTER TABLE "public"."iam_webhook_deliveries" ADD CONSTRAINT "iam_webhook_deliveries_attempt_count_check" CHECK (attempt_count >= 0);
ALTER TABLE "public"."iam_webhook_deliveries" ADD CONSTRAINT "iam_webhook_deliveries_status_check" CHECK (status = ANY (ARRAY['pending'::text, 'succeeded'::text, 'failed'::text]));
ALTER TABLE "public"."iam_webhook_deliveries" ADD CONSTRAINT "iam_webhook_deliveries_webhook_id_event_id_key" UNIQUE ("webhook_id", "event_id");
CREATE INDEX "idx_iam_events_project_created" ON "public"."iam_events" ("project_id", "environment", "created_at", "id");
CREATE INDEX "idx_iam_events_user_created" ON "public"."iam_events" ("project_id", "environment", "user_id", "created_at") WHERE (user_id <> ''::text);
CREATE INDEX "idx_iam_webhook_deliveries_project_created" ON "public"."iam_webhook_deliveries" ("project_id", "environment", "created_at", "id");
CREATE INDEX "idx_iam_webhook_deliveries_retry" ON "public"."iam_webhook_deliveries" ("status", "next_attempt_at") WHERE (status = ANY (ARRAY['pending'::text, 'failed'::text]));
CREATE INDEX "idx_iam_webhooks_project_env" ON "public"."iam_webhooks" ("project_id", "environment");
CREATE UNIQUE INDEX "idx_iam_webhooks_idempotency" ON "public"."iam_webhooks" ("project_id", "environment", "idempotency_key") WHERE (idempotency_key <> ''::text);

-- sqld:down
DROP INDEX "public"."idx_iam_webhooks_idempotency";
DROP INDEX "public"."idx_iam_webhooks_project_env";
DROP INDEX "public"."idx_iam_webhook_deliveries_retry";
DROP INDEX "public"."idx_iam_webhook_deliveries_project_created";
DROP INDEX "public"."idx_iam_events_user_created";
DROP INDEX "public"."idx_iam_events_project_created";
ALTER TABLE "public"."iam_webhook_deliveries" DROP CONSTRAINT "iam_webhook_deliveries_webhook_id_event_id_key";
ALTER TABLE "public"."iam_webhook_deliveries" DROP CONSTRAINT "iam_webhook_deliveries_status_check";
ALTER TABLE "public"."iam_webhook_deliveries" DROP CONSTRAINT "iam_webhook_deliveries_attempt_count_check";
ALTER TABLE "public"."iam_webhooks" DROP COLUMN "environment";
ALTER TABLE "public"."iam_webhooks" DROP COLUMN "idempotency_key";
ALTER TABLE "public"."iam_events" DROP COLUMN "user_id";
ALTER TABLE "public"."iam_events" DROP COLUMN "aggregate_id";
DROP TABLE "public"."iam_webhook_deliveries";
