---
id: account-security
title: Account security & recovery
sidebar_label: Account security
---

# Account security & recovery

IAM detects account activity, groups incidents, delivers notifications and owns
protection and recovery state. Your application renders the available actions
through `iam.security.flow` in the existing TypeScript SDK. The embedded reference
application is available at `/security/{project_id}/{environment}`; project admins
configure the feature in **Account Security**.

## Policy and activation

The policy is separate for every project and environment. Read or replace it with
`GET` / `PUT /v1/projects/{project_id}/admin/security/policy`, using the administrative
bearer token and `X-Environment`. Read the current policy before replacing it.

| Setting | Default | Meaning |
| --- | --- | --- |
| `version` | `1` | Policy schema version, not an edit counter |
| `mode` | `disabled` | `disabled`, `observe`, or `enforce` |
| `continue_url` | empty | Application page handling security continuations |
| `client_urls` | `{}` | Optional registered client ID → application page |
| `notify` | `false` | Deliver detected-incident notifications in enforce mode |
| `require_new_device_proof` | `false` | Require additional proof before an unfamiliar AAL1 sign-in |
| `flow_ttl_seconds` | `1800` | Restricted flow lifetime |
| `trust_ttl_seconds` | `2592000` | Explicit device trust lifetime |
| `retention_days` | `90` | Incident, case, decision and delivery retention |
| `failure_threshold` | `5` | Failure count triggering an incident |
| `failure_window_seconds` | `900` | Failure counting window |
| `notification_cooldown_seconds` | `1800` | Coalescing and notification cooldown |

`disabled` preserves existing authentication behavior and refuses new security
flows. `observe` records detection without automated incident notifications or
new-device sign-in enforcement. Explicitly initiated review/recovery remains
available in observe mode. `enforce` enables the configured sign-in proof and
incident notifications. Verification-code and support deliveries are needed for
user-initiated recovery even when `notify` is false.

Configure SMTP and, for phone accounts, SMS through the existing
[notification providers](/guides/notifications). Set an application continuation
URL before enabling notifications or support recovery. URLs must use HTTPS, except
HTTP on localhost; credentials, query strings and fragments are rejected. IAM
chooses a configured client mapping when an authenticated client context exists,
otherwise the default URL. Request bodies cannot supply arbitrary redirects.
These continuation URLs are independent of **App Clients → Redirect URIs**, which
controls OAuth callbacks. A policy validation error identifies `continue_url` or
the invalid `client_urls` entry; enabling notifications requires a default URL.

The additive database migration runs on server startup. Finish updating all
replicas before enabling enforcement: an older server does not implement the new
account guard. Keep the policy disabled until the application and providers are
ready, then use observe mode to inspect activity before enabling enforcement.

## Detection and device recognition

Successful session creation is observed at the shared authentication boundary.
Signals include unfamiliar devices, changed IP addresses and recent password or
MFA failures. Token reuse, device mismatch, password/contact changes, MFA and
recovery-code changes, passkey changes and linked-identity changes also produce
incidents. Events distinguish successful actions from blocked attempts.

IP and user agent are context, not identity proof. There is no geolocation, ASN
lookup or impossible-travel detector. Configure trusted reverse proxies so the
observed IP actually belongs to the caller. Failure observations use a separate
transaction so a rejected authentication transaction does not erase evidence.
They are best effort if that separate write fails.

`registerDevice()` creates a random device capability and stores only its hash on
the server. The SDK sends it as `X-Device-Token`; recognition does not grant trust.
Explicit trust requires a separate, device-bound proof flow. Confirming “This was
me” does not trust a device. Device revocation revokes its actual sessions and
removes trust. All device operations are scoped to the signed-in account.

## Headless application integration

```ts
import { createIamClient } from '@gopherex/iam-sdk';

const iam = createIamClient({
  baseUrl: 'https://auth.example.com',
  clientId: 'prj_example',
  environment: 'live',
});

const unsubscribe = iam.security.flow.onChange((state, error) => {
  // Render state.step and state.next_actions; show errors and attempt limits.
  // Do not infer permission from a button or a previous response.
});
await iam.security.flow.resume();

// Invoke only after the user chooses to inspect the notification.
await iam.security.flow.continueFromURL(window.location.href);

// Or start from authenticated account activity.
const incidents = await iam.security.listIncidents({ limit: 30 });
await iam.security.flow.start({ incidentId: 'incident_id' });
await iam.security.flow.verifyCode(userEnteredCode);

// If required, select and prove a previously configured factor.
await iam.security.flow.submit('select_factor', { factor_id: selectedFactor });
await iam.security.flow.submit('verify_mfa', {
  factor_id: selectedFactor,
  code: userEnteredFactorCode,
});

// This value comes from an unchecked-by-default application checkbox.
await iam.security.flow.reportActivity({ allSessions: checkbox.checked });
await iam.security.flow.setPassword(userEnteredNewPassword);
// Recovery has finished. Offer ordinary sign-in; no session was issued.
unsubscribe();
```

The snippet illustrates successive actions; execute each only when advertised by
`next_actions`. Use `verifyPasskey()` after preparing a passkey proof, and
`registerRecoveryPasskey()` when the configured recovery method is passkey.
Passwordless projects advertise phone verification or `complete_recovery` as
appropriate for their enabled authentication methods.

Security continuations use the fragment `#iam_security=…`, avoiding transmission
in the page request. The SDK saves the continuation and a random exchange secret before removing the fragment.
After a lost response, reload or `resume()` retries that same exchange. A consumed
link alone cannot retrieve the result: retries require the same secret, expire
after 30 seconds and stop working once the flow advances.
A link is single-use and starts a restricted scenario; it is not ownership proof.
Opening a page alone never revokes sessions. Recovery capabilities are sent only
as `X-Security-Token`, separately from ordinary authorization.

