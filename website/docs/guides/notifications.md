---
id: notifications
title: Email & SMS delivery
sidebar_label: Email & SMS
---

# Email & SMS delivery

Verification codes, magic links, OTPs, invites and password resets all leave
through a **provider** configured per project. Until one is configured and
enabled, those messages are dropped — sign-up with email verification will
appear to hang for the user, so this is the first thing to set up after creating
a project.

Delivery goes through the transactional outbox, so a message survives a provider
outage and is retried rather than lost.

## Email (SMTP)

`smtp` is the only supported type; any other value is refused.

```bash
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/email-providers \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{
    "type": "smtp",
    "enabled": true,
    "config": {
      "host": "smtp.example.com",
      "port": 587,
      "username": "postmaster@example.com",
      "password": "…",
      "from": "auth@example.com",
      "from_name": "Acme",
      "start_tls": true
    }
  }'
```

| Key | Default | Notes |
| --- | --- | --- |
| `host` | — | required |
| `port` | `587` | |
| `username` / `password` | — | `password` is encrypted at rest |
| `from` | — | envelope + header sender |
| `from_name` | — | display name |
| `secure` (alias `ssl`) | `false` | implicit TLS (port 465) |
| `start_tls` (alias `tls`) | on unless set | STARTTLS on a plaintext port |

Only one email provider is used: the first **enabled** one.

## SMS

Three types are implemented. Every one refuses a non-`https` endpoint, so a
credential is never sent over cleartext.

### `twilio`

```bash
curl -sX POST .../admin/sms-providers -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"twilio","enabled":true,
       "config":{"account_sid":"AC…","auth_token":"…","from":"+15550100"}}'
```

`account_sid`, `auth_token` and `from` are all required. The Messages endpoint
is derived from the SID; override it with `url` only if you proxy Twilio.

### `aws_sns`

```bash
curl -sX POST .../admin/sms-providers -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"aws_sns","enabled":true,
       "config":{"region":"eu-central-1","access_key_id":"…","secret_access_key":"…","from":"Acme"}}'
```

`endpoint` overrides the host for any SNS-compatible service (for example
`https://notifications.yandexcloud.net`). `from` becomes the sender id where the
carrier supports one.

### `generic`

Any HTTP endpoint you control. IAM POSTs JSON:

```json
{ "to": "+15550100", "text": "Your code is 123456", "from": "Acme" }
```

```bash
curl -sX POST .../admin/sms-providers -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"generic","enabled":true,
       "config":{"url":"https://sms.internal.example.com/send","auth_token":"…","from":"Acme"}}'
```

`url` is required. Credentials are optional: `auth_token` (or `api_key` /
`token` / `secret`) is sent as a bearer, `username` + `password` as basic auth.

### Send a test

```bash
curl -sX POST .../admin/sms-providers/send-test -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"to":"+15550100"}'
```

## Email templates

Eight built-in templates cover everything IAM sends. Each ships copy in `en` and
`ru`, and each can be overridden per project.

| Key | Sent when |
| --- | --- |
| `email_verification` | an address needs verifying |
| `otp` | an email sign-in code |
| `magic_link` | a passwordless sign-in link |
| `email_change` | confirming a new address |
| `password_reset` | a reset was requested |
| `mfa_email` | an MFA challenge over email |
| `flow_continue` | "continue on another device" for a resumable flow |
| `invite` | an invitation to sign up |

```bash
# what exists, built-in copy included
curl -s .../admin/email-templates -H "Authorization: Bearer $ADMIN_TOKEN"

# override the subject and body for one locale
curl -sX PATCH .../admin/email-templates/otp -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"locale":"en","subject":"Your Acme code","text":"Code: {{.code}}","html":"<p>Code: <b>{{.code}}</b></p>"}'

# render it without sending
curl -sX POST .../admin/email-templates/otp/preview -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"locale":"en"}'

# send it for real — the fastest way to prove SMTP works
curl -sX POST .../admin/email-templates/otp/send-test -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"to":"ops@example.com","locale":"en"}'
```

Bodies are Go templates. `{{.code}}` and `{{.link}}` are the variables the
notification layer supplies; `{{with .link}}…{{end}}` renders a block only when a
link exists.

## Which language a message uses

The locale is resolved in order: the request's locale → the user's stored
locale → the project's `auth.default_locale` → `en`. Set the project's locales
in the [`auth` document](/guides/admin-config); override individual strings per
locale under `admin/i18n/{locale}`.

## Testing without a real provider

In a non-`live` environment, `GET /v1/test/messages` returns the messages IAM
would have sent, so an end-to-end test can read the code it needs. See
[Test mode](/guides/test-mode).
