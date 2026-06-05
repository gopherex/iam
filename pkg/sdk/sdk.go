// Package sdk is the ergonomic Go SDK for IAM. It wraps the generated client
// in pkg/api with convenience constructors, auth helpers and typed errors, so
// downstream Go services integrate by importing github.com/gopherex/iam/pkg/sdk.
//
// The concrete surface is filled in once pkg/api is generated and the client
// stack is fixed.
package sdk
