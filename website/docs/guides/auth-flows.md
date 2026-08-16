---
id: auth-flows
title: Building with auth flows
sidebar_label: Auth flows
---

# Building with auth flows

The `FlowController` (`iam.flow`) drives the server-side
[resumable flow state machine](/rest-api/flows). Your UI becomes a thin
renderer of `FlowState.step` — the server owns every transition, so the flow
survives reloads, tab switches and cross-device continuation.

## The loop

```ts
import { iam } from './iam';

// 1. subscribe — the single source of truth for your UI
iam.flow.onChange((state, error) => {
  switch (state.step) {
    case 'collect_credentials': return renderCredentials();
    case 'verify_email':        return renderCodeInput(state.contact.email_masked);
    case 'set_password':        return renderNewPassword();
    case 'mfa_required':        return renderMfa(state.factors);
    case 'accept_consents':     return renderConsents(state.consents_required);
    case 'completed':           return; // SDK emits SIGNED_IN
  }
  if (error) toast(error.message); // transient — flow stays pending
});

// 2. start
await iam.flow.start({ kind: 'signup', email, password, name });
```

## Handling each step

```ts
// verify_email / verify_phone
await iam.flow.verifyEmail({ code });          // or { token } from a link
// set_password (recovery)
await iam.flow.setPassword({ password });
// mfa_required
await iam.flow.submit('verify_mfa', { code });
// accept_consents
await iam.flow.submit('accept_consents', { consents: [{ key, version }] });
// resend a challenge (guarded by resend_at → 429 flow_resend_too_soon)
await iam.flow.resend();
```

## Resuming

```ts
// on app boot — cookie first, then persisted localStorage token
const state = await iam.flow.resume();
if (state && state.status === 'pending') renderStep(state);

// from a deep link (email "continue on this device")
await iam.flow.resumeByToken(tokenFromUrl);
```

## Abandoning

```ts
await iam.flow.abandon(); // DELETE the flow, clears local + cookie state
```

## Flow kinds

| Kind | Path |
| --- | --- |
| `signup` | `collect_credentials → verify_email → [accept_consents] → completed` |
| `signin` | `collect_credentials → [mfa_required] → completed` |
| `recovery` | `collect_credentials → verify_email → set_password → completed` |
| `email_change` | **not a flow** — use `iam.account.changeEmailStart/Verify` |

:::tip Why flows over raw calls
With flows you never manage `challenge_id`, `flow_token` rotation, or step
ordering yourself — the server returns `next_actions[]` and you render them. Use
the [raw runtime endpoints](/rest-api/runtime) only for bespoke UX.
:::
