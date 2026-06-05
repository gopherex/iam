# IAM — developer entry point.
# Run `make` (or `make help`) to list targets.

MODULE      := github.com/gopherex/iam
OPENAPI     := openapi/openapi.yaml
# Generated ogen output is module-private (internal/oas); hand-written API
# implementation that consumers import lives in pkg/api.
OAS_DIR     := internal/oas
OAS_PKG     := oas
BIN_DIR     := bin
SERVER_BIN  := $(BIN_DIR)/iam
OGEN        := go run github.com/ogen-go/ogen/cmd/ogen@latest
# ogen consumes OpenAPI 3.0; this is the down-projected build artifact.
OPENAPI_30  := openapi/.build/openapi.3.0.yaml

.DEFAULT_GOAL := help

## help: list available targets
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | awk -F': ' '{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

## tidy: sync go.mod/go.sum
.PHONY: tidy
tidy:
	go mod tidy

## validate: validate the OpenAPI spec (3.1)
.PHONY: validate
validate:
	python -m openapi_spec_validator $(OPENAPI)

## generate: regenerate all code from the spec (Go + TS)
.PHONY: generate
generate: generate-go generate-ts

## generate-go: generate the ogen code (module-private) into internal/oas
.PHONY: generate-go
generate-go:
	@mkdir -p $(dir $(OPENAPI_30))
	python3 scripts/openapi_to_30.py $(OPENAPI) $(OPENAPI_30)
	$(OGEN) --config .ogen.yaml --target $(OAS_DIR) --package $(OAS_PKG) --clean $(OPENAPI_30)

## generate-ts: generate the TypeScript SDK from the spec
.PHONY: generate-ts
generate-ts:
	cd ts && yarn generate

## build: build the server binary (embeds the admin web build)
.PHONY: build
build: build-web
	CGO_ENABLED=0 go build -o $(SERVER_BIN) ./cmd/iam

## build-web: build the admin SPA served by the server
.PHONY: build-web
build-web:
	cd web && yarn install --immutable && yarn build

## run: run the server locally
.PHONY: run
run:
	go run ./cmd/iam

## dev: bring up the full dev infra (docker compose)
.PHONY: dev
dev:
	docker compose up -d

## down: tear down the dev infra
.PHONY: down
down:
	docker compose down

## test: run Go tests
.PHONY: test
test:
	go test ./...

## lint: vet + format check
.PHONY: lint
lint:
	go vet ./...
	gofmt -l .

## fmt: format Go code
.PHONY: fmt
fmt:
	gofmt -w .

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
