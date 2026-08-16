---
id: quickstart
title: Quickstart
sidebar_position: 2
---

# Quickstart

Run IAM locally, create a project, and complete your first sign-in — in a few
minutes.

## 1. Run the server

The repo ships a full local stack (Postgres + the IAM server + the embedded
admin panel):

```bash
git clone https://github.com/gopherex/iam.git
cd iam
docker compose up
```

This brings up:

- **Postgres** (`postgres:17`, on `127.0.0.1:5436`).
- **IAM** on **http://localhost:8080**, which applies database migrations
  automatically and seeds a **`Root`** project on first boot.

Open **http://localhost:8080** and sign in to the admin panel with the dev
master key:

```
dev
```

:::warning DEV ONLY
`docker compose up` uses a hard-coded master key (`dev`) and a committed
development AES encryption key. **Never** use these outside local development —
see [Self-hosting → Configuration](/self-hosting/configuration).
:::

## 2. Create a project and an admin token

Runtime auth calls are scoped to a **project** (identified by the `X-Client-Id`
header). Create one with the **operator** master key, then mint a
**project-admin** token:

```bash
# Create a project (operator / master key)
curl -sX POST http://localhost:8080/mgmt/v1/projects \
  -H "Authorization: Bearer dev" \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme","slug":"acme","default_locale":"en"}'
# -> 201 {"project":{"id":"prj_XXXX","name":"Acme",...}}

# Mint a project-admin token (shown once)
curl -sX POST http://localhost:8080/mgmt/v1/projects/prj_XXXX/admin-tokens \
  -H "Authorization: Bearer dev" \
  -H "Content-Type: application/json" \
  -d '{"name":"local-admin"}'
# -> 200 {"admin_token":"<shown once>","expires_at":"..."}
```

The `project id` (`prj_XXXX`) is your `X-Client-Id` for runtime calls. See
[Principals & credentials](/concepts/principals) for who uses which token.

## 3. Sign a user up and in

The recommended path is the [resumable auth flow](/rest-api/flows) — one
endpoint drives signup, the emailed code, and the resulting session.

```bash
# Start a signup flow (server issues an email verification challenge)
curl -sX POST http://localhost:8080/v1/auth/flows \
  -H "X-Client-Id: prj_XXXX" \
  -H "Content-Type: application/json" \
  -d '{"kind":"signup","email":"ada@example.com","password":"hunter2pw","name":"Ada"}'
```
```json
{
  "flow_token": "ftk_9s2...",
  "kind": "signup", "status": "pending", "step": "verify_email",
  "next_actions": ["submit_code", "resend"],
  "contact": { "email_masked": "a***@example.com" },
  "challenge": { "channel": "email", "attempts_left": 5 }
}
```

Locally there's no real mailbox — read the emitted code from the **test inbox**
(requires an admin token and a non-live environment):

```bash
curl -s "http://localhost:8080/v1/test/messages?to=ada@example.com" \
  -H "Authorization: Bearer <admin_token>" -H "X-Environment: test"
```

Submit the code to complete signup and receive a session:

```bash
curl -sX POST http://localhost:8080/v1/auth/flows/ftk_9s2.../submit \
  -H "X-Client-Id: prj_XXXX" \
  -H "Content-Type: application/json" \
  -d '{"action":"submit_code","payload":{"code":"123456"}}'
```
```json
{
  "status": "completed", "step": "completed",
  "session": { "access_token": "eyJ...", "refresh_token": "rt_...",
               "expires_in": 1800, "token_type": "Bearer" }
}
```

Use the access token as a bearer credential:

```bash
curl -s http://localhost:8080/v1/users/me \
  -H "X-Client-Id: prj_XXXX" -H "Authorization: Bearer eyJ..."
```

## 4. Or use the SDK

For a web app, the [TypeScript SDK](/guides/sdk-quickstart) wraps all of the
above:

```ts
import { createIamClient } from '@gopherex/iam-sdk';

const iam = createIamClient({ baseUrl: 'http://localhost:8080', clientId: 'prj_XXXX' });

await iam.flow.start({ kind: 'signup', email: 'ada@example.com', password: 'hunter2pw', name: 'Ada' });
await iam.flow.verifyEmail({ code: '123456' }); // completes → SIGNED_IN
```

## Next steps

- **[Concepts](/concepts/overview)** — projects, environments, sessions, MFA.
- **[SDK quickstart](/guides/sdk-quickstart)** — full web-app integration.
- **[Auth flows](/guides/auth-flows)** — signup / signin / recovery in depth.
- **[Self-hosting](/self-hosting/docker)** — configure and deploy for real.
