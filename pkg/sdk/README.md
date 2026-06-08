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

## gRPC

Use the separate package so non-gRPC users do not import gRPC APIs:

```go
server := grpc.NewServer(
	grpc.UnaryInterceptor(iamsdkgrpc.UnaryServerInterceptor(auth)),
	grpc.StreamInterceptor(iamsdkgrpc.StreamServerInterceptor(auth)),
)
```
