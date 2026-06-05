# IAM — developer entry point.
# Run `make` (or `make help`) to list targets.

MODULE      := github.com/gopherex/iam
OPENAPI     := openapi/openapi.yaml
API_PKG_DIR := pkg/api
BIN_DIR     := bin
SERVER_BIN  := $(BIN_DIR)/iam
OGEN        := go run github.com/ogen-go/ogen/cmd/ogen@latest

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

## generate-go: generate the Go API (ogen) into pkg/api
.PHONY: generate-go
generate-go:
	$(OGEN) --config .ogen.yaml --target $(API_PKG_DIR) --package api --clean $(OPENAPI)

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
