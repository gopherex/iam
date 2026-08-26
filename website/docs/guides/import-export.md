---
id: import-export
title: Import, export & jobs
sidebar_label: Import & export
---

# Import, export & jobs

Bulk work runs as a **job**: the request returns a `job_id` immediately and a
background worker drains it, so a million-row import does not sit on an HTTP
connection. Jobs are picked up by whichever replica takes the lease.

## Migrating users from another IdP

The move is two steps: prove your existing password hashes will still verify,
then import them.

### 1. Check a hash before you commit

```bash
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/import/password-hashes/verify \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"format":"bcrypt","hash":"$2a$12$…","password":"the-known-plaintext"}'
# -> { "valid": true }
```

**bcrypt is the only format IAM verifies.** A hash in any other scheme
(`argon2`, `scrypt`, `pbkdf2`, `md5`, a vendor-specific format) cannot be
imported — importing it would store a credential nobody can ever sign in with.
For those, import the users without a hash and run a password reset, or keep the
old system alongside and let people re-set their password on first sign-in.

### 2. Import

```bash
curl -sX POST .../admin/import/users -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "password_hash_format": "bcrypt",
    "users": [
      {"email":"ada@example.com","name":"Ada","password_hash":"$2a$12$…"},
      {"email":"bob@example.com","name":"Bob"}
    ]
  }'
# -> { "job_id": "job_imp_1", "status": "pending" }
```

Up to 1000 users per request; send several batches for more. A row that fails —
a bad address, a duplicate email, a non-bcrypt hash — is **counted, not fatal**:
the job finishes and reports per-row tallies, so one bad record does not lose the
batch.

Imported users are created active with no email verification sent. A user
imported without `password_hash` has no password: give them one with
`POST admin/users/{id}/password`, or send them through password recovery.

`send_invites` is accepted in the request body but does not currently send
anything; use [invites](/guides/admin-config) explicitly if you want that.

## Watching a job

```bash
curl -s .../admin/jobs                 -H "Authorization: Bearer $ADMIN_TOKEN"
curl -s .../admin/jobs/job_imp_1       -H "Authorization: Bearer $ADMIN_TOKEN"
curl -sX POST .../admin/jobs/job_imp_1/cancel -H "Authorization: Bearer $ADMIN_TOKEN"
```

```json
{
  "id": "job_imp_1",
  "type": "import_users",
  "status": "completed",
  "progress": { "total": 2, "processed": 2, "failed": 0 },
  "report_url": null
}
```

`status` is `running`, `completed`, `failed` or `cancelled`. A failed job carries
the reason, and an import carries the first hundred per-row errors.

## Audit export

```bash
curl -sX POST .../admin/audit/export -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"from":"2026-01-01T00:00:00Z","to":"2026-02-01T00:00:00Z","format":"json"}'
# -> { "job_id": "job_aud_9", "status": "pending" }

curl -s .../admin/exports/job_aud_9 -H "Authorization: Bearer $ADMIN_TOKEN"
# -> { "status": "completed", "download_url": "data:application/json;base64,…" }
```

The export is JSON and is returned **inline as a `data:` URL**, not as a link to
object storage — decode it client-side. It is capped at 50 000 rows, so narrow
the window rather than exporting a year at once.

## Subject data export

Two endpoints ask for the same thing: an administrator answering a
subject-access request, and the person themselves.

```bash
# administrator
curl -sX POST .../admin/users/usr_9/export -H "Authorization: Bearer $ADMIN_TOKEN"
# -> { "job_id": "job_exp_3" }
curl -s  .../admin/exports/job_exp_3       -H "Authorization: Bearer $ADMIN_TOKEN"

# the person, on their own session
curl -sX POST https://auth.example.com/v1/users/me/export -H "Authorization: Bearer $ACCESS"
curl -s  https://auth.example.com/v1/users/me/export/job_exp_4 -H "Authorization: Bearer $ACCESS"
```

The document contains the profile, the linked identities, the sessions, the
OAuth grants and the security activity — up to 1000 rows per collection.

It deliberately **excludes credential material**: a password hash, a TOTP seed
and a refresh token are facts about the account rather than about the person, and
including them would turn an access request into a credential leak.

Like the audit export, it comes back inline as a `data:` URL rather than as a
link to object storage.

## Config as code

Project configuration moves separately from data, through the operator plane's
`config:export` / `config:plan` / `config:apply`, or the project-admin
`GET/PUT admin/config`. See [Admin & config](/guides/admin-config).
