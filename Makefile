SHELL := bash

# IAM — developer entry point.
# Run `make` (or `make help`) to list targets.

MODULE      := github.com/gopherex/iam
MAX_MAJOR   := 1
OPENAPI     := openapi/openapi.yaml
# Generated ogen output is module-private (internal/oas); hand-written API
# implementation that consumers import lives in pkg/api.
OAS_DIR     := internal/oas
OAS_PKG     := oas
BIN_DIR     := bin
SERVER_BIN  := $(BIN_DIR)/iam
BUILD_PKG   := $(MODULE)/internal/build
SERVICE_NAME ?= iam
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0)
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME  ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_LDFLAGS := -X '$(BUILD_PKG).ServiceName=$(SERVICE_NAME)' -X '$(BUILD_PKG).Version=$(VERSION)' -X '$(BUILD_PKG).Commit=$(COMMIT)' -X '$(BUILD_PKG).BuildTime=$(BUILD_TIME)'
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
	CGO_ENABLED=0 go build -tags embed -ldflags "$(BUILD_LDFLAGS)" -o $(SERVER_BIN) ./cmd/iam

## build-web: build the SDK then the admin SPA into web/dist (embedded by `build`)
.PHONY: build-web
build-web: generate-ts
	cd ts && yarn install --frozen-lockfile && yarn build
	cd web && yarn install --frozen-lockfile && yarn build

## run: run the server locally
.PHONY: run
run:
	go run -ldflags "$(BUILD_LDFLAGS)" ./cmd/iam

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

## release: interactive tag-driven release (Docker image + @gopherex/iam-sdk)
## WARNING: Option 2 (recreate last tag) force-pushes tags by deleting and
## recreating them on the current HEAD. This is intended for LOCAL development
## use only — do NOT use this in CI/CD pipelines. In production, release tags
## MUST be treated as immutable once published.
.PHONY: release
release:
	@set -eu; \
	cd "$$(git rev-parse --show-toplevel)"; \
	if [ -n "$$(git status --porcelain)" ]; then \
	  echo "Working tree is not clean — commit or stash first:"; \
	  git status --short; \
	  exit 1; \
	fi; \
	cur="$$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' | sed 's/^v//' | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)"; \
	cur="$${cur:-0.0.0}"; \
	head="$$(git rev-parse --short HEAD)"; \
	echo "Latest release: v$$cur    HEAD: $$head"; \
	echo; \
	echo "  1) bump version"; \
	echo "  2) recreate last tag (v$$cur) on HEAD   [force]"; \
	echo "  3) cancel"; \
	read -r -p "> " action; \
	case "$$action" in \
	1) \
	  MA="$${cur%%.*}"; rest="$${cur#*.}"; MI="$${rest%%.*}"; PA="$${rest#*.}"; \
	  echo; \
	  echo "  1) major  -> v$$((MA+1)).0.0"; \
	  echo "  2) minor  -> v$$MA.$$((MI+1)).0"; \
	  echo "  3) patch  -> v$$MA.$$MI.$$((PA+1))"; \
	  read -r -p "> " comp; \
	  case "$$comp" in \
	    1) MA=$$((MA+1)); MI=0; PA=0 ;; \
	    2) MI=$$((MI+1)); PA=0 ;; \
	    3) PA=$$((PA+1)) ;; \
	    *) echo "Aborted."; exit 0 ;; \
	  esac; \
	  if [ "$$MA" -gt "$(MAX_MAJOR)" ]; then \
	    echo "v$$MA requires semantic import versioning (/v$$MA in the Go module path)."; \
	    echo "Not supported yet — stay on v0/v1."; \
	    exit 1; \
	  fi; \
	  new="$$MA.$$MI.$$PA"; \
	  echo; \
	  echo "Release v$$new — will:"; \
	  echo "  - set @gopherex/iam-sdk version $$new"; \
	  echo "  - update web/yarn.lock for the local SDK file dependency"; \
	  echo "  - commit 'release v$$new'"; \
	  echo "  - create tag v$$new and push HEAD + tag"; \
	  echo "  - CI will publish ghcr.io/gopherex/iam:$$new + latest and @gopherex/iam-sdk@$$new"; \
	  read -r -p "Type 'yes' to proceed: " ok; \
	  [ "$$ok" = "yes" ] || { echo "Aborted."; exit 0; }; \
	  VERSION="$$new" node -e "const fs=require('fs'); const p='ts/packages/sdk/package.json'; const j=JSON.parse(fs.readFileSync(p,'utf8')); j.version=process.env.VERSION; fs.writeFileSync(p, JSON.stringify(j,null,2)+'\n');"; \
	  (cd web && yarn install >/dev/null); \
	  git add -A; \
	  git diff --cached --quiet || git commit -m "release v$$new"; \
	  git tag -a "v$$new" -m "v$$new"; \
	  git push origin HEAD; \
	  git push origin "v$$new"; \
	  echo "Released v$$new."; \
	  ;; \
	2) \
	  if [ "$$cur" = "0.0.0" ] && ! git tag -l 'v0.0.0' | grep -q .; then \
	    echo "No release tag to recreate."; exit 1; \
	  fi; \
	  pkg_ver="$$(node -p 'require("./ts/packages/sdk/package.json").version')"; \
	  if [ "$$pkg_ver" != "$$cur" ]; then \
	    echo "@gopherex/iam-sdk version $$pkg_ver does not match v$$cur."; \
	    echo "Use bump release instead of recreating the tag."; \
	    exit 1; \
	  fi; \
	  echo; \
	  echo "Will DELETE and recreate tag v$$cur on $$head, then force-push."; \
	  read -r -p "Type 'yes' to proceed: " ok; \
	  [ "$$ok" = "yes" ] || { echo "Aborted."; exit 0; }; \
	  git tag -d "v$$cur" 2>/dev/null || true; \
	  git push origin ":refs/tags/v$$cur" 2>/dev/null || true; \
	  git tag -a "v$$cur" -m "v$$cur"; \
	  git push origin --force "v$$cur"; \
	  echo "Recreated v$$cur on $$head."; \
	  ;; \
	*) \
	  echo "Cancelled."; \
	  ;; \
	esac
