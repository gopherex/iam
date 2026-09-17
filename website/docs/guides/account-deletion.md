---
title: Account deletion
---

Self-service deletion has a **7-day default cancellation period**. Project admins
can configure 1–365 days independently for each environment under **Config →
Account deletion**. Deletion works independently of the account-security detection
mode. Changing this setting affects new requests only.

During the cancellation period the user retains full access, including existing
sessions, new logins and profile updates. Login and activity neither cancel the
request nor move its deadline. Repeating a pending request returns the same request
ID and deadline. Cancellation is explicit and requires identity confirmation.

## SDK and proof

```ts
const { data: status } = await iam.account.deletion.get();
const result = await iam.account.deletion.request({ password });
// result.data contains request_id, status, requested_at, delete_at, grace_days.
// Keep the ordinary session and show the exact deletion date.

const cancelled = await iam.account.deletion.cancel({ password });
```

`iam.account.deleteAccount({ password })` is an alias for scheduling. It no longer
deletes immediately or clears local credentials. Upgrade callers that previously
assumed immediate deletion. `DELETE /v1/users/me` now requires a JSON body and
returns the deletion state (including `ok: true`). Read state with
`GET /v1/users/me/deletion`; cancel with `POST /v1/users/me/deletion/cancel`.

Password accounts must supply the current password; attempts are rate limited.
With MFA, or without a password, IAM also requires an isolated proof flow. On
`step_up_required`, use `error.details.security_flow_token`:

```ts
await iam.security.flow.resumeByToken(String(result.error!.details!.security_flow_token));
// Render currentState.step and next_actions using your application's UI.
// Submit contact/MFA/passkey proof using iam.security.flow.
// When authorize_deletion is offered, require explicit confirmation:
const authorized = await iam.security.flow.submit('authorize_deletion');
const scheduled = await iam.account.deletion.request({
  password,
  proofToken: authorized.state!.flow_token,
});
```

Use the same sequence with `deletion.cancel` for cancellation. A proof expires
after five minutes and is bound to the account, environment, initiating session
and operation. A request proof cannot cancel a deletion. Recovery through support
does not directly authorize deletion: finish recovery, sign in and confirm anew.
Proof completion does not create an ordinary session or itself schedule deletion.

The runnable reference application at `/security/{projectId}/{environment}`
demonstrates password confirmation, proof steps, the deadline and cancellation.

## Deadline and cleanup

At `delete_at`, IAM rejects authenticated access and new authentication even if
cleanup is delayed. Cancellation is accepted only before that instant. A durable
worker checks pending requests every minute and on startup. Each account is
finalized transactionally; failures roll back and are retried. Concurrent workers
and cancellation serialize on the account record.

Cleanup removes the IAM user, sessions, refresh tokens, credentials, identities,
MFA/passkeys, grants, consents, roles, account-bound flows and recovery data.
The minimal deletion record and audit/event history remain subject to the existing
retention rules; deletion does not erase backups or application-owned data.
Administrative immediate deletion remains a separate privileged operation.

Applications validating JWTs locally must enforce revocation through their normal
revocation integration or IAM session validation. Offline signature validation
alone cannot discover a new deletion deadline before the token expires.

## Notifications and application data

IAM queues `account_deletion` notifications on scheduling and cancellation, to the
verified email or, when absent, verified phone. Delivery uses the configured
notification provider and retry queue. The message includes the fixed deadline;
its optional application link uses the account-security `continue_url` setting.
Opening the link does not sign in or cancel anything. With no verified contact,
the application must show the status obtained from the API.

Subscribe to the public [webhook events](/concepts/webhooks-hooks):

| Event | Application action |
| --- | --- |
| `user.deletion_scheduled` | Show the deadline; preserve access and data |
| `user.deletion_cancelled` | Remove the pending-deletion indication |
| `user.deleted` | Finalize application-owned cleanup idempotently |

Scheduled/cancelled events contain `user_id`, `request_id` and `delete_at` in the
versioned project/environment envelope. Do not delete application data merely on
the scheduling event. Follow webhook signature verification and retry handling.

## Administration

The user list marks scheduled deletion and its date. **User → deletion** shows
request time, fixed deadline, time remaining, cancellation and recent history.
**Cancel scheduled deletion** lets an administrator cancel the specific request
before its deadline, with a required reason. The history records the administrator
and reason; the user receives the cancellation notification. Cancellation keeps
existing bans and account-security restrictions. It cannot restore an account
whose deadline has passed. A stale dialog cannot cancel a newer request.

`POST /v1/projects/{project_id}/admin/users/{user_id}/deletion/cancel` accepts
`{ "request_id": "…", "reason": "…" }` with project-admin or operator credentials.
Repeating cancellation of the same request is idempotent.

Admin API endpoints are `GET/PUT /v1/projects/{project_id}/admin/account-deletion-policy`
and `GET /v1/projects/{project_id}/admin/users/{user_id}/deletion`, scoped by
`X-Environment`. Policy updates require project-admin or operator authorization.


## Reusing an email address

Within the same project and environment, the address remains occupied while
self-service deletion is pending, including the interval after the deadline and
before the cleanup transaction commits. Registration follows the usual existing
account response and does not cancel deletion or create a duplicate account.

After final cleanup, registration at the same address creates a **new user ID**.
The new account follows the project's normal registration and verification rules;
it does not inherit email verification, passwords, roles, sessions or deletion
history from the old account. A delayed retry of the old deletion targets only
the old user ID.

Application data and cleanup handlers must identify the owner by project,
environment and immutable user ID. Never use email alone to reconnect historical
data or to process `user.deleted`: the address may already belong to a different
account. If an application intentionally supports transferring data to a new
account, that requires a separate verified process.

## Recovery and token validation

Verified account recovery displays a pending deletion and allows the user to
cancel it explicitly through `iam.security.flow.cancelDeletion(requestId)` before
the deadline, even after revocation of the normal session. Recovery does not
implicitly cancel or extend deletion. Unverified support requests confer no such
permission. See [account recovery](/guides/account-security).

The deadline is enforced by bearer validation, token introspection and OIDC token
issuance before the cleanup worker runs. Applications validating JWTs offline
still need their own revocation/deadline strategy or online IAM validation.
Deletion notifications carry no expiring proof and can be retried after provider
recovery without the 30-minute limit of security continuations.
