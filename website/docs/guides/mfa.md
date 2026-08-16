---
id: mfa
title: Multi-factor authentication
sidebar_label: MFA
---

# Multi-factor authentication

IAM supports TOTP, SMS, email and WebAuthn factors plus recovery codes. There
are two distinct phases: **enrollment** (adding a factor, on a signed-in
account) and **challenge at login** (stepping up an authenticated session to
AAL2). See [Sessions & AAL](/concepts/sessions) for the assurance-level model.

## Enroll TOTP (authenticator app)

```ts
const { data } = await iam.mfa.enrollTotp({ name: 'Authenticator' });
// data: { factorId, secret, otpauthUrl, qrSvg }
showQr(data.qrSvg);           // render the SVG, or build a QR from otpauthUrl

// user scans, then confirms a code to activate the factor
await iam.mfa.verifyTotp({ factorId: data.factorId, code });
```

## Enroll email / SMS factors

```ts
const { data } = await iam.mfa.enrollEmail('ada@example.com'); // { factorId, challengeId }
// user receives a code, verify via the same challenge:
await iam.auth.verifyMfa({ challengeId: data.challengeId, code });

await iam.mfa.enrollSms('+15551234567');
```

## Recovery codes

```ts
const { data } = await iam.mfa.generateRecoveryCodes({ regenerate: true });
showOnce(data.codes); // display exactly once — they are not retrievable later
```

## Challenge at login (step-up)

When `signInWithPassword` returns `nextStep: 'mfa_required'`, it also returns a
`flowToken` and the eligible `factors`:

```ts
const { data } = await iam.auth.signInWithPassword({ email, password });

if (data.nextStep === 'mfa_required') {
  const factor = data.factors[0];

  // for TOTP the user already has a code; for sms/email trigger a challenge:
  await iam.auth.challengeMfa({ flowToken: data.flowToken, factorId: factor.id });

  const res = await iam.auth.verifyMfa({ flowToken: data.flowToken, code });
  // res.data.session is now AAL2
}
```

Using the [flow controller](/guides/auth-flows)? The `mfa_required` step is
handled by `iam.flow.submit('verify_mfa', { code })`.

## Step-up an already-active session

For sensitive operations (delete account, change email), require a fresh factor:

```ts
await iam.auth.stepUp({ purpose: 'delete_account', requiredAal: 2 });
```

## Managing factors

```ts
await iam.mfa.listFactors();
await iam.mfa.removeFactor(factorId);
```

## WebAuthn as a second factor

Passkeys can act as an MFA factor too — see the WebAuthn methods in the
[SDK reference](/sdk/typescript#iamwebauthn). Enroll with
`/v1/auth/mfa/webauthn/enroll/{options,verify}`.

## Errors

| `error.code` | Meaning |
| --- | --- |
| `mfa_required` | session must step up before proceeding |
| `mfa_invalid` | wrong code / assertion |
| `mfa_factor_not_allowed` | factor type disabled by policy |
| `step_up_required` | operation needs a higher AAL than the session has |
