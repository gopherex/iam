-- sqld:up
ALTER TABLE iam_users ADD COLUMN security_recovered_at timestamptz;
UPDATE iam_users SET security_recovered_at=(data->>'security_recovered_at')::timestamptz
WHERE data->>'security_recovered_at' IS NOT NULL;
ALTER TABLE iam_security_continuations ADD COLUMN exchange_data text NOT NULL DEFAULT '';

-- sqld:down
UPDATE iam_users SET data=jsonb_set(data,'{security_recovered_at}',to_jsonb(security_recovered_at))
WHERE security_recovered_at IS NOT NULL;
ALTER TABLE iam_security_continuations DROP COLUMN exchange_data;
ALTER TABLE iam_users DROP COLUMN security_recovered_at;
