-- sqld:up
-- The OIDC issuer is now an absolute URL derived from service.http.public_url
-- (OIDC Discovery 1.0 §3 requires the issuer to prefix the discovery document's
-- own URL). iam_environments.issuer used to be a free-form column that could
-- only ever hold the old relative "/p/<project>/e/<env>" form; the value is now
-- computed on read from the configured public base URL and never persisted.
-- Clear it so no stale relative issuer can survive the upgrade and be served
-- alongside the absolute one.
UPDATE "public"."iam_environments" SET "issuer" = NULL WHERE "issuer" IS NOT NULL;

-- sqld:down
-- The pre-upgrade value is not recoverable; the column simply stays empty.
SELECT 1;
