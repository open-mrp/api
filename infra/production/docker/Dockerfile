# Global ARG to be used across stages
ARG SERVICE_NAME

# --- Build Stage ---
# Runs on the build machine's own platform and cross-compiles for the target, so a multi-arch
# build never runs the Go compiler under emulation.
FROM --platform=$BUILDPLATFORM golang:1.27 AS builder
ARG SERVICE_NAME
ARG TARGETOS
ARG TARGETARCH
# RELEASE_VERSION is the release tag (e.g. v4.6.0); agent-service pins its source-code tools to it.
ARG RELEASE_VERSION=""
WORKDIR /app

# Cache dependencies using BuildKit cache mount
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy shared and service code
COPY shared/ ./shared/
COPY services/ ./services/

# Build the service binary with build cache
WORKDIR /app/services/${SERVICE_NAME}/cmd
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build \
      -ldflags "-X github.com/open-mrp/api/shared/buildinfo.Release=${RELEASE_VERSION}" \
      -o /app/main .

# --- Final Stage ---
FROM alpine:latest
ARG SERVICE_NAME
RUN apk --no-cache add ca-certificates
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main ./service

# Handle optional templates: copy the service source and move templates if they exist
COPY services/${SERVICE_NAME}/ ./_tmp_source/
RUN if [ -d ./_tmp_source/templates ]; then mv ./_tmp_source/templates ./templates; fi \
    && rm -rf ./_tmp_source

# Start the service
CMD ["./service"]
