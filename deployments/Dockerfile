# Multi-stage build for the IAM server: build the TS SDK + admin SPA, then the
# Go binary with the SPA embedded (-tags embed), into a minimal static image.
#
# Build from the repository root:
#   docker build -f deployments/Dockerfile -t iam .

# --- 1. admin SPA (and the SDK it consumes) ---
FROM node:22-alpine AS web
WORKDIR /src
# Generate + build the SDK first (the SPA depends on it via a file: link).
COPY ts ./ts
COPY openapi/openapi.yaml ./openapi/openapi.yaml
RUN cd ts && yarn install --frozen-lockfile && yarn generate && yarn build
# Build the SPA -> web/dist.
COPY web ./web
RUN cd web && yarn install --frozen-lockfile && yarn build

# --- 2. Go binary (embeds web/dist) ---
FROM golang:1.26-alpine AS go
WORKDIR /src
RUN apk add --no-cache git
ARG SERVICE_NAME=iam
ARG VERSION=0.0.0
ARG COMMIT=unknown
ARG BUILD_TIME=
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN build_time="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}" && \
    CGO_ENABLED=0 go build -tags embed -trimpath \
    -ldflags="-s -w -X github.com/gopherex/iam/internal/build.ServiceName=${SERVICE_NAME} -X github.com/gopherex/iam/internal/build.Version=${VERSION} -X github.com/gopherex/iam/internal/build.Commit=${COMMIT} -X github.com/gopherex/iam/internal/build.BuildTime=${build_time}" \
    -o /iam ./cmd/iam

# --- 3. runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go /iam /iam
# 8080 = API + admin SPA, 8081 = liveness/readiness probes.
EXPOSE 8080 8081
USER nonroot:nonroot
ENTRYPOINT ["/iam"]
