// Package api is the IAM HTTP API generated from openapi/openapi.yaml by ogen
// (https://ogen.dev). It holds the request/response types, client and server
// interfaces so any consumer can talk to IAM by importing this package.
//
// Run `make generate-go` (or `go generate ./pkg/api`) to regenerate. The
// generated files are committed so importers need no codegen toolchain.
package api

//go:generate go run github.com/ogen-go/ogen/cmd/ogen@latest --config ../../.ogen.yaml --target . --package api --clean ../../openapi/openapi.yaml
