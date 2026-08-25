-- sqld:up
-- Registration access tokens (RFC 7592) are looked up by their digest on every
-- call a client makes to manage itself. The digest lives in the client's data
-- envelope, so without this index the lookup is a sequential scan over every
-- client of every project.
CREATE INDEX "idx_iam_app_clients_registration_token"
    ON "public"."iam_app_clients" ((data ->> 'RegistrationTokenHash'))
 WHERE data ->> 'RegistrationTokenHash' IS NOT NULL;

-- sqld:down
DROP INDEX "public"."idx_iam_app_clients_registration_token";
