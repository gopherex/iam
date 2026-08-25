---
id: go
title: Go SDK
sidebar_label: Go (resource server)
---

# Go SDK reference

`github.com/gopherex/iam/pkg/sdk` is for **resource servers** — the services
behind IAM that have to authenticate the bearer tokens IAM issued. It verifies
tokens and hands your handler a principal; it does not sign users in. That is the
job of the [TypeScript SDK](/sdk/typescript) or the
[REST API](/rest-api/overview).

## Wiring

Load `AuthenticatorConfig` the way the IAM service loads its own config
(`mapstructure`, `default`, `validate` tags), then build one authenticator and
reuse it:

```go
auth, err := sdk.NewAuthenticator(sdk.AuthenticatorConfig{
	Mode:        sdk.ValidationModeHybrid,
	BaseURL:     "https://auth.example.com",
	Credential:  "service-token",
	ProjectID:   "prj_7Fk2",
	Environment: "live",
	Audience:    "api",
})
if err != nil {
	return err
}
if err := sdk.Warm(ctx, auth); err != nil {
	return err
}
```

`Warm` fetches the JWKS up front, so the first real request does not pay for it.

## Validation modes

| Mode | How a token is checked |
| --- | --- |
| `remote` | every request calls IAM `/v1/tokens/verify` |
| `local` | JWTs are verified in-process against the published JWKS |
| `hybrid` | local first, falling back to remote when the local path cannot authenticate the token |

## HTTP

```go
handler := sdk.HTTPMiddleware(auth, protectedHandler)
```

Inside handlers:

```go
principal, ok := sdk.PrincipalFrom(r.Context())
```

The principal carries what the token asserts, with helpers for the checks a
resource server actually makes:

```go
principal.HasScope("billing:read")
principal.MeetsAAL(2)              // the session stepped up / used MFA
principal.InGroup("ops")           // an IAM role, from the `groups` claim
principal.InAnyGroup("ops", "sre")
```

`Groups` is present only when the token was issued with the **`groups` scope** —
IAM does not put roles in every token. Request the scope from the client, and
assign the roles with
`PUT /v1/projects/{project_id}/admin/users/{user_id}/roles`. The values are the
ones an admin assigned; a client can ask for the scope, never for a role. See
[Roles in the token](/concepts/oidc-federation).

## Revocation and the validation mode

IAM can revoke an access token before it expires — `/oauth2/revoke`, ending the
session, or reuse detection on a refresh token all put it on a denylist. Whether
this SDK notices depends on the mode:

| Mode | Sees a revocation |
| --- | --- |
| `remote` | yes — every call asks IAM, which checks the denylist |
| `hybrid` | no — local verification succeeds, so it never falls through to remote |
| `local` | no — the process never asks IAM anything |

Local verification is offline by design; that is what makes it fast, and it is
why a revoked token keeps verifying until it expires on its own. The exposure is
one access-token lifetime — the project's `session_policy.access_ttl`, 10 minutes
by default. Use `remote` where a revocation must take effect at once (admin
consoles, anything money moves through), and shorten `access_ttl` where `local`
is worth its latency.

## gRPC

The interceptors live in a separate package, so services that do not speak gRPC
do not import it:

```go
server := grpc.NewServer(
	grpc.UnaryInterceptor(iamsdkgrpc.UnaryServerInterceptor(auth)),
	grpc.StreamInterceptor(iamsdkgrpc.StreamServerInterceptor(auth)),
)
```

## Webhooks

Build a verifier from the endpoint's signing secret and check a delivery before
you act on it:

```go
verifier, err := sdk.NewWebhookVerifier(sdk.WebhookVerifierConfig{SigningSecret: secret})
event, err := verifier.Verify(r.Header, body)
```

`Verify` rejects a stale timestamp as well as a bad signature, so a captured
delivery cannot be replayed later. See
[Webhooks & hooks](/concepts/webhooks-hooks).
