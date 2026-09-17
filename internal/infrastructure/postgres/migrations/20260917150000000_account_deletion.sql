-- sqld:up
CREATE TABLE iam_account_deletions (
 project_id text NOT NULL,
 environment text NOT NULL,
 user_id text NOT NULL,
 request_id text NOT NULL,
 status text NOT NULL,
 delete_at timestamptz NOT NULL,
 data jsonb NOT NULL,
 PRIMARY KEY(project_id,environment,user_id)
);
CREATE INDEX idx_account_deletions_due ON iam_account_deletions(delete_at) WHERE status='pending';


-- The users JSON envelope is read by existing profile/admin APIs. Preserve its
-- authoritative deletion state when a concurrent profile/auth update carries
-- an older account snapshot.
CREATE FUNCTION iam_preserve_account_deletion() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE deletion_data jsonb;
BEGIN
 SELECT data INTO deletion_data FROM iam_account_deletions
 WHERE project_id=NEW.project_id AND environment=NEW.environment AND user_id=NEW.id;
 IF deletion_data IS NOT NULL THEN
  NEW.data := jsonb_set(NEW.data, '{deletion}', deletion_data);
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER iam_users_preserve_deletion BEFORE UPDATE OF data ON iam_users
FOR EACH ROW EXECUTE FUNCTION iam_preserve_account_deletion();

-- sqld:down
DROP TRIGGER iam_users_preserve_deletion ON iam_users;
DROP FUNCTION iam_preserve_account_deletion();
DROP TABLE iam_account_deletions;
