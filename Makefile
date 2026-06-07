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
# Postgres store codegen (sqld toolchain): reads schema.sql + queries/*.sql,
# writes gen/db + gen/bob and migrations.
SQLD_CFG    := internal/infrastructure/postgres/sqld.yaml
SQLD_MIGR   := internal/infrastructure/postgres/migrations
# DB-backed test wiring (auto-starts the compose postgres on :5436).
POSTGRES_PASSWORD ?= $(shell grep -E '^POSTGRES_PASSWORD=' .env 2>/dev/null | cut -d= -f2)
POSTGRES_PASSWORD := $(if $(POSTGRES_PASSWORD),$(POSTGRES_PASSWORD),iam)
TEST_DATABASE_URL ?= postgres://iam:$(POSTGRES_PASSWORD)@127.0.0.1:5436/iam?sslmode=disable

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

## generate: regenerate all code (API + DB store)
.PHONY: generate
generate: generate-go generate-ts db-gen

## tools: install the sqld code generators into ./bin
.PHONY: tools
tools:
	GOFLAGS=-mod=mod GOBIN=$(CURDIR)/bin go install github.com/gopherex/sqld/cmd/sqld@v1.0.0
	GOFLAGS=-mod=mod GOBIN=$(CURDIR)/bin go install github.com/gopherex/sqld/cmd/sqld-gen-go@v1.0.0
	GOFLAGS=-mod=mod GOBIN=$(CURDIR)/bin go install github.com/gopherex/sqld/cmd/sqld-gen-bob@v1.0.0

## db-gen: generate gen/db + gen/bob from schema.sql + queries/*.sql
.PHONY: db-gen
db-gen: tools
	./bin/sqld generate -c $(SQLD_CFG)

## migrate-generate: generate a migration by schema diff (usage: make migrate-generate name=add_table)
.PHONY: migrate-generate
migrate-generate: tools
	@test -n "$(name)" || (echo "usage: make migrate-generate name=add_table" && exit 2)
	./bin/sqld migrate generate $(name) -c $(SQLD_CFG)

## migrate-clear: regenerate the single bootstrap migration from schema.sql, then regenerate code
.PHONY: migrate-clear
migrate-clear: tools
	rm -f $(SQLD_MIGR)/*.sql
	./bin/sqld migrate generate bootstrap -c $(SQLD_CFG)
	@for f in $(SQLD_MIGR)/*.sql; do \
		awk 'index(tolower($$0), "-- sqld:" "up") == 1 { next } index(tolower($$0), "-- sqld:" "down") == 1 { exit } { print }' "$$f" > "$$f.tmp"; \
		mv "$$f.tmp" "$$f"; \
	done
	$(MAKE) db-gen

## generate-go: generate the ogen code (module-private) into internal/oas
.PHONY: generate-go
generate-go:
	@mkdir -p $(dir $(OPENAPI_30))
	python3 scripts/openapi_to_30.py $(OPENAPI) $(OPENAPI_30)
	$(OGEN) --config .ogen.yaml --target $(OAS_DIR) --package $(OAS_PKG) --clean $(OPENAPI_30)

## generate-ts: generate the TypeScript SDK from the spec (@hey-api/openapi-ts)
.PHONY: generate-ts
generate-ts:
	cd ts && yarn install --frozen-lockfile && yarn generate

## build: build the server binary (embeds the admin web build via -tags embed)
.PHONY: build
build: build-web
	CGO_ENABLED=0 go build -tags embed -o $(SERVER_BIN) ./cmd/iam

## build-web: build the SDK then the admin SPA into web/dist (embedded by `build`)
.PHONY: build-web
build-web:
	cd ts && yarn install --frozen-lockfile && yarn build
	cd web && yarn install --frozen-lockfile && yarn build

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

## test-db: run integration tests (testcontainers spins up throwaway Postgres; needs Docker)
.PHONY: test-db
test-db:
	go test -tags=integration ./... -count=1 -p 1

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
