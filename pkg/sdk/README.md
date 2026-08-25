# Go Server SDK

`pkg/sdk` is for resource servers that need to authenticate IAM bearer tokens.

## Wiring

Load `AuthenticatorConfig` with the same config loader style used by the IAM
service (`mapstructure`, `default`, `validate` tags), then build one
authenticator:

```go
auth, err := sdk.NewAuthenticator(sdk.AuthenticatorConfig{
	Mode:        sdk.ValidationModeHybrid,
	BaseURL:     "https://iam.example.com",
	Credential:  "service-token",
	ProjectID:   "proj_123",
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

`remote` calls IAM `/v1/tokens/verify` for every request. `local` verifies JWTs
in-process from public JWKS. `hybrid` verifies locally first and falls back to
remote verification when the local path cannot authenticate the token.

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
IAM does not put roles in every token. Request it from the client, and assign the
roles under
`PUT /v1/projects/{project_id}/admin/users/{user_id}/roles`. The values are the
ones an admin assigned; a client can ask for the scope, never for a role.

## gRPC

Use the separate package so non-gRPC users do not import gRPC APIs:

```go
server := grpc.NewServer(
	grpc.UnaryInterceptor(iamsdkgrpc.UnaryServerInterceptor(auth)),
	grpc.StreamInterceptor(iamsdkgrpc.StreamServerInterceptor(auth)),
)
```
