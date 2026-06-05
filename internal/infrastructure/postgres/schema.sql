-- Authoritative schema for the IAM Postgres store. sqld reads this to generate
-- gen/db (typed query funcs), gen/bob (bob models) and the bootstrap migration.
--
-- Storage model (mirrors the komeet/stroppy pattern): each record type is one
-- table carrying the queryable envelope columns (id, project_id, created_at,
-- updated_at, plus any secondary keys) and the full domain object in a
-- `data jsonb` column. IAM is project-scoped, so project_id is the partition
-- key. These two tables are a representative seed; real IAM tables are added as
-- the domain is designed.

-- ===== iam_users =====
CREATE TABLE iam_users (
  id            text PRIMARY KEY,
  project_id    text NOT NULL,
  primary_email text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  data          jsonb NOT NULL
);
CREATE INDEX idx_iam_users_project ON iam_users (project_id);
CREATE UNIQUE INDEX uq_iam_users_project_email ON iam_users (project_id, primary_email)
  WHERE primary_email IS NOT NULL;

-- ===== iam_sessions =====
CREATE TABLE iam_sessions (
  id         text PRIMARY KEY,
  project_id text NOT NULL,
  user_id    text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  data       jsonb NOT NULL
);
CREATE INDEX idx_iam_sessions_project ON iam_sessions (project_id);
CREATE INDEX idx_iam_sessions_user ON iam_sessions (user_id);
