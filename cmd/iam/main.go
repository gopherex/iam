// Command iam runs the IAM server: it serves the HTTP API (generated from
// openapi/openapi.yaml) and the embedded admin SPA (built from ../../web).
//
// Stacks (HTTP runtime, storage, config) are intentionally not wired yet.
package main

import (
	"log"
)

func main() {
	// TODO: wire config, storage, the ogen-generated API server (pkg/api),
	// and the embedded admin web build once the stack is fixed.
	log.Println("iam: server entrypoint (not yet wired)")
}
