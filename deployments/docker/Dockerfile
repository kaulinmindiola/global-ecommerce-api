# ==============================================================================
# MULTI-STAGE DOCKERFILE - Global E-commerce API
# Strategy: 3-stage build for minimal, secure production image
#
# Stage 1 (deps):    Download and cache Go dependencies
# Stage 2 (builder): Compile the application binary
# Stage 3 (runtime): Minimal Alpine image with binary only
#
# Result: ~15MB production image vs ~1GB development image
# ==============================================================================

# ==============================================================================
# STAGE 1: DEPENDENCY CACHE
# Purpose: Cache dependencies separately to speed up rebuilds
# When dependencies don't change, this layer is reused from cache
# ==============================================================================
ARG GO_VERSION=1.25.10
FROM golang:${GO_VERSION}-alpine AS deps

# Install git (required for some go modules)
RUN apk add --no-cache git

WORKDIR /app

# Copy ONLY dependency files first
# This layer is cached as long as go.mod and go.sum don't change
COPY go.mod go.sum ./

# Download dependencies with verification
# -x: Print executed commands (visible in CI logs)
RUN go mod download -x && \
    go mod verify

# ==============================================================================
# STAGE 2: BUILDER
# Purpose: Compile the Go application into a static binary
# ==============================================================================
FROM golang:${GO_VERSION}-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    ca-certificates \
    tzdata

WORKDIR /app

# Copy cached dependencies from deps stage
COPY --from=deps /go/pkg/mod /go/pkg/mod

# Copy source code
# (After dependencies to maximize cache hits)
COPY . .

# Build arguments for version injection
ARG VERSION=dev
ARG BUILD_DATE
ARG COMMIT_HASH=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Build the binary with optimizations:
# CGO_ENABLED=0 : Static binary, no C dependencies
# -w            : Disable DWARF generation (smaller binary)
# -s            : Disable symbol table (smaller binary)
# -trimpath     : Remove file system paths from binary
# ldflags       : Inject version information at compile time
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -trimpath \
    -ldflags="-w -s \
    -X main.version=${VERSION} \
    -X main.buildDate=${BUILD_DATE} \
    -X main.commitHash=${COMMIT_HASH}" \
    -o /app/bin/ecommerce-api \
    ./cmd/api

# Verify the binary was built successfully
RUN /app/bin/ecommerce-api --version >/dev/null

# Verify binary is truly static
RUN ! ldd /app/bin/ecommerce-api 2>&1 | grep -q "libc"

# ==============================================================================
# STAGE 3: SECURITY HARDENING LAYER
# Purpose: Prepare minimal filesystem for runtime
# ==============================================================================
FROM alpine:3.19 AS security

# Install security tools
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget

# Create non-root user and group
# Using fixed UID/GID for consistency across environments
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup -h /app -s /bin/sh

# Create necessary directories with correct permissions
RUN mkdir -p /app/config /app/logs && \
    chown -R appuser:appgroup /app

# ==============================================================================
# STAGE 4: RUNTIME (FINAL IMAGE)
# Purpose: Minimal production image with only what's needed
# ==============================================================================
FROM scratch AS runtime

# Copy CA certificates for HTTPS calls (e.g., external currency APIs)
COPY --from=security /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data for UTC/local time operations
COPY --from=security /usr/share/zoneinfo /usr/share/zoneinfo

# Copy passwd/group for non-root user
COPY --from=security /etc/passwd /etc/passwd
COPY --from=security /etc/group /etc/group

# Copy the compiled binary
COPY --from=builder /app/bin/ecommerce-api /app/ecommerce-api

# Copy default config
COPY --from=builder /app/config /app/config

# Set working directory
WORKDIR /app

# Run as non-root user (security best practice)
USER appuser:appgroup

# Expose API port
EXPOSE 8080

# Health check - verify API responds correctly
# --interval=30s : Check every 30 seconds
# --timeout=10s  : Give 10 seconds to respond
# --start-period=5s : Wait 5 seconds before first check (startup grace)
# --retries=3    : Mark unhealthy after 3 consecutive failures
HEALTHCHECK \
    --interval=30s \
    --timeout=10s \
    --start-period=5s \
    --retries=3 \
    CMD ["/app/ecommerce-api", "healthcheck"]

# Set environment defaults
ENV APP_ENV=production \
    APP_PORT=8080 \
    LOG_FORMAT=json \
    LOG_LEVEL=info \
    TZ=UTC

# Labels for image metadata (OCI spec)
LABEL org.opencontainers.image.title="Global E-commerce API" \
    org.opencontainers.image.description="Multi-currency e-commerce REST API" \
    org.opencontainers.image.vendor="Kaulin Mindiola" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.source="https://github.com/kaulinmindiola/global-ecommerce-api" \
    org.opencontainers.image.documentation="https://github.com/kaulinmindiola/global-ecommerce-api/blob/main/README.md"

# Run the binary
ENTRYPOINT ["/app/ecommerce-api"]
CMD ["serve"]