The SDK scopes flow storage by server, project, environment and purpose. Security
flows use tab-local `sessionStorage`, falling back to memory when storage is
unavailable; `securityStorage` allows an application-specific adapter. It resumes
a pending flow after reload even after the normal session was revoked. Storage
failure can prevent resuming after reload, so keep the notification/support entry
point accessible. The new default ordinary-flow key is scoped too; in-flight flows
stored under the old `iam.flow` key must be restarted, or the application may
explicitly retain its legacy `storageKey` during migration.

Mutations rotate the capability and increment the version. An identical retry
with the previous capability is accepted for 30 seconds, allowing recovery from a
lost response. A changed request or stale version is rejected. Codes expire after
10 minutes, allow five attempts and have a one-minute resend delay. Account and
contact throttles apply across flows as well as per-flow attempt limits.

When unfamiliar-device enforcement returns `step_up_required`, the authentication
error contains `details.security_flow_token`. Pass it to
`iam.security.flow.resumeByToken(token)`. A successful `sign_in_approved` outcome
stores one-use proof in the SDK for the user's next ordinary sign-in; it does not
silently complete sign-in.

## Protection scope and ownership

Ownership is proved using the verified contact captured before the incident,
plus a previously configured factor when required, or an audited support grant.
Newly added factors cannot prove ownership of an older incident. Supported proof
includes TOTP, recovery codes, delivered factor codes and WebAuthn.

“This was not me” revokes the affected session by default. If the incident has no
associated session, the application must let the user choose one, or explicitly
choose all. Selecting all revokes every session and removes trust from every
device. The choice persists through password replacement, passwordless recovery
and escalation to support.

Protection also installs an account guard that prevents fresh sign-ins and
sensitive credential changes while recovery is pending. It remains even if the
browser flow expires or is closed; support recovery can take over that guard.
After recovery, IAM persistently invalidates older ownership and sign-in proofs,
including across later profile or contact changes. IAM restores the previously verified contacts, removes credentials
added after the incident and releases the guard. Support-authorized recovery
replaces the old proof methods and sets the separately verified contact reviewed
by support. Recovery never issues an access or refresh token.

An unrelated session survives selective protection, including subsequent password
replacement. Its user may still have account access once recovery finishes; the
checkbox determines this deliberately. Remote token validation observes session
revocation immediately. Offline JWT validation cannot consult IAM and retains its
normal token-expiry window; see [Sessions](/concepts/sessions).

## Recovery through support

`iam.security.flow.recover({ identifier, contact })` starts a public request using
an email or phone account identifier and an available email for contact. The
contact must be verified before opening the case. Verifying that new address
proves only that support can reach the requester. It does not prove account
ownership, reveal account existence, revoke sessions or issue a session.

The application displays the case number, status and user/support messages.
`support_message` adds information. A case survives browser-flow expiry; starting
again with the same identifier and verified contact resumes an open case. Repeated
requests are throttled.

Project administrators list and inspect cases under
`/v1/projects/{project_id}/admin/security/cases`. A decision additionally requires
`security:recovery`, or an operator principal. Decision actions are `approve`,
`reject` and `request_information`; every decision requires a user-visible message
and verification evidence. IAM records the actor's stable credential ID and stores
evidence encrypted. Support must establish ownership through its own reviewed
procedure; account details or a newly verified email alone are insufficient.

Approval sends a single-use grant to the verified contact. It authorizes only
protection and restoration of the case's account. A replacement grant supersedes
previous exchanged grants, and a completed case cannot restore access again.
Rejection and requests for information notify the same contact. No administrator
password, session impersonation or unrestricted login link is sent.

## Delivery, observability and retention

Delivery intent and the sanitized outbox event are committed with the security
operation. The worker loads the encrypted payload by delivery ID and routes it to
SMTP or SMS. Codes, full contacts and capabilities stay out of public incident and
outbox payloads. The existing encryption key is required to read pending jobs.

The delivery list shows `queued`, `accepted`, `failed`, `blocked` or `expired`,
attempt count, masked contact and a sanitized error code. `accepted` means the
provider accepted the request, not that the recipient received or read it.
A missing provider is `blocked`; one channel does not block the other. Admins can
retry eligible deliveries after fixing configuration. Informational notifications,
including scheduled/cancelled deletion, have no proof expiry. Expired proof must be
reissued through the flow. Transport retries can produce duplicate messages if a
provider accepted a message just before the worker lost its response.

Retention removes closed incidents and completed/rejected cases after their last
update, including encrypted private fields. Open cases, unresolved incidents,
active recovery dependencies and their related decisions/deliveries remain available. Stale devices with no associated sessions are also
removed. Expired continuation tokens and attempt buckets are cleaned separately.
Active account guards have no retention expiry: completing authorized recovery is
required to release them. Lists use bounded, cursor-based pagination.

## Pending deletion during recovery

After ownership is proved, the restricted flow exposes `deletion` and advertises
`cancel_deletion` while the fixed deadline is still in the future. Render the
remaining time and an explicit cancel button. Call
`iam.security.flow.cancelDeletion(state.deletion.request_id)` only on that choice.
Cancellation records the user's action, sends the cancellation notification and
preserves the recovery guard and the selected session-revocation scope. It requires
no ordinary session. A contact-only support request cannot view or cancel deletion.
Recovery does not automatically stop the timer. After the deadline cancellation
is rejected, including during recovery; finish recovery or cancel beforehand.
